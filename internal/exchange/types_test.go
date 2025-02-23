package exchange

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestName_String(t *testing.T) {
	tests := []struct {
		name     Name
		expected string
	}{
		{
			name:     BINANCE,
			expected: "binance",
		},
		{
			name:     BYBIT,
			expected: "bybit",
		},
		{
			name:     BITGET,
			expected: "bitget",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.name.String())
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name         Name
		expectedURL  string
		expectedPath string
	}{
		{
			name:         BINANCE,
			expectedURL:  "https://api.binance.com",
			expectedPath: "api/v3/ticker/price",
		},
		{
			name:         BYBIT,
			expectedURL:  "https://api.bybit.com",
			expectedPath: "v5/market/tickers",
		},
		{
			name:         BITGET,
			expectedURL:  "https://api.bitget.com",
			expectedPath: "api/v2/spot/market/tickers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name.String(), func(t *testing.T) {
			e := New(tt.name)
			assert.Equal(t, tt.name, e.Name)
			assert.Equal(t, tt.expectedURL, e.BaseURL)
			assert.Equal(t, tt.expectedPath, e.PricePath)
		})
	}
}

func TestExchange_PriceURL(t *testing.T) {
	tests := []struct {
		name        string
		exchange    *Exchange
		pair        string
		expectedURL string
	}{
		{
			name:        "binance price url",
			exchange:    New(BINANCE),
			pair:        "BTCUSDT",
			expectedURL: "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT",
		},
		{
			name:        "bybit price url",
			exchange:    New(BYBIT),
			pair:        "BTCUSDT",
			expectedURL: "https://api.bybit.com/v5/market/tickers?category=spot&symbol=BTCUSDT",
		},
		{
			name:        "bitget price url",
			exchange:    New(BITGET),
			pair:        "BTCUSDT",
			expectedURL: "https://api.bitget.com/api/v2/spot/market/tickers?symbol=BTCUSDT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedURL, tt.exchange.PriceURL(tt.pair))
		})
	}
}
