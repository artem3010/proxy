package lru_cache

import (
	"context"
	"testing"
)

// TestNewLRUCache verifies that a newly created cache is empty and has the correct capacity.
func TestNewLRUCache(t *testing.T) {
	cache := NewLRUCache[string, int](context.Background(), 5, 5)
	if cache.capacity != 5 {
		t.Errorf("expected capacity 5, got %d", cache.capacity)
	}
	if len(cache.items) != 0 {
		t.Errorf("expected empty items, got %d", len(cache.items))
	}
	if cache.order.Len() != 0 {
		t.Errorf("expected empty order, got %d", cache.order.Len())
	}
}

// TestLRUCache_SetAndGet uses table-driven tests to check that setting and then getting a Key returns the expected Value.
func TestLRUCache_SetAndGet(t *testing.T) {
	// Note: This test assumes that Get is implemented as:
	//    return elem.Value.(*CacheItem[K, V]).Value, true
	testCases := []struct {
		name          string
		key           string
		value         int
		priority      int
		updatedKey    string // if non-empty, update the same Key with new Value/Priority
		updatedValue  int
		updatedPrio   int
		expectedValue int
	}{
		{
			name:          "Simple Set and Get",
			key:           "a",
			value:         1,
			priority:      10,
			expectedValue: 1,
		},
		{
			name:          "Update existing Key",
			key:           "a",
			value:         1,
			priority:      10,
			updatedKey:    "a",
			updatedValue:  2,
			updatedPrio:   20,
			expectedValue: 2,
		},
	}

	cache := NewLRUCache[string, int](context.Background(), 5, 5)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache.Set(tc.key, tc.value, tc.priority)
			// If an update is requested, perform it.
			if tc.updatedKey != "" {
				cache.Set(tc.updatedKey, tc.updatedValue, tc.updatedPrio)
			}
			val, ok := cache.Get(tc.key)
			if !ok {
				t.Fatalf("expected Key %q to be found", tc.key)
			}
			if val != tc.expectedValue {
				t.Errorf("expected Value %d, got %d", tc.expectedValue, val)
			}
		})
	}
}

// TestLRUCache_Delete verifies that after deletion a Key is no longer available.
func TestLRUCache_Delete(t *testing.T) {
	cache := NewLRUCache[string, int](context.Background(), 5, 5)
	cache.Set("a", 1, 10)
	cache.Delete("a")
	if _, ok := cache.Get("a"); ok {
		t.Error("expected Key 'a' to be deleted")
	}
}

// TestLRUCache_BatchGet verifies that BatchGet returns values for found keys and a list of not-found keys.
func TestLRUCache_BatchGet(t *testing.T) {
	cache := NewLRUCache[string, int](context.Background(), 5, 5)
	cache.Set("a", 1, 10)
	cache.Set("b", 2, 20)
	// Request keys "a" and "c" (where "c" is missing).
	keys := []string{"a", "c"}
	values, notFound := cache.BatchGet(keys)
	// Expect values contains only 1 and notFound contains "c".
	if len(values) != 1 || values[0] != 1 {
		t.Errorf("expected values [1], got %v", values)
	}
	if len(notFound) != 1 || notFound[0] != "c" {
		t.Errorf("expected notFound [\"c\"], got %v", notFound)
	}
}

// TestLRUCache_Eviction verifies that when capacity is exceeded, a candidate is evicted.
func TestLRUCache_Eviction(t *testing.T) {
	// Create a cache with capacity 2.
	cache := NewLRUCache[string, int](context.Background(), 2, 5)
	// Insert two items.
	cache.Set("a", 1, 10) // Priority 10
	cache.Set("b", 2, 20) // Priority 20
	// At this point, maxPriority should be 20.
	// When adding a new item with Priority 20, the candidate with Priority equal to maxPriority should be evicted.
	cache.Set("c", 3, 20)
	// According to the eviction logic, the candidate is selected from the back of the list.
	// For keys "a" (Priority 10) and "b" (Priority 20), the back element is "a" first;
	// then iterating backwards finds "b" which has Priority 20 and is evicted.
	if _, ok := cache.Get("b"); ok {
		t.Error("expected Key 'b' to be evicted")
	}
	// "a" and "c" should remain.
	if val, ok := cache.Get("a"); !ok || val != 1 {
		t.Errorf("expected Key 'a' to be present with Value 1, got %v", val)
	}
	if val, ok := cache.Get("c"); !ok || val != 3 {
		t.Errorf("expected Key 'c' to be present with Value 3, got %v", val)
	}
}
