package handler

import (
	"context"
	"proxy/internal/schema"
)

type measureGetter interface {
	Get(ctx context.Context, inventoryIds map[string]schema.Row) ([]schema.Row, error)
}
