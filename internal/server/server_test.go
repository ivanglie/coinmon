package server

import (
	"bytes"
	"context"
	"encoding/json"
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

func setSuccessResponses() {
	server.client = &mockHttpClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			resp := &http.Response{}
			resp.StatusCode = 200

			switch {
			case strings.Contains(req.URL.String(), "binance"):
				binanceResponse := exchange.BinanceResponse{
					Symbol: "BTCUSDT",
					Price:  "99999.99",
				}

				jsonResponse, err := json.Marshal(binanceResponse)
				if err != nil {
					return resp, err
				}

				resp.Body = io.NopCloser(bytes.NewReader(jsonResponse))
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

				jsonResponse, err := json.Marshal(bybitResponse)
				if err != nil {
					return resp, err
				}

				resp.Body = io.NopCloser(bytes.NewReader(jsonResponse))
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

				jsonResponse, err := json.Marshal(bitgetResponse)
				if err != nil {
					return resp, err
				}

				resp.Body = io.NopCloser(bytes.NewReader(jsonResponse))
			}

			return resp, nil
		},
	}
}

func setErrorResponses() {
	server.client = &mockHttpClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			resp := &http.Response{}
			resp.StatusCode = 400

			switch {
			case strings.Contains(req.URL.String(), "binance"):
				binanceErrorResponse := exchange.BinanceErrorResponse{Code: 400, Msg: "Bad Request"}
				jsonResponse, err := json.Marshal(binanceErrorResponse)
				if err != nil {
					return resp, err
				}

				resp.Body = io.NopCloser(bytes.NewReader(jsonResponse))
			case strings.Contains(req.URL.String(), "bybit"):
				bybitErrorResponse := exchange.BybitResponse{RetCode: 400, RetMsg: "Bad Request"}
				jsonResponse, err := json.Marshal(bybitErrorResponse)
				if err != nil {
					return resp, err
				}

				resp.Body = io.NopCloser(bytes.NewReader(jsonResponse))
			case strings.Contains(req.URL.String(), "bitget"):
				bitgetErrorResponse := exchange.BitgetResponse{Code: "400", Msg: "Bad Request"}
				jsonResponse, err := json.Marshal(bitgetErrorResponse)
				if err != nil {
					return resp, err
				}

				resp.Body = io.NopCloser(bytes.NewReader(jsonResponse))
			}

			return resp, nil
		},
	}
}

func TestServer_fetchPrice(t *testing.T) {
	setupTest()

	ctx := context.Background()

	binance := exchanges[0]
	bybit := exchanges[1]
	bitget := exchanges[2]

	setSuccessResponses()

	p, err := server.fetchPrice(ctx, binance, "BTCUSDT")
	// t.Log(binance.Name, p, err)
	assert.NoError(t, err)
	assert.Equal(t, 99999.99, p)

	p, err = server.fetchPrice(ctx, bybit, "BTCUSDT")
	assert.NoError(t, err)
	assert.Equal(t, 99999.98, p)

	p, err = server.fetchPrice(ctx, bitget, "BTCUSDT")
	assert.NoError(t, err)
	assert.Equal(t, 99999.97, p)

	setErrorResponses()

	_, err = server.fetchPrice(ctx, binance, "BTCUSDT")
	assert.Error(t, err)

	_, err = server.fetchPrice(ctx, bybit, "BTCUSDT")
	assert.Error(t, err)

	_, err = server.fetchPrice(ctx, bitget, "BTCUSDT")
	assert.Error(t, err)

	teardownTest()
}
