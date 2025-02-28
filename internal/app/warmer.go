package app

import (
	"context"
	"encoding/json"
	"proxy/internal/schema"
	"proxy/internal/storage"
	"proxy/internal/storage/lru_cache"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

const warmUpKey = "warmingUpKey"

func startWarmUpper(ctx context.Context,
	redisAddr string,
	redisPassword string,
	redisDb int,
	warmupSaverPeriod int64,
	cache *lru_cache.LRUCache[string, schema.Row],
	storage *storage.Storage,
) {
	redis := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDb,
	})

	go exportIDsPeriodically(ctx, time.Duration(warmupSaverPeriod)*time.Hour, cache, redis)

	go warmup(ctx, redis, storage)
}

func warmup(ctx context.Context, redis *redis.Client, s *storage.Storage) {
	select {
	case <-ctx.Done():
		log.Info().Msg("ExportIDsPeriodically: context canceled, stopping export goroutine")
		return
	default:
		keys, err := redis.Get(ctx, warmUpKey).Result()
		if err != nil {
			log.Error().Err(err).Msg("couldn't warm up")
			return
		}

		var rows []string
		if err := json.Unmarshal([]byte(keys), &rows); err != nil {
			log.Error().Err(err).Msg("Error unmarshaling data into schema.Row")
			return
		}

		if rows == nil {
			return
		}

		s.Get(ctx, toMap(rows))
	}
}

func toMap(rows []string) map[string]schema.Row {
	result := make(map[string]schema.Row, len(rows))

	for _, val := range rows {
		result[val] = schema.Row{
			InventoryId: val,
		}
	}
	return result
}

func exportIDsPeriodically(ctx context.Context, interval time.Duration, cache *lru_cache.LRUCache[string, schema.Row], redis *redis.Client) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("ExportIDsPeriodically: context canceled, stopping export goroutine")
			return
		case <-ticker.C:

			values := cache.GetValues()

			keys := make([]string, 0, len(values))

			for _, val := range values {
				keys = append(keys, val.InventoryId)
			}
			data, err := json.Marshal(keys)
			if err != nil {
				log.Error().Err(err).Msg("Error marshaling cache keys")
				continue
			}

			if err != nil {
				log.Error().Err(err).Msg("Error compressing cache keys")
				continue
			}
			//TODO use shared lock
			redis.Set(ctx, warmUpKey, data, 0)
		}
	}
}
