package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ivanglie/coinmon/internal/exchange"
	"github.com/ivanglie/coinmon/pkg/log"
)

// DetailedResponse represents detailed price response
type DetailedResponse struct {
	Pair   string  `json:"pair"`
	Price  float64 `json:"price"`
	Source string  `json:"source"`
}

type httpServer interface {
	ListenAndServe() error
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Server handles HTTP requests to exchanges
type Server struct {
	exchanges []*exchange.Exchange
	listener  httpServer
	client    httpClient
}

// New creates a new server instance
func New(addr string) *Server {
	exchanges := []*exchange.Exchange{
		exchange.New(exchange.BINANCE),
		exchange.New(exchange.BYBIT),
		exchange.New(exchange.BITGET),
	}

	s := &Server{
		exchanges: exchanges,
		listener: &http.Server{
			Addr:         addr,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	http.HandleFunc("/api/v1/spot/", s.HandleSpot)

	return s
}

// Start starts the server
func (s *Server) Start() error {
	return s.listener.ListenAndServe()
}

// HandleSpot handles /api/v1/spot/{pair} requests
func (s *Server) HandleSpot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pair := strings.TrimPrefix(r.URL.Path, "/api/v1/spot/")
	if pair == "" {
		http.Error(w, "Missing trading pair", http.StatusBadRequest)
		return
	}
	pair = strings.ToUpper(pair)

	isDetailed := r.URL.Query().Get("details") == "true"

	price, source, err := s.firstPriceWithDetails(r.Context(), pair)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	if isDetailed {
		w.Header().Set("Content-Type", "application/json")
		response := DetailedResponse{Pair: pair, Price: price, Source: source}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("Failed to encode response: " + err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		w.Header().Set("Content-Type", "text/plain")
		if _, err := fmt.Fprintf(w, "%f", price); err != nil {
			log.Error("Failed to write response: " + err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}

func (s *Server) firstPriceWithDetails(ctx context.Context, pair string) (price float64, source string, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		price  float64
		source string
		err    error
	}

	type errorResponse struct {
		Message string   `json:"message"`
		Errors  []string `json:"errors"`
	}

	results := make(chan result, len(s.exchanges))

	for _, ex := range s.exchanges {
		go func(ex *exchange.Exchange) {
			p, e := s.fetchPrice(ctx, ex, pair)
			select {
			case <-ctx.Done():
				return
			case results <- result{p, ex.Name.String(), e}:
			}
		}(ex)
	}

	var errors []string
	for i := 0; i < len(s.exchanges); i++ {
		r := <-results
		if r.err != nil {
			errMsg := fmt.Sprintf("%s: %v", r.source, r.err)
			log.Error("Error from " + errMsg)
			errors = append(errors, errMsg)
			continue
		}

		log.Info(fmt.Sprintf("Got price %.2f from %s", r.price, r.source))
		cancel()
		return r.price, r.source, nil
	}

	errResp := errorResponse{
		Message: "all exchanges failed",
		Errors:  errors,
	}

	b, err := json.Marshal(errResp)
	if err != nil {
		return 0, "", fmt.Errorf("failed to marshal error response: %v", err)
	}

	log.Error(string(b))
	return 0, "", fmt.Errorf("%s", string(b))
}

func (s *Server) fetchPrice(ctx context.Context, e *exchange.Exchange, pair string) (float64, error) {
	url := e.PriceURL(pair)
	log.Info(fmt.Sprintf("Requesting %s price for %s: %s", e.Name, pair, url))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("do request: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		switch e.Name {
		case exchange.BINANCE:
			var r exchange.BinanceErrorResponse
			if err := json.Unmarshal(body, &r); err != nil {
				return 0, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
			}

			return 0, fmt.Errorf("code=%d, msg=%s", r.Code, r.Msg)
		case exchange.BYBIT:
			var r exchange.BybitResponse
			if err := json.Unmarshal(body, &r); err != nil {
				return 0, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
			}

			return 0, fmt.Errorf("code=%d, msg=%s", r.RetCode, r.RetMsg)
		case exchange.BITGET:
			var r exchange.BitgetResponse
			if err := json.Unmarshal(body, &r); err != nil {
				return 0, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
			}

			return 0, fmt.Errorf("code=%s, msg=%s", r.Code, r.Msg)
		}

		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))

	switch e.Name {
	case exchange.BINANCE:
		var r exchange.BinanceResponse
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return 0, fmt.Errorf("decode response: %w", err)
		}

		price, err := strconv.ParseFloat(r.Price, 64)
		if err != nil {
			return 0, fmt.Errorf("parse price: %w", err)
		}

		return price, nil
	case exchange.BYBIT:
		var r exchange.BybitResponse
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return 0, fmt.Errorf("decode response: %w", err)
		}

		if len(r.Result.List) == 0 {
			return 0, fmt.Errorf("empty response")
		}

		price, err := strconv.ParseFloat(r.Result.List[0].LastPrice, 64)
		if err != nil {
			return 0, fmt.Errorf("parse price: %w", err)
		}

		return price, nil
	case exchange.BITGET:
		var r exchange.BitgetResponse
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return 0, fmt.Errorf("decode response: %w", err)
		}

		if len(r.Data) == 0 {
			return 0, fmt.Errorf("empty response")
		}

		price, err := strconv.ParseFloat(r.Data[0].LastPr, 64)
		if err != nil {
			return 0, fmt.Errorf("parse price: %w", err)
		}

		return price, nil
	}

	return 0, fmt.Errorf("unknown exchange")
}
