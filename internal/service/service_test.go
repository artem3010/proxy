package service

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"proxy/internal/schema"
)

type storageMock struct {
	getFunc func(ctx context.Context, inventoryIds map[string]schema.Row) ([]schema.Row, error)
}

func (m *storageMock) Get(ctx context.Context, inventoryIds map[string]schema.Row) ([]schema.Row, error) {
	return m.getFunc(ctx, inventoryIds)
}

func TestService_Get(t *testing.T) {
	row1 := schema.Row{
		InventoryId: "1",
		Priority:    1,
		EmissionsBreakdown: schema.EmissionsBreakdown{
			TotalEmissionsGrams:  10.0,
			InventoryCoverage:    "full",
			ClimateRiskCompliant: true,
		},
	}
	row2 := schema.Row{
		InventoryId: "2",
		Priority:    2,
		EmissionsBreakdown: schema.EmissionsBreakdown{
			TotalEmissionsGrams:  20.0,
			InventoryCoverage:    "partial",
			ClimateRiskCompliant: false,
		},
	}
	row1Dup := schema.Row{
		InventoryId: "1",
		Priority:    2,
		EmissionsBreakdown: schema.EmissionsBreakdown{
			TotalEmissionsGrams:  15.0,
			InventoryCoverage:    "partial",
			ClimateRiskCompliant: false,
		},
	}

	testCases := []struct {
		name           string
		input          []schema.Row
		storageFunc    func(ctx context.Context, ids map[string]schema.Row) ([]schema.Row, error)
		expectedOutput []schema.Row
		expectedError  string
	}{
		{
			name:           "Empty input returns empty slice",
			input:          []schema.Row{},
			storageFunc:    nil,
			expectedOutput: []schema.Row{},
			expectedError:  "",
		},
		{
			name:  "Unique rows successful retrieval",
			input: []schema.Row{row1, row2},
			storageFunc: func(ctx context.Context, ids map[string]schema.Row) ([]schema.Row, error) {
				if len(ids) != 2 {
					return nil, errors.New("unexpected number of keys in map")
				}
				return []schema.Row{row1, row2}, nil
			},
			expectedOutput: []schema.Row{row1, row2},
			expectedError:  "",
		},
		{
			name:  "Duplicate rows are removed",
			input: []schema.Row{row1, row1Dup},
			storageFunc: func(ctx context.Context, ids map[string]schema.Row) ([]schema.Row, error) {
				// Expect only one key ("1") in the map.
				if len(ids) != 1 {
					return nil, errors.New("duplicates were not removed")
				}
				// Return the first row.
				return []schema.Row{row1}, nil
			},
			expectedOutput: []schema.Row{row1},
			expectedError:  "",
		},
		{
			name:  "Storage returns error",
			input: []schema.Row{row1},
			storageFunc: func(ctx context.Context, ids map[string]schema.Row) ([]schema.Row, error) {
				return nil, errors.New("storage error")
			},
			expectedOutput: nil,
			expectedError:  "storage error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			serviceInstance := New(nil)
			if tc.storageFunc != nil {
				serviceInstance.storage = &storageMock{
					getFunc: tc.storageFunc,
				}
			} else {
				serviceInstance.storage = &storageMock{
					getFunc: func(ctx context.Context, ids map[string]schema.Row) ([]schema.Row, error) {
						return []schema.Row{}, nil
					},
				}
			}

			output, err := serviceInstance.Get(context.Background(), tc.input)
			if tc.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.expectedError) {
					t.Fatalf("expected error containing %q, got %v", tc.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(output, tc.expectedOutput) {
				t.Errorf("expected output %v, got %v", tc.expectedOutput, output)
			}
		})
	}
}

func TestRemoveDuplicates(t *testing.T) {
	row1 := schema.Row{InventoryId: "1", Priority: 1}
	row1Dup := schema.Row{InventoryId: "1", Priority: 2}
	row2 := schema.Row{InventoryId: "2", Priority: 2}

	testCases := []struct {
		name     string
		input    []schema.Row
		expected []schema.Row
	}{
		{
			name:     "No duplicates",
			input:    []schema.Row{row1, row2},
			expected: []schema.Row{row1, row2},
		},
		{
			name:     "With duplicates",
			input:    []schema.Row{row1, row1Dup, row2},
			expected: []schema.Row{row1, row2},
		},
		{
			name:     "All duplicates",
			input:    []schema.Row{row1, row1Dup},
			expected: []schema.Row{row1},
		},
		{
			name:     "Empty input",
			input:    []schema.Row{},
			expected: []schema.Row{},
		},
	}

	for _, tc := range testCases {
		tc := tc // capture loop variable
		t.Run(tc.name, func(t *testing.T) {
			output := removeDuplicates(tc.input)
			if !testEq(output, tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, output)
			}
		})
	}
}
func testEq(a, b []schema.Row) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestCollectToMap(t *testing.T) {
	row1 := schema.Row{InventoryId: "1", Priority: 1}
	row2 := schema.Row{InventoryId: "2", Priority: 2}

	testCases := []struct {
		name     string
		input    []schema.Row
		expected map[string]schema.Row
	}{
		{
			name:  "Collect two rows",
			input: []schema.Row{row1, row2},
			expected: map[string]schema.Row{
				"1": row1,
				"2": row2,
			},
		},
		{
			name:     "Empty input",
			input:    []schema.Row{},
			expected: map[string]schema.Row{},
		},
		{
			name:  "Duplicates override",
			input: []schema.Row{row1, row1},
			expected: map[string]schema.Row{
				"1": row1,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // capture loop variable
		t.Run(tc.name, func(t *testing.T) {
			output := collectToMap(tc.input)
			if !reflect.DeepEqual(output, tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, output)
			}
		})
	}
}
