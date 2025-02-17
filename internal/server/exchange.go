package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/ivanglie/coinmon/internal/exchange"
	"github.com/ivanglie/coinmon/pkg/log"
)

func (s *Server) firstPriceWithDetails(ctx context.Context, pair string) (float64, string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		price float64
		err   error
		ex    *exchange.Exchange
	}

	results := make(chan result, len(s.exchanges))

	for _, e := range s.exchanges {
		go func(e *exchange.Exchange) {
			price, err := s.fetchPrice(ctx, e, pair)
			select {
			case <-ctx.Done():
				return
			case results <- result{price, err, e}:
			}
		}(e)
	}

	var lastErr error
	for i := 0; i < len(s.exchanges); i++ {
		result := <-results
		if result.err != nil {
			log.Error(fmt.Sprintf("Error from %s: %v", result.ex.Name, result.err))
			lastErr = result.err
			continue
		}

		log.Info(fmt.Sprintf("Got price from %s", result.ex.Name))
		cancel()
		return result.price, result.ex.Name.String(), nil
	}

	log.Error("All exchanges failed")
	return 0, "", fmt.Errorf("all exchanges failed. Last error: %v", lastErr)
}

func (s *Server) fetchPrice(ctx context.Context, e *exchange.Exchange, pair string) (float64, error) {
	url := e.PriceURL(pair)
	log.Info(fmt.Sprintf("Requesting %s price for %s: %s", e.Name, pair, url))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
			var errResp exchange.BinanceErrorResponse
			if err := json.Unmarshal(body, &errResp); err != nil {
				return 0, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
			}
			return 0, fmt.Errorf("binance error: code=%d, msg=%s", errResp.Code, errResp.Msg)

		case exchange.BYBIT:
			var errResp exchange.BybitErrorResponse
			if err := json.Unmarshal(body, &errResp); err != nil {
				return 0, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
			}
			return 0, fmt.Errorf("bybit error: code=%d, msg=%s", errResp.RetCode, errResp.RetMsg)

		case exchange.BITGET:
			var errResp exchange.BitgetErrorResponse
			if err := json.Unmarshal(body, &errResp); err != nil {
				return 0, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
			}
			return 0, fmt.Errorf("bitget error: code=%s, msg=%s", errResp.Code, errResp.Title)
		}
		return 0, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))

	switch e.Name {
	case exchange.BINANCE:
		var binanceResp exchange.BinanceResponse
		if err := json.NewDecoder(resp.Body).Decode(&binanceResp); err != nil {
			return 0, fmt.Errorf("decode response: %w", err)
		}
		price, err := strconv.ParseFloat(binanceResp.Price, 64)
		if err != nil {
			return 0, fmt.Errorf("parse price: %w", err)
		}
		return price, nil

	case exchange.BYBIT:
		var bybitResp exchange.BybitResponse
		if err := json.NewDecoder(resp.Body).Decode(&bybitResp); err != nil {
			return 0, fmt.Errorf("decode response: %w", err)
		}
		if len(bybitResp.Result.List) == 0 {
			return 0, fmt.Errorf("empty response")
		}
		price, err := strconv.ParseFloat(bybitResp.Result.List[0].LastPrice, 64)
		if err != nil {
			return 0, fmt.Errorf("parse price: %w", err)
		}
		return price, nil

	case exchange.BITGET:
		var bitgetResp exchange.BitgetResponse
		if err := json.NewDecoder(resp.Body).Decode(&bitgetResp); err != nil {
			return 0, fmt.Errorf("decode response: %w", err)
		}
		if len(bitgetResp.Data) == 0 {
			return 0, fmt.Errorf("empty response")
		}
		price, err := strconv.ParseFloat(bitgetResp.Data[0].LastPr, 64)
		if err != nil {
			return 0, fmt.Errorf("parse price: %w", err)
		}
		return price, nil
	}

	return 0, fmt.Errorf("unknown exchange")
}
