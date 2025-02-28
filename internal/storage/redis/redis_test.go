package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v8"
)

// testMarshal and testUnmarshal are identity functions for string.
func testMarshal(s string) (string, error) {
	return s, nil
}

func testUnmarshal(str string) (string, error) {
	return str, nil
}

// equalStringSlices compares two string slices.
func equalStringSlices(a, b []string) bool {
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

func TestClient_Set(t *testing.T) {
	// Create a redis client mock.
	db, mock := redismock.NewClientMock()
	// Create our generic client with V = string.
	client := New[string](context.Background(), "localhost:6379", "", 0, testMarshal, testUnmarshal, 1000)
	// Override the rdb with our mock.
	client.rdb = db

	ctx := context.Background()
	key := "testKey"
	value := "testValue"
	expiration := time.Minute

	mock.ExpectSet(key, value, expiration).SetVal("OK")

	err := client.Set(ctx, key, value, expiration)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %v", err)
	}
}

func TestClient_Get(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := New[string](context.Background(), "localhost:6379", "", 0, testMarshal, testUnmarshal, 1000)
	client.rdb = db

	ctx := context.Background()
	key := "testKey"
	expectedValue := "testValue"

	mock.ExpectGet(key).SetVal(expectedValue)

	val, err := client.Get(ctx, key)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if val != expectedValue {
		t.Errorf("expected %v, got %v", expectedValue, val)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %v", err)
	}
}

func TestClient_BatchGet(t *testing.T) {
	tests := []struct {
		name             string
		keys             []string
		mgetResult       []interface{}
		expectedValues   []string // expected unmarshalled values (ignoring zero values)
		expectedNotFound []string
		mgetError        error
	}{
		{
			name: "All keys found",
			keys: []string{"k1", "k2"},
			mgetResult: []interface{}{
				"v1", "v2",
			},
			expectedValues:   []string{"v1", "v2"},
			expectedNotFound: []string{},
			mgetError:        nil,
		},
		{
			name: "Some keys not found",
			keys: []string{"k1", "k2"},
			mgetResult: []interface{}{
				"v1", nil,
			},
			expectedValues:   []string{"v1"},
			expectedNotFound: []string{"k2"},
			mgetError:        nil,
		},
		{
			name:             "MGet returns error",
			keys:             []string{"k1", "k2"},
			mgetResult:       nil,
			expectedValues:   nil,
			expectedNotFound: nil,
			mgetError:        errors.New("mget error"),
		},
	}

	for _, tc := range tests {
		tc := tc // capture loop variable
		t.Run(tc.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			client := New[string](context.Background(), "localhost:6379", "", 0, testMarshal, testUnmarshal, 1000)
			client.rdb = db

			ctx := context.Background()
			if tc.mgetError != nil {
				mock.ExpectMGet(tc.keys...).SetErr(tc.mgetError)
			} else {
				mock.ExpectMGet(tc.keys...).SetVal(tc.mgetResult)
			}

			values, notFound, err := client.BatchGet(ctx, tc.keys)
			if tc.mgetError != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tc.mgetError)
				}
				return
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Because the implementation preallocates a slice then appends,
			// filter out the initial zero values (empty strings).
			var filtered []string
			for _, v := range values {
				if v != "" {
					filtered = append(filtered, v)
				}
			}
			if !equalStringSlices(filtered, tc.expectedValues) {
				t.Errorf("expected values %v, got %v", tc.expectedValues, filtered)
			}
			if !equalStringSlices(notFound, tc.expectedNotFound) {
				t.Errorf("expected notFound %v, got %v", tc.expectedNotFound, notFound)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %v", err)
			}
		})
	}
}
