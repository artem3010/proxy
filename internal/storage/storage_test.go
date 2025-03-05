package storage

import (
	"context"
	"errors"
	lru "proxy/internal/storage/lru_cache"
	"testing"
	"time"

	"proxy/internal/schema"

	"github.com/stretchr/testify/require"
)

type lruCacheMock struct {
	batchGetFunc func(keys []string) ([]schema.Row, []string)
	updateFunc   func(rows []lru.CacheItem[string, schema.Row])
}

func (m *lruCacheMock) BatchGet(keys []string) ([]schema.Row, []string) {
	return m.batchGetFunc(keys)
}
func (m *lruCacheMock) GetValues() []schema.Row {
	return m.GetValues()
}

func (m *lruCacheMock) SetBatch(rows []lru.CacheItem[string, schema.Row]) {
	if m.updateFunc != nil {
		m.updateFunc(rows)
	}
}

type redisCacheMock struct {
	batchGetFunc func(ctx context.Context, keys []string) ([]schema.Row, []string, error)
	updateFunc   func(redis []schema.Row)
}

func (m *redisCacheMock) Set(ctx context.Context, key string, value schema.Row, expiration time.Duration) error {
	return nil
}

func (m *redisCacheMock) BatchGet(ctx context.Context, keys []string) ([]schema.Row, []string, error) {
	return m.batchGetFunc(ctx, keys)
}

func (m *redisCacheMock) SetBatch(ctx context.Context, keys []string, values []schema.Row) {
	m.SetBatch(ctx, keys, values)
}

type emissionClientMock struct {
	getEmissionsFunc func(ctx context.Context, inventoryIds []schema.Row) ([]schema.Row, error)
}

func (m *emissionClientMock) GetEmissions(ctx context.Context, inventoryIds []schema.Row) ([]schema.Row, error) {
	return m.getEmissionsFunc(ctx, inventoryIds)
}

var (
	rowA = schema.Row{
		InventoryId: "A",
		Priority:    1,
		EmissionsBreakdown: schema.EmissionsBreakdown{
			TotalEmissionsGrams:  10,
			InventoryCoverage:    "full",
			ClimateRiskCompliant: true,
		},
	}
	rowB = schema.Row{
		InventoryId: "B",
		Priority:    2,
		EmissionsBreakdown: schema.EmissionsBreakdown{
			TotalEmissionsGrams:  20,
			InventoryCoverage:    "partial",
			ClimateRiskCompliant: false,
		},
	}
	rowC = schema.Row{
		InventoryId: "C",
		Priority:    3,
		EmissionsBreakdown: schema.EmissionsBreakdown{
			TotalEmissionsGrams:  30,
			InventoryCoverage:    "full",
			ClimateRiskCompliant: true,
		},
	}
)

func TestStorage_Get(t *testing.T) {
	tests := []struct {
		name           string
		ctxFunc        func() context.Context
		input          map[string]schema.Row
		lruBatchGet    func(keys []string) ([]schema.Row, []string)
		redisBatchGet  func(ctx context.Context, keys []string) ([]schema.Row, []string, error)
		emissionGet    func(ctx context.Context, inventoryIds []schema.Row) ([]schema.Row, error)
		expectedResult []schema.Row
	}{
		{
			name: "All found in LRU",
			ctxFunc: func() context.Context {
				return context.Background()
			},
			input: map[string]schema.Row{
				"A": rowA,
				"B": rowB,
			},
			lruBatchGet: func(keys []string) ([]schema.Row, []string) {
				return []schema.Row{rowA, rowB}, []string{}
			},
			redisBatchGet: func(ctx context.Context, keys []string) ([]schema.Row, []string, error) {
				return nil, nil, nil
			},
			emissionGet: func(ctx context.Context, inventoryIds []schema.Row) ([]schema.Row, error) {
				return nil, nil
			},
			expectedResult: []schema.Row{rowA, rowB},
		},
		{
			name: "Missing in LRU, found in Redis and Emission",
			ctxFunc: func() context.Context {
				return context.Background()
			},
			input: map[string]schema.Row{
				"A": rowA,
				"B": rowB,
				"C": rowC,
			},
			lruBatchGet: func(keys []string) ([]schema.Row, []string) {
				return []schema.Row{rowA}, []string{"B", "C"}
			},
			redisBatchGet: func(ctx context.Context, keys []string) ([]schema.Row, []string, error) {
				return []schema.Row{rowB}, []string{"C"}, nil
			},
			emissionGet: func(ctx context.Context, inventoryIds []schema.Row) ([]schema.Row, error) {
				return []schema.Row{rowC}, nil
			},
			expectedResult: []schema.Row{rowA, rowB, rowC},
		},
		{
			name: "Redis error and Emission error",
			ctxFunc: func() context.Context {
				return context.Background()
			},
			input: map[string]schema.Row{
				"A": rowA,
				"B": rowB,
			},
			lruBatchGet: func(keys []string) ([]schema.Row, []string) {
				return []schema.Row{}, []string{"A", "B"}
			},
			redisBatchGet: func(ctx context.Context, keys []string) ([]schema.Row, []string, error) {
				return nil, []string{"A", "B"}, errors.New("redis error")
			},
			emissionGet: func(ctx context.Context, inventoryIds []schema.Row) ([]schema.Row, error) {
				return nil, errors.New("emission error")
			},
			expectedResult: []schema.Row{},
		},
		{
			name: "Context canceled before Redis fetch",
			ctxFunc: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // cancel immediately
				return ctx
			},
			input: map[string]schema.Row{
				"A": rowA,
				"B": rowB,
			},
			lruBatchGet: func(keys []string) ([]schema.Row, []string) {
				return []schema.Row{}, []string{"A", "B"}
			},
			redisBatchGet: func(ctx context.Context, keys []string) ([]schema.Row, []string, error) {
				return []schema.Row{rowA}, []string{"B"}, nil
			},
			emissionGet: func(ctx context.Context, inventoryIds []schema.Row) ([]schema.Row, error) {
				return nil, nil
			},
			expectedResult: []schema.Row{},
		},
		{
			name: "Context canceled before Emission fetch",
			ctxFunc: func() context.Context {
				ctx, _ := context.WithTimeout(context.Background(), 1*time.Millisecond)
				time.Sleep(2 * time.Millisecond)
				return ctx
			},
			input: map[string]schema.Row{
				"A": rowA,
			},
			lruBatchGet: func(keys []string) ([]schema.Row, []string) {
				return []schema.Row{}, []string{"A"}
			},
			redisBatchGet: func(ctx context.Context, keys []string) ([]schema.Row, []string, error) {
				return []schema.Row{}, []string{"A"}, nil
			},
			emissionGet: func(ctx context.Context, inventoryIds []schema.Row) ([]schema.Row, error) {
				time.Sleep(10 * time.Millisecond)
				return []schema.Row{rowA}, nil
			},
			expectedResult: []schema.Row{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.ctxFunc()

			lruMock := &lruCacheMock{
				batchGetFunc: tc.lruBatchGet,
				updateFunc:   func(rows []lru.CacheItem[string, schema.Row]) {},
			}
			redisMock := &redisCacheMock{
				batchGetFunc: tc.redisBatchGet,
			}
			emissionMock := &emissionClientMock{
				getEmissionsFunc: tc.emissionGet,
			}

			s := &storage{
				lruLocalCache:   lruMock,
				redisCache:      redisMock,
				emissionService: emissionMock,
			}

			result, err := s.Get(ctx, tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			require.Equal(t, result, tc.expectedResult)
		})
	}
}

func TestExtractKeys(t *testing.T) {
	input := map[string]schema.Row{
		"A": rowA,
		"B": rowB,
		"C": rowC,
	}
	keys := extractKeys(input)
	expectedKeys := []string{"A", "B", "C"}
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}
	for _, expected := range expectedKeys {
		if !keyMap[expected] {
			t.Errorf("expected key %q not found in result %v", expected, keys)
		}
	}
}
