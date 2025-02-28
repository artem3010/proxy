package service

import (
	"context"
	"proxy/internal/schema"
)

type storage interface {
	Get(ctx context.Context, inventoryIds map[string]schema.Row) ([]schema.Row, error)
}
