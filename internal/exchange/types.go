package exchange

import "fmt"

// Name represents supported cryptocurrency exchanges
type Name int

const (
	BINANCE Name = iota
	BYBIT
	BITGET
)

// String returns string representation of exchange name
func (n Name) String() string {
	return [...]string{"binance", "bybit", "bitget"}[n]
}

// BinanceResponse represents Binance API response
type BinanceResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
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

// Exchange represents a cryptocurrency exchange with its configuration
type Exchange struct {
	Name      Name
	BaseURL   string
	PricePath string
}

func baseURLs() map[Name]string {
	return map[Name]string{
		BINANCE: "https://api.binance.com",
		BYBIT:   "https://api.bybit.com",
		BITGET:  "https://api.bitget.com",
	}
}

func pricePaths() map[Name]string {
	return map[Name]string{
		BINANCE: "api/v3/ticker/price",
		BYBIT:   "v5/market/tickers",
		BITGET:  "api/v2/spot/market/tickers",
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
	if e.Name == BYBIT {
		return fmt.Sprintf("%s/%s?category=spot&symbol=%s", e.BaseURL, e.PricePath, pair)
	}
	return fmt.Sprintf("%s/%s?symbol=%s", e.BaseURL, e.PricePath, pair)
}
