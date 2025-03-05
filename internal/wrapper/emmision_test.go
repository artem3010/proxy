package wrapper

import (
	"context"
	"errors"
	socopev3dto "proxy/internal/dto/socope_v3_dto"
	"testing"
	"time"

	"proxy/internal/schema"

	"github.com/stretchr/testify/require"
)

// emissionClientMock implements the emissionClient interface for testing.
type emissionClientMock struct {
	fetchFunc func(ctx context.Context, request socopev3dto.RequestBody) (*socopev3dto.ResponseBody, error)
}

func (m *emissionClientMock) FetchEmissions(ctx context.Context, request socopev3dto.RequestBody) (*socopev3dto.ResponseBody, error) {
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
	dtoResponse := &socopev3dto.ResponseBody{
		Rows: []socopev3dto.ResponseRow{
			{
				InventoryID: "id1",
				EmissionsBreakdown: socopev3dto.EmissionsBreakdown{
					TotalEmissionsGrams:  100.0,
					InventoryCoverage:    "full",
					ClimateRiskCompliant: true,
				},
			},
			{
				InventoryID: "id2",
				EmissionsBreakdown: socopev3dto.EmissionsBreakdown{
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
		fetchResponse *socopev3dto.ResponseBody
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
				fetchFunc: func(ctx context.Context, request socopev3dto.RequestBody) (*socopev3dto.ResponseBody, error) {
					// Optionally, you could check the content of request here.
					return tc.fetchResponse, tc.fetchError
				},
			}
			svc := &service{
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
			require.Equal(t, rows, tc.expectedRows)
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

	expected := socopev3dto.RequestBody{
		Rows: []socopev3dto.RequestRow{
			{InventoryID: "id1"},
			{InventoryID: "id2"},
			{InventoryID: "id3"},
		},
	}

	result := toDto(inputIDs)
	require.Equal(t, result, expected)
}

func TestDtoSchema(t *testing.T) {
	inputResponse := &socopev3dto.ResponseBody{
		Rows: []socopev3dto.ResponseRow{
			{
				InventoryID: "id1",
				EmissionsBreakdown: socopev3dto.EmissionsBreakdown{
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
	require.Equal(t, result, expected)
}
