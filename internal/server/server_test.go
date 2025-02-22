package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ivanglie/coinmon/internal/exchange"
	"github.com/stretchr/testify/assert"
)

type mockHttpServer struct{}

func (m *mockHttpServer) ListenAndServe() error {
	return nil
}

type mockHttpClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

type mockResponseFunc func(req *http.Request) (*http.Response, error)

var (
	server = &Server{}

	exchanges = []*exchange.Exchange{
		exchange.New(exchange.BINANCE),
		exchange.New(exchange.BYBIT),
		exchange.New(exchange.BITGET),
	}
)

func setupTest() {
	server = New(":8080")
	server.exchanges = exchanges
	server.srv = &mockHttpServer{}
	server.client = &mockHttpClient{}
}

func teardownTest() {
	server = nil
}

func mockSuccessfulResponse(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
	}

	switch {
	case strings.Contains(req.URL.String(), "binance"):
		binanceResponse := exchange.BinanceResponse{
			Symbol: "BTCUSDT",
			Price:  "99999.99",
		}
		return mockJSONResponse(resp, binanceResponse)

	case strings.Contains(req.URL.String(), "bybit"):
		bybitResponse := exchange.BybitResponse{
			RetCode: 0,
			RetMsg:  "OK",
			Result: struct {
				Category string `json:"category"`
				List     []struct {
					Symbol    string `json:"symbol"`
					LastPrice string `json:"lastPrice"`
				} `json:"list"`
			}{
				Category: "spot",
				List: []struct {
					Symbol    string `json:"symbol"`
					LastPrice string `json:"lastPrice"`
				}{
					{
						Symbol:    "BTCUSDT",
						LastPrice: "99999.98",
					},
				},
			},
		}
		return mockJSONResponse(resp, bybitResponse)

	case strings.Contains(req.URL.String(), "bitget"):
		bitgetResponse := exchange.BitgetResponse{
			Code: "00000",
			Msg:  "success",
			Data: []struct {
				Symbol string `json:"symbol"`
				LastPr string `json:"lastPr"`
			}{
				{
					Symbol: "BTCUSDT",
					LastPr: "99999.97",
				},
			},
		}
		return mockJSONResponse(resp, bitgetResponse)

	default:
		return nil, fmt.Errorf("unknown exchange in URL: %s", req.URL.String())
	}
}

func mockErrorResponse(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
	}

	switch {
	case strings.Contains(req.URL.String(), "binance"):
		binanceResponse := exchange.BinanceErrorResponse{
			Code: 400,
			Msg:  "Bad Request",
		}
		return mockJSONResponse(resp, binanceResponse)

	case strings.Contains(req.URL.String(), "bybit"):
		bybitResponse := exchange.BybitResponse{
			RetCode: 400,
			RetMsg:  "Bad Request",
		}
		return mockJSONResponse(resp, bybitResponse)

	case strings.Contains(req.URL.String(), "bitget"):
		bitgetResponse := exchange.BitgetResponse{
			Code: "400",
			Msg:  "Bad Request",
		}
		return mockJSONResponse(resp, bitgetResponse)

	default:
		return nil, fmt.Errorf("unknown exchange in URL: %s", req.URL.String())
	}
}

func mockInvalidPairResponse(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
	}

	switch {
	case strings.Contains(req.URL.String(), "binance"):
		binanceResponse := exchange.BinanceErrorResponse{
			Code: -1100,
			Msg:  "Illegal characters found in parameter 'symbol'; legal range is '^[A-Z0-9_.]{1,20}$'.",
		}
		return mockJSONResponse(resp, binanceResponse)

	case strings.Contains(req.URL.String(), "bybit"):
		bybitResponse := exchange.BybitResponse{
			RetCode: 10001,
			RetMsg:  "Not supported symbols",
		}
		return mockJSONResponse(resp, bybitResponse)

	case strings.Contains(req.URL.String(), "bitget"):
		bitgetResponse := exchange.BitgetResponse{
			Code: "40034",
			Msg:  "Parameter does not exist",
		}
		return mockJSONResponse(resp, bitgetResponse)

	default:
		return nil, fmt.Errorf("unknown exchange in URL: %s", req.URL.String())
	}
}

func mockEmptyPairResponse(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
	}

	switch {
	case strings.Contains(req.URL.String(), "binance"):
		binanceResponse := exchange.BinanceErrorResponse{
			Code: -1105,
			Msg:  "Parameter 'symbol' was empty.",
		}
		return mockJSONResponse(resp, binanceResponse)

	case strings.Contains(req.URL.String(), "bybit"):
		bybitResponse := exchange.BybitResponse{
			RetCode: 10001,
			RetMsg:  "Not supported symbols",
		}
		return mockJSONResponse(resp, bybitResponse)

	case strings.Contains(req.URL.String(), "bitget"):
		bitgetResponse := exchange.BitgetResponse{
			Code: "40034",
			Msg:  "Parameter does not exist",
		}
		return mockJSONResponse(resp, bitgetResponse)

	default:
		return nil, fmt.Errorf("unknown exchange in URL: %s", req.URL.String())
	}
}

func mockJSONResponse(resp *http.Response, data interface{}) (*http.Response, error) {
	jsonResponse, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	resp.Body = io.NopCloser(bytes.NewReader(jsonResponse))
	return resp, nil
}

func TestServer_fetchPrice(t *testing.T) {
	setupTest()
	defer teardownTest()

	tests := []struct {
		name          string
		exchange      *exchange.Exchange
		pair          string
		mockResponse  mockResponseFunc
		expectedPrice float64
		expectError   bool
	}{
		{
			name:          "binance success",
			exchange:      exchanges[0],
			pair:          "BTCUSDT",
			mockResponse:  mockSuccessfulResponse,
			expectedPrice: 99999.99,
		},
		{
			name:          "bybit success",
			exchange:      exchanges[1],
			pair:          "BTCUSDT",
			mockResponse:  mockSuccessfulResponse,
			expectedPrice: 99999.98,
		},
		{
			name:          "bitget success",
			exchange:      exchanges[2],
			pair:          "BTCUSDT",
			mockResponse:  mockSuccessfulResponse,
			expectedPrice: 99999.97,
		},
		{
			name:         "binance error",
			exchange:     exchanges[0],
			pair:         "BTCUSDT",
			mockResponse: mockErrorResponse,
			expectError:  true,
		},
		{
			name:         "bybit error",
			exchange:     exchanges[1],
			pair:         "BTCUSDT",
			mockResponse: mockErrorResponse,
			expectError:  true,
		},
		{
			name:         "bitget error",
			exchange:     exchanges[2],
			pair:         "BTCUSDT",
			mockResponse: mockErrorResponse,
			expectError:  true,
		},
		{
			name:         "binance invalid pair",
			exchange:     exchanges[0],
			pair:         "INVALID",
			mockResponse: mockInvalidPairResponse,
			expectError:  true,
		},
		{
			name:         "bybit invalid pair",
			exchange:     exchanges[1],
			pair:         "INVALID",
			mockResponse: mockInvalidPairResponse,
			expectError:  true,
		},
		{
			name:         "bitget invalid pair",
			exchange:     exchanges[2],
			pair:         "INVALID",
			mockResponse: mockInvalidPairResponse,
			expectError:  true,
		},
		{
			name:         "binance empty pair",
			exchange:     exchanges[0],
			pair:         "",
			mockResponse: mockEmptyPairResponse,
			expectError:  true,
		},
		{
			name:         "bybit empty pair",
			exchange:     exchanges[1],
			pair:         "",
			mockResponse: mockEmptyPairResponse,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			server.client = &mockHttpClient{
				doFunc: tt.mockResponse,
			}

			price, err := server.fetchPrice(ctx, tt.exchange, tt.pair)
			if tt.expectError {
				t.Log(tt.exchange.Name, tt.pair, err)
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPrice, price)
		})
	}
}
