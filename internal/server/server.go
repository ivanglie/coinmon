package server

import (
	"encoding/json"
	"fmt"
	"net/http"
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

// Server handles HTTP requests to exchanges
type Server struct {
	exchanges []*exchange.Exchange
	srv       *http.Server
	client    *http.Client
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
		srv: &http.Server{
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
	return s.srv.ListenAndServe()
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
