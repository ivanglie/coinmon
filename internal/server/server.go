package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ivanglie/coinmon/internal/exchange"
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
	client    *http.Client
}

// New creates a new server instance
func New() *Server {
	exchanges := []*exchange.Exchange{
		exchange.New(exchange.BINANCE),
		exchange.New(exchange.BYBIT),
		exchange.New(exchange.BITGET),
	}

	return &Server{
		exchanges: exchanges,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
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
		response := DetailedResponse{
			Pair:   pair,
			Price:  price,
			Source: source,
		}
		json.NewEncoder(w).Encode(response)
	} else {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%f", price)
	}
}
