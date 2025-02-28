package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"proxy/internal/dto/socope_v3_dto"
	"strings"
	"testing"
	"time"
)

// roundTripFunc позволяет подменять выполнение HTTP-запроса.
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNewClient(t *testing.T) {
	testCases := []struct {
		name        string
		inputURL    string
		expectedURL string
		wantErr     bool
	}{
		{
			name:     "err, empty url",
			inputURL: "",
			wantErr:  true,
		},
		{
			name:        "user URL",
			inputURL:    "http://example.com/api",
			expectedURL: "http://example.com/api",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClient(tc.inputURL, 1*time.Millisecond)
			if tc.wantErr && err != nil {
				return
			}
			if client.APIURL != tc.expectedURL {
				t.Errorf("ожидали APIURL = %q, получили %q", tc.expectedURL, client.APIURL)
			}
		})
	}
}

func TestFetchEmissions(t *testing.T) {
	expectedResponse := socope_v3_dto.ResponseBody{
		Rows: []socope_v3_dto.ResponseRow{
			{
				InventoryID: "123",
				EmissionsBreakdown: socope_v3_dto.EmissionsBreakdown{
					TotalEmissionsGrams:  100.0,
					InventoryCoverage:    "full",
					ClimateRiskCompliant: true,
				},
			},
		},
	}
	successRespBytes, _ := json.Marshal(expectedResponse)

	testCases := []struct {
		name             string
		transport        http.RoundTripper
		request          socope_v3_dto.RequestBody
		expectedResponse *socope_v3_dto.ResponseBody
		expectedError    string
	}{
		{
			name: "ok, success",
			transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost {
					return nil, errors.New("wrong method")
				}
				if ct := req.Header.Get("Content-Type"); ct != "application/json" {
					return nil, errors.New("wrong Content-Type")
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(successRespBytes)),
					Header:     make(http.Header),
				}, nil
			}),
			request: socope_v3_dto.RequestBody{
				Rows: []socope_v3_dto.RequestRow{
					{InventoryID: "123", Priority: 1},
				},
			},
			expectedResponse: &expectedResponse,
			expectedError:    "",
		},
		{
			name: "code is not 200",
			transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Status:     "500 Internal Server Error",
					Body:       io.NopCloser(bytes.NewReader([]byte("error"))),
					Header:     make(http.Header),
				}, nil
			}),
			request: socope_v3_dto.RequestBody{
				Rows: []socope_v3_dto.RequestRow{
					{InventoryID: "123", Priority: 1},
				},
			},
			expectedResponse: nil,
			expectedError:    "unexpected status code",
		},
		{
			name: "err",
			transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("network error")
			}),
			request: socope_v3_dto.RequestBody{
				Rows: []socope_v3_dto.RequestRow{
					{InventoryID: "123", Priority: 1},
				},
			},
			expectedResponse: nil,
			expectedError:    "network error",
		},
		{
			name: "decoding error JSON",
			transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte("invalid json"))),
					Header:     make(http.Header),
				}, nil
			}),
			request: socope_v3_dto.RequestBody{
				Rows: []socope_v3_dto.RequestRow{
					{InventoryID: "123", Priority: 1},
				},
			},
			expectedResponse: nil,
			// При попытке декодирования JSON будет возвращена ошибка, содержащая сообщение вроде "invalid character..."
			expectedError: "invalid character",
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			client, _ := NewClient("http://dummy", 0)
			client.client.Transport = tc.transport

			ctx := context.Background()
			resp, err := client.FetchEmissions(ctx, tc.request)

			if tc.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("want %q, got %v", tc.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Для простоты сравнения сериализуем полученный и ожидаемый ответы.
			actualBytes, err := json.Marshal(resp)
			if err != nil {
				t.Errorf("marshal err: %v", err)
			}
			expectedBytes, err := json.Marshal(tc.expectedResponse)
			if err != nil {
				t.Errorf("marshal err: %v", err)
			}
			if !bytes.Equal(actualBytes, expectedBytes) {
				t.Errorf("want %s, got %s", string(expectedBytes), string(actualBytes))
			}
		})
	}
}
