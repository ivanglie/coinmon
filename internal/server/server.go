package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ivanglie/coinmon/internal/exchange"
)

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

	price, err := s.firstPrice(r.Context(), pair)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(price)
}

// HandleSpotDetails handles /api/v1/spot/{pair}/details requests
func (s *Server) HandleSpotDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pair := strings.TrimPrefix(r.URL.Path, "/api/v1/spot/")
	pair = strings.TrimSuffix(pair, "/details")
	if pair == "" {
		http.Error(w, "Missing trading pair", http.StatusBadRequest)
		return
	}

	price, source, err := s.firstPriceWithDetails(r.Context(), pair)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	response := DetailedResponse{
		Pair:   pair,
		Price:  price,
		Source: source,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Routes sets up the server routes
func (s *Server) Routes() {
	http.HandleFunc("/api/v1/spot/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/details") {
			s.HandleSpotDetails(w, r)
			return
		}
		s.HandleSpot(w, r)
	})
}
