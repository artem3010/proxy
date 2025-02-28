package storage

import (
	"context"
	"proxy/internal/schema"
	"proxy/internal/storage/lru_cache"
	"time"
)

type emissionClient interface {
	GetEmissions(ctx context.Context, inventoryIds []schema.Row) ([]schema.Row, error)
}

type lruLocalCache[K comparable, V any] interface {
	BatchGet(keys []K) ([]V, []K)
	Update(rows []lru_cache.CacheItem[K, V])
	GetValues() []V
}
type redisCache[V any] interface {
	BatchGet(ctx context.Context, keys []string) ([]V, []string, error)
	Update(keys []string, values []V)
	Set(ctx context.Context, key string, value V, expiration time.Duration) error
}
