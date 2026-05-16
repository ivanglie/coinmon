// Package exchange provides types and utilities for cryptocurrency exchange integrations.
package exchange

import "fmt"

// Name represents supported cryptocurrency exchanges
type Name int

// Exchange names
const (
	BINANCE Name = iota
	BYBIT
	BITGET
	KRAKEN
)

var names = [...]string{
	BINANCE: "binance",
	BYBIT:   "bybit",
	BITGET:  "bitget",
	KRAKEN:  "kraken",
}

// String returns exchange name
func (n Name) String() string {
	return names[n]
}

// Exchange represents a cryptocurrency exchange with its configuration
type Exchange struct {
	Name      Name
	BaseURL   string
	PricePath string
}

// BinanceResponse represents Binance API response
type BinanceResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

// BinanceErrorResponse represents Binance error response
type BinanceErrorResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// BybitResponse represents Bybit API response
type BybitResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string `json:"category"`
		List     []struct {
			Symbol    string `json:"symbol"`
			LastPrice string `json:"lastPrice"`
		} `json:"list"`
	} `json:"result"`
}

// BitgetResponse represents Bitget API response
type BitgetResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		Symbol string `json:"symbol"`
		LastPr string `json:"lastPr"`
	} `json:"data"`
}

// KrakenResponse represents Kraken API response
type KrakenResponse struct {
	Error  []string `json:"error"`
	Result map[string]struct {
		C [2]string `json:"c"` // last trade: [price, lot_volume]
	} `json:"result"`
}

func baseURLs() map[Name]string {
	return map[Name]string{
		BINANCE: "https://api.binance.com",
		BYBIT:   "https://api.bybit.com",
		BITGET:  "https://api.bitget.com",
		KRAKEN:  "https://api.kraken.com",
	}
}

func pricePaths() map[Name]string {
	return map[Name]string{
		BINANCE: "api/v3/ticker/price",
		BYBIT:   "v5/market/tickers",
		BITGET:  "api/v2/spot/market/tickers",
		KRAKEN:  "0/public/Ticker",
	}
}

// New creates a new Exchange instance with default configuration
func New(name Name) *Exchange {
	return &Exchange{
		Name:      name,
		BaseURL:   baseURLs()[name],
		PricePath: pricePaths()[name],
	}
}

// PriceURL returns complete URL for price request
func (e *Exchange) PriceURL(pair string) string {
	switch e.Name {
	case BYBIT:
		return fmt.Sprintf("%s/%s?category=spot&symbol=%s", e.BaseURL, e.PricePath, pair)
	case KRAKEN:
		return fmt.Sprintf("%s/%s?pair=%s", e.BaseURL, e.PricePath, pair)
	default:
		return fmt.Sprintf("%s/%s?symbol=%s", e.BaseURL, e.PricePath, pair)
	}
}
