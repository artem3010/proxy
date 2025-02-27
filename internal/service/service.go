package service

import (
	"context"
	"proxy/internal/schema"
)

type Service struct {
	storage storage
}

func New(storage storage) *Service {
	return &Service{
		storage: storage,
	}
}

func (s *Service) Get(ctx context.Context, inventoryIds []schema.Row) ([]schema.Row, error) {
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
		if _, value := allKeys[item.InventoryId]; !value {
			allKeys[item.InventoryId] = true
			list = append(list, item)
		}
	}
	return list
}
