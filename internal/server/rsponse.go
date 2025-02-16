package server

// DetailedResponse represents detailed price response
type DetailedResponse struct {
	Pair   string  `json:"pair"`
	Price  float64 `json:"price"`
	Source string  `json:"source"`
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
