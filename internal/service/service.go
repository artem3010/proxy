package service

import (
	"context"
	"proxy/internal/schema"
)

type service struct {
	storage storage
}

// New returns service for emissions
func New(storage storage) *service {
	return &service{
		storage: storage,
	}
}

// Get returns emissions for ids
func (s *service) Get(ctx context.Context, inventoryIds []schema.Row) ([]schema.Row, error) {
	if len(inventoryIds) == 0 {
		return []schema.Row{}, nil
	}

	inventoryIds = removeDuplicates(inventoryIds)

	idsMap := collectToMap(inventoryIds)

	return s.storage.Get(ctx, idsMap)
}

func collectToMap(ids []schema.Row) map[string]schema.Row {
	result := make(map[string]schema.Row, len(ids))

	for _, val := range ids {
		result[val.InventoryId] = val
	}

	return result
}

func removeDuplicates(ids []schema.Row) []schema.Row {
	allKeys := make(map[string]bool)
	var list []schema.Row
	for _, item := range ids {
		if !allKeys[item.InventoryId] {
			allKeys[item.InventoryId] = true
			list = append(list, item)
		}
	}
	return list
}
