package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ivanglie/coinmon/internal/exchange"
	"github.com/stretchr/testify/assert"
)

type mockHttpServer struct {
	listenAndServeFunc func() error
}

func (m *mockHttpServer) ListenAndServe() error {
	return m.listenAndServeFunc()
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
	server.listener = &mockHttpServer{}
	server.client = &mockHttpClient{}
}

func teardownTest() {
	server.exchanges = nil
	server.listener = nil
	server.client = nil
	server = nil
}

type mockResponseFunc func(req *http.Request) (*http.Response, error)

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

func mockSuccessfulResponseWithDelay(delays map[string]time.Duration) mockResponseFunc {
	return func(req *http.Request) (*http.Response, error) {
		var exchange string
		switch {
		case strings.Contains(req.URL.String(), "binance"):
			exchange = "binance"
		case strings.Contains(req.URL.String(), "bybit"):
			exchange = "bybit"
		case strings.Contains(req.URL.String(), "bitget"):
			exchange = "bitget"
		}

		if delay, ok := delays[exchange]; ok {
			time.Sleep(delay)
		}

		return mockSuccessfulResponse(req)
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

func TestServer_Start(t *testing.T) {
	tests := []struct {
		name        string
		serverError error
		expectError bool
	}{
		{
			name:        "server starts successfully",
			serverError: nil,
			expectError: false,
		},
		{
			name:        "server fails to start",
			serverError: fmt.Errorf("failed to start server"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				listener: &mockHttpServer{
					listenAndServeFunc: func() error {
						return tt.serverError
					},
				},
			}

			err := s.Start()
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.serverError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_HandleIndex(t *testing.T) {
	tmpDir := t.TempDir()
	templateDir := filepath.Join(tmpDir, "web", "template")
	err := os.MkdirAll(templateDir, 0755)
	assert.NoError(t, err)

	indexHTML := `<!DOCTYPE html>
<html>
<head><title>Test Coinmon API</title></head>
<body>
	<h1>🪙 Test Coinmon API</h1>
	<p>Cryptocurrency price API with data from multiple exchanges</p>
	<div class="endpoint">
		<a href="/api/v1/spot/BTCUSDT">/api/v1/spot/BTCUSDT</a>
	</div>
</body>
</html>`

	err = os.WriteFile(filepath.Join(templateDir, "index.html"), []byte(indexHTML), 0644)
	assert.NoError(t, err)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	tests := []struct {
		name           string
		method         string
		path           string
		acceptEncoding string
		expectedStatus int
		expectedType   string
		expectedBody   string
		checkHeaders   bool
	}{
		{
			name:           "successful root request",
			method:         http.MethodGet,
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html; charset=utf-8",
			expectedBody:   "Test Coinmon API",
			checkHeaders:   true,
		},
		{
			name:           "contains API endpoint link",
			method:         http.MethodGet,
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "/api/v1/spot/BTCUSDT",
		},
		{
			name:           "root request with gzip compression",
			method:         http.MethodGet,
			path:           "/",
			acceptEncoding: "gzip, deflate",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html; charset=utf-8",
			expectedBody:   "Test Coinmon API",
			checkHeaders:   true,
		},
		{
			name:           "non-root path returns 404",
			method:         http.MethodGet,
			path:           "/nonexistent",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "POST method not allowed",
			method:         http.MethodPost,
			path:           "/",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "PUT method not allowed",
			method:         http.MethodPut,
			path:           "/",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "DELETE method not allowed",
			method:         http.MethodDelete,
			path:           "/",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				exchanges: exchanges,
				listener: &mockHttpServer{
					listenAndServeFunc: func() error { return nil },
				},
				client: &mockHttpClient{
					doFunc: mockSuccessfulResponse,
				},
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			w := httptest.NewRecorder()
			s.HandleIndex(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedType != "" {
				assert.Equal(t, tt.expectedType, w.Header().Get("Content-Type"))
			}

			if tt.expectedBody != "" {
				body := w.Body.String()

				if w.Header().Get("Content-Encoding") == "gzip" {
					gr, err := gzip.NewReader(w.Body)
					assert.NoError(t, err)
					defer gr.Close()

					decompressed, err := io.ReadAll(gr)
					assert.NoError(t, err)
					body = string(decompressed)
				}

				assert.Contains(t, body, tt.expectedBody)
			}

			if tt.checkHeaders && tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
				assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
				assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
			}
		})
	}
}

func TestServer_HandleIndex_TemplateNotFound(t *testing.T) {
	s := &Server{
		exchanges: exchanges,
		listener: &mockHttpServer{
			listenAndServeFunc: func() error { return nil },
		},
		client: &mockHttpClient{
			doFunc: mockSuccessfulResponse,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	s.HandleIndex(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "no such file or directory")
}

func TestServer_HandleIndex_InvalidTemplateSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	templateDir := filepath.Join(tmpDir, "web", "template")
	err := os.MkdirAll(templateDir, 0755)
	assert.NoError(t, err)

	invalidHTML := `<html><body>{{range}}</body></html>`
	err = os.WriteFile(filepath.Join(templateDir, "index.html"), []byte(invalidHTML), 0644)
	assert.NoError(t, err)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	s := &Server{
		exchanges: exchanges,
		listener: &mockHttpServer{
			listenAndServeFunc: func() error { return nil },
		},
		client: &mockHttpClient{
			doFunc: mockSuccessfulResponse,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	s.HandleIndex(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "range")
}

func TestServer_HandleIndex_TemplateExecuteError(t *testing.T) {
	tmpDir := t.TempDir()
	templateDir := filepath.Join(tmpDir, "web", "template")
	err := os.MkdirAll(templateDir, 0755)
	assert.NoError(t, err)

	errorHTML := `<html><body>{{printf .NonExistent}}</body></html>`
	err = os.WriteFile(filepath.Join(templateDir, "index.html"), []byte(errorHTML), 0644)
	assert.NoError(t, err)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	s := &Server{
		exchanges: exchanges,
		listener: &mockHttpServer{
			listenAndServeFunc: func() error { return nil },
		},
		client: &mockHttpClient{
			doFunc: mockSuccessfulResponse,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	s.HandleIndex(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Internal server error")
}

func TestServer_HandleIndex_GzipCompression(t *testing.T) {
	tmpDir := t.TempDir()
	templateDir := filepath.Join(tmpDir, "web", "template")
	err := os.MkdirAll(templateDir, 0755)
	assert.NoError(t, err)

	largeHTML := `<!DOCTYPE html><html><head><title>Large Page</title></head><body>`
	for i := 0; i < 100; i++ {
		largeHTML += fmt.Sprintf("<p>This is paragraph number %d with some content to make it larger.</p>", i)
	}
	largeHTML += `</body></html>`

	err = os.WriteFile(filepath.Join(templateDir, "index.html"), []byte(largeHTML), 0644)
	assert.NoError(t, err)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	tests := []struct {
		name           string
		acceptEncoding string
		expectGzip     bool
	}{
		{
			name:           "with gzip support",
			acceptEncoding: "gzip, deflate",
			expectGzip:     true,
		},
		{
			name:           "without gzip support",
			acceptEncoding: "deflate, br",
			expectGzip:     false,
		},
		{
			name:           "no accept-encoding header",
			acceptEncoding: "",
			expectGzip:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				exchanges: exchanges,
				listener: &mockHttpServer{
					listenAndServeFunc: func() error { return nil },
				},
				client: &mockHttpClient{
					doFunc: mockSuccessfulResponse,
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			w := httptest.NewRecorder()
			s.HandleIndex(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			if tt.expectGzip {
				assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))

				gr, err := gzip.NewReader(w.Body)
				assert.NoError(t, err)
				defer gr.Close()

				decompressed, err := io.ReadAll(gr)
				assert.NoError(t, err)
				assert.Contains(t, string(decompressed), "Large Page")
			} else {
				assert.Empty(t, w.Header().Get("Content-Encoding"))
				assert.Contains(t, w.Body.String(), "Large Page")
			}
		})
	}
}

func TestServer_HandleSpot(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		path             string
		mockResponse     mockResponseFunc
		expectedStatus   int
		expectedResponse string
		expectedContains bool
	}{
		{
			name:   "successful price request",
			method: http.MethodGet,
			path:   "/api/v1/spot/BTCUSDT",
			mockResponse: mockSuccessfulResponseWithDelay(map[string]time.Duration{
				"binance": 50 * time.Millisecond,
				"bybit":   100 * time.Millisecond,
				"bitget":  150 * time.Millisecond,
			}),
			expectedStatus:   http.StatusOK,
			expectedResponse: "99999.990000",
			expectedContains: false,
		},
		{
			name:   "successful detailed request",
			method: http.MethodGet,
			path:   "/api/v1/spot/BTCUSDT?details=true",
			mockResponse: mockSuccessfulResponseWithDelay(map[string]time.Duration{
				"binance": 50 * time.Millisecond,
				"bybit":   100 * time.Millisecond,
				"bitget":  150 * time.Millisecond,
			}),
			expectedStatus:   http.StatusOK,
			expectedResponse: `{"pair":"BTCUSDT","price":99999.99,"source":"binance"}`,
			expectedContains: true,
		},
		{
			name:             "method not allowed",
			method:           http.MethodPost,
			path:             "/api/v1/spot/BTCUSDT",
			mockResponse:     mockSuccessfulResponse,
			expectedStatus:   http.StatusMethodNotAllowed,
			expectedResponse: "Method not allowed\n",
			expectedContains: false,
		},
		{
			name:             "missing pair",
			method:           http.MethodGet,
			path:             "/api/v1/spot/",
			mockResponse:     mockSuccessfulResponse,
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: "Missing trading pair\n",
			expectedContains: false,
		},
		{
			name:             "invalid pair",
			method:           http.MethodGet,
			path:             "/api/v1/spot/INVALID",
			mockResponse:     mockInvalidPairResponse,
			expectedStatus:   http.StatusServiceUnavailable,
			expectedResponse: "all exchanges failed",
			expectedContains: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				exchanges: exchanges,
				client: &mockHttpClient{
					doFunc: tt.mockResponse,
				},
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			s.HandleSpot(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedContains {
				assert.Contains(t, w.Body.String(), tt.expectedResponse)
			} else {
				assert.Equal(t, tt.expectedResponse, w.Body.String())
			}
		})
	}
}

func TestServer_firstPriceWithDetails(t *testing.T) {
	type errorResponse struct {
		Message string   `json:"message"`
		Errors  []string `json:"errors"`
	}

	tests := []struct {
		name           string
		pair           string
		mockResponse   mockResponseFunc
		expectedPrice  float64
		expectedSource string
		expectError    bool
		expectedErrors []string
	}{
		{
			name: "successful response from first exchange",
			pair: "BTCUSDT",
			mockResponse: mockSuccessfulResponseWithDelay(map[string]time.Duration{
				"binance": 50 * time.Millisecond,
				"bybit":   100 * time.Millisecond,
				"bitget":  150 * time.Millisecond,
			}),
			expectedPrice:  99999.99,
			expectedSource: "binance",
			expectError:    false,
		},
		{
			name:         "all exchanges fail",
			pair:         "INVALID",
			mockResponse: mockInvalidPairResponse,
			expectError:  true,
			expectedErrors: []string{
				"bitget: code=40034, msg=Parameter does not exist",
				"bybit: code=10001, msg=Not supported symbols",
				"binance: code=-1100, msg=Illegal characters found in parameter 'symbol'; legal range is '^[A-Z0-9_.]{1,20}$'.",
			},
		},
		{
			name:         "empty pair",
			pair:         "",
			mockResponse: mockEmptyPairResponse,
			expectError:  true,
			expectedErrors: []string{
				"binance: code=-1105, msg=Parameter 'symbol' was empty.",
				"bybit: code=10001, msg=Not supported symbols",
				"bitget: code=40034, msg=Parameter does not exist",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				exchanges: exchanges,
				client: &mockHttpClient{
					doFunc: tt.mockResponse,
				},
			}

			price, source, err := s.firstPriceWithDetails(context.Background(), tt.pair)

			if tt.expectError {
				assert.Error(t, err)
				var errResp errorResponse
				assert.NoError(t, json.Unmarshal([]byte(err.Error()), &errResp))
				assert.Equal(t, "all exchanges failed", errResp.Message)

				for _, expectedErr := range tt.expectedErrors {
					assert.Contains(t, errResp.Errors, expectedErr)
				}
				assert.Equal(t, len(tt.expectedErrors), len(errResp.Errors))
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPrice, price)
			assert.Equal(t, tt.expectedSource, source)
		})
	}
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
				// t.Log(tt.exchange.Name, tt.pair, err)
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPrice, price)
		})
	}
}
