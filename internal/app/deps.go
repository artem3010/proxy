package app

import (
	"context"
	"proxy/internal/schema"
)

type lruCache[K comparable, V any] interface {
	GetValues() []V
}
type storage interface {
	Get(ctx context.Context, inventoryIds map[string]schema.Row) ([]schema.Row, error)
}
