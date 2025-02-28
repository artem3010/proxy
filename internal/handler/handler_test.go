package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"proxy/internal/schema"
)

type measureGetterMock struct {
	getFunc func(ctx context.Context, inventoryIds map[string]schema.Row) ([]schema.Row, error)
}

func (m *measureGetterMock) Get(ctx context.Context, inventoryIds map[string]schema.Row) ([]schema.Row, error) {
	return m.getFunc(ctx, inventoryIds)
}

func TestHandler_Handle(t *testing.T) {
	testCases := []struct {
		name             string
		method           string
		requestBody      string
		measureGetter    func(ctx context.Context, inventoryIds map[string]schema.Row) ([]schema.Row, error)
		expectedStatus   int
		expectedResponse string
	}{
		{
			name:             "Invalid HTTP Method",
			method:           http.MethodGet,
			requestBody:      `{"inventoryIds": [{"id": "1", "priority": 1}]}`,
			measureGetter:    nil, // Not used because method is checked first
			expectedStatus:   http.StatusMethodNotAllowed,
			expectedResponse: "Invalid request method",
		},
		{
			name:             "Invalid JSON",
			method:           http.MethodPost,
			requestBody:      `invalid json`,
			measureGetter:    nil,
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: "Invalid request body",
		},
		{
			name:             "Validation error (empty inventoryIds)",
			method:           http.MethodPost,
			requestBody:      `{"inventoryIdss": []}`,
			measureGetter:    nil,
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: "Invalid request body",
		},
		{
			name:        "measureGetter returns error",
			method:      http.MethodPost,
			requestBody: `{"inventoryIds": [{"id": "1", "priority": 1}]}`,
			measureGetter: func(ctx context.Context, inventoryIds map[string]schema.Row) ([]schema.Row, error) {
				return nil, errors.New("get error")
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: "Internal server error",
		},
		{
			name:        "Successful response",
			method:      http.MethodPost,
			requestBody: `{"inventoryIds": [{"id": "1", "priority": 1}]}`,
			measureGetter: func(ctx context.Context, inventoryIds map[string]schema.Row) ([]schema.Row, error) {
				return []schema.Row{
					{
						InventoryId: "1",
						Priority:    1,
						EmissionsBreakdown: schema.EmissionsBreakdown{
							TotalEmissionsGrams:  123.45,
							InventoryCoverage:    "partial",
							ClimateRiskCompliant: true,
						},
					},
				}, nil
			},
			expectedStatus:   http.StatusOK,
			expectedResponse: `"inventoryId":"1"`,
		},
	}

	for _, tc := range testCases {
		tc := tc // capture loop variable
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			h := New(nil, 0)
			if tc.measureGetter != nil {
				h.measureGetter = &measureGetterMock{
					getFunc: tc.measureGetter,
				}
			} else {
				h.measureGetter = &measureGetterMock{
					getFunc: func(ctx context.Context, inventoryIds map[string]schema.Row) ([]schema.Row, error) {
						return nil, nil
					},
				}
			}

			h.Handle(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, rr.Code)
			}

			if !strings.Contains(rr.Body.String(), tc.expectedResponse) {
				t.Errorf("expected response to contain %q, got %q", tc.expectedResponse, rr.Body.String())
			}
		})
	}
}
