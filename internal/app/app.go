package app

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"proxy/internal/client"
	"proxy/internal/env"
	"proxy/internal/handler"
	"proxy/internal/schema"
	"proxy/internal/storage"
	"proxy/internal/storage/local_lru_cache"
	redisStorage "proxy/internal/storage/redis"
	"proxy/internal/wrapper"
	"proxy/middleware"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type App struct{}

const (
	successCode = 0
)

func New() *App {
	return &App{}
}

func (a *App) Run() (exitCode int) {
	ctx := context.Background()

	env.LoadEnv()
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	mux := http.NewServeMux()

	serverPort := env.GetEnv("PORT", "8080")
	redisAddr := env.GetEnv("REDIS_ADDR", "")
	redisPassword := env.GetEnv("REDIS_PASSWORD", "")
	redisDb, err := strconv.Atoi(env.GetEnv("REDIS_DB", "0"))
	redisAsyncChanSize, err := strconv.Atoi(env.GetEnv("REDIS_CHAN_SIZE", "1000"))
	if err != nil {
		log.Error().Msg("can't parse REDIS_CHAN_SIZE")
		return 1
	}
	lruCacheSize, err := strconv.Atoi(env.GetEnv("LRU_CACHE_SIZE", "1000"))
	if err != nil {
		log.Error().Msg("can't parse LRU_CACHE_SIZE")
		return 1
	}
	lruChanSize, err := strconv.Atoi(env.GetEnv("LRU_CHAN_SIZE", "1000"))
	if err != nil {
		log.Error().Msg("can't parse LRU_CHAN_SIZE")
		return 1
	}
	emissionTimeoutMs, err := strconv.ParseInt(env.GetEnv("EMISSION_TIMEOUT_MS", "1000"), 10, 64)
	if err != nil {
		log.Error().Msg("can't parse EMISSION_TIMEOUT_MS")
		return 1
	}
	updatePeriod, err := strconv.ParseInt(env.GetEnv("UPDATE_CACHE_PERIOD_HOUR", "1000"), 10, 64)
	if err != nil {
		log.Error().Msg("can't parse UPDATE_CACHE_PERIOD_HOUR")
		return 1
	}
	apiTimeout, err := strconv.ParseInt(env.GetEnv("V1_MEASURE_TIMEOUT_MS", "100"), 10, 64)
	if err != nil {
		log.Error().Msg("can't parse V1_MEASURE_TIMEOUT_MS")
		return 1
	}
	warmupSaverPeriod, err := strconv.ParseInt(env.GetEnv("WARMUP_SAVER_PERIOD_HOUR", "1"), 10, 64)
	if err != nil {
		log.Error().Msg("can't parse V1_MEASURE_TIMEOUT_MS")
		return 1
	}
	apiUrl := env.GetEnv("EMISSION_URL", "http://localhost:8081/v2/measure")
	if err != nil {
		log.Error().Msg("can't parse EMISSION_URL")
		return 1
	}
	lruCache := local_lru_cache.NewLRUCache[string, schema.Row](ctx,
		lruCacheSize,
		lruChanSize,
	)
	redis := redisStorage.NewClient[schema.Row](ctx,
		redisAddr,
		redisPassword,
		redisDb,
		marshalRow,
		unmarshalRow,
		redisAsyncChanSize,
	)
	emissionClient := client.NewClient(apiUrl)
	emissionWrapper := wrapper.New(emissionClient, time.Duration(emissionTimeoutMs)*time.Millisecond)
	emissionStorage := storage.New(ctx, lruCache, redis, emissionWrapper, updatePeriod)

	proxyHandel := handler.New(emissionStorage, apiTimeout)

	startWarmUpper(ctx, redisAddr, redisPassword, redisDb, warmupSaverPeriod, lruCache, emissionStorage)

	mux.HandleFunc("/api/v1/measure", middleware.JsonMiddleware(http.HandlerFunc(proxyHandel.Handle)).ServeHTTP)

	if err := http.ListenAndServe(":"+serverPort, mux); err != nil {
		log.Fatal().Err(err).Msg("Server crashed")
	}

	return successCode
}

func marshalRow(row schema.Row) (string, error) {
	data, err := json.Marshal(row)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unmarshalRow(s string) (schema.Row, error) {
	var row schema.Row
	err := json.Unmarshal([]byte(s), &row)
	if err != nil {
		return schema.Row{}, err
	}
	return row, nil
}
