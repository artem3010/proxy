package wrapper

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"proxy/internal/dto/socope_v3_dto"
	"proxy/internal/schema"
)

// emissionClientMock implements the emissionClient interface for testing.
type emissionClientMock struct {
	fetchFunc func(ctx context.Context, request socope_v3_dto.RequestBody) (*socope_v3_dto.ResponseBody, error)
}

func (m *emissionClientMock) FetchEmissions(ctx context.Context, request socope_v3_dto.RequestBody) (*socope_v3_dto.ResponseBody, error) {
	return m.fetchFunc(ctx, request)
}

func TestService_GetEmissions(t *testing.T) {
	// Sample input IDs and expected conversion result.
	sampleIDs := []schema.Row{
		{
			InventoryId: "id1",
		}, {
			InventoryId: "id2",
		},
	}
	dtoResponse := &socope_v3_dto.ResponseBody{
		Rows: []socope_v3_dto.ResponseRow{
			{
				InventoryID: "id1",
				EmissionsBreakdown: socope_v3_dto.EmissionsBreakdown{
					TotalEmissionsGrams:  100.0,
					InventoryCoverage:    "full",
					ClimateRiskCompliant: true,
				},
			},
			{
				InventoryID: "id2",
				EmissionsBreakdown: socope_v3_dto.EmissionsBreakdown{
					TotalEmissionsGrams:  200.0,
					InventoryCoverage:    "partial",
					ClimateRiskCompliant: false,
				},
			},
		},
	}
	expectedSchema := []schema.Row{
		{
			InventoryId: "id1",
			EmissionsBreakdown: schema.EmissionsBreakdown{
				TotalEmissionsGrams:  100.0,
				InventoryCoverage:    "full",
				ClimateRiskCompliant: true,
			},
		},
		{
			InventoryId: "id2",
			EmissionsBreakdown: schema.EmissionsBreakdown{
				TotalEmissionsGrams:  200.0,
				InventoryCoverage:    "partial",
				ClimateRiskCompliant: false,
			},
		},
	}

	testCases := []struct {
		name          string
		timeout       time.Duration
		fetchResponse *socope_v3_dto.ResponseBody
		fetchError    error
		inputIDs      []schema.Row
		expectedRows  []schema.Row
		expectedError string
	}{
		{
			name:          "Successful response",
			timeout:       100 * time.Millisecond,
			fetchResponse: dtoResponse,
			fetchError:    nil,
			inputIDs:      sampleIDs,
			expectedRows:  expectedSchema,
		},
		{
			name:          "Emission client error",
			timeout:       100 * time.Millisecond,
			fetchResponse: nil,
			fetchError:    errors.New("fetch error"),
			inputIDs:      sampleIDs,
			expectedError: "fetch error",
		},
		{
			name:          "Empty input returns empty slice",
			timeout:       100 * time.Millisecond,
			fetchResponse: dtoResponse, // Even if response exists, no IDs are requested.
			fetchError:    nil,
			inputIDs:      []schema.Row{},
			expectedRows:  []schema.Row{},
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Create a service with the mocked emission client.
			mockClient := &emissionClientMock{
				fetchFunc: func(ctx context.Context, request socope_v3_dto.RequestBody) (*socope_v3_dto.ResponseBody, error) {
					// Optionally, you could check the content of request here.
					return tc.fetchResponse, tc.fetchError
				},
			}
			svc := &Service{
				emissionClient: mockClient,
				timeout:        tc.timeout,
			}
			ctx := context.Background()
			rows, err := svc.GetEmissions(ctx, tc.inputIDs)
			if tc.expectedError != "" {
				if err == nil || err.Error() != tc.expectedError {
					t.Fatalf("expected error %q, got %v", tc.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(rows, tc.expectedRows) {
				t.Errorf("expected rows %v, got %v", tc.expectedRows, rows)
			}
		})
	}
}

func TestToDto(t *testing.T) {
	inputIDs := []schema.Row{
		{
			InventoryId: "id1",
		},
		{
			InventoryId: "id2",
		},
		{
			InventoryId: "id3",
		},
	}

	expected := socope_v3_dto.RequestBody{
		Rows: []socope_v3_dto.RequestRow{
			{InventoryID: "id1"},
			{InventoryID: "id2"},
			{InventoryID: "id3"},
		},
	}

	result := toDto(inputIDs)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestDtoSchema(t *testing.T) {
	inputResponse := &socope_v3_dto.ResponseBody{
		Rows: []socope_v3_dto.ResponseRow{
			{
				InventoryID: "id1",
				EmissionsBreakdown: socope_v3_dto.EmissionsBreakdown{
					TotalEmissionsGrams:  123.4,
					InventoryCoverage:    "coverage1",
					ClimateRiskCompliant: true,
				},
			},
		},
	}
	expected := []schema.Row{
		{
			InventoryId: "id1",
			EmissionsBreakdown: schema.EmissionsBreakdown{
				TotalEmissionsGrams:  123.4,
				InventoryCoverage:    "coverage1",
				ClimateRiskCompliant: true,
			},
		},
	}

	result := toSchema(inputResponse)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}
