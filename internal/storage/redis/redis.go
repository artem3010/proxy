package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/go-redis/redis/v8"
)

// client common wrapper above a redis, with async saver
type client[V any] struct {
	rdb       *redis.Client
	marshal   func(V) (string, error)
	unmarshal func(string) (V, error)
	saveChan  chan redisEntity[V]
}

type redisEntity[V any] struct {
	key   string
	value V
}

// New return instance of redis wrapper
func New[V any](ctx context.Context,
	addr string,
	password string,
	db int,
	marshal func(V) (string, error),
	unmarshal func(string) (V, error),
	chanSize int) *client[V] {

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	client := &client[V]{
		rdb:       rdb,
		marshal:   marshal,
		unmarshal: unmarshal,
		saveChan:  make(chan redisEntity[V], chanSize),
	}

	//goroutine that saves elem async
	go client.runUpdater(ctx)

	return client
}

// Set put value and duration
func (c *client[V]) Set(ctx context.Context, key string, value V, expiration time.Duration) error {
	strValue, err := c.marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, strValue, expiration).Err()
}

func (c *client[V]) Get(ctx context.Context, key string) (V, error) {
	strValue, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		var zero V
		return zero, err
	}
	return c.unmarshal(strValue)
}

// SetBatch async save
func (c *client[V]) SetBatch(ctx context.Context, keys []string, values []V) {
	pipeline := c.rdb.Pipeline()
	for i, v := range values {
		pipeline.Set(ctx, keys[i], v, 24*time.Hour)
	}
	_, err := pipeline.Exec(ctx)
	if err != nil {
		log.Err(err).Msg("error during saving in redis")
		return
	}
}

// BatchGet get several values
func (c *client[V]) BatchGet(ctx context.Context, keys []string) ([]V, []string, error) {
	results, err := c.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, nil, err
	}

	var res []V
	notFound := make([]string, 0)
	for i, r := range results {
		if r == nil {
			notFound = append(notFound, keys[i])
			continue
		}
		if str, ok := r.(string); ok {
			val, err := c.unmarshal(str)
			if err != nil {
				notFound = append(notFound, keys[i])
				continue
			}
			res = append(res, val)
			continue
		}
		notFound = append(notFound, keys[i])
	}
	//TODO change by writing metrics
	log.Info().Msg(fmt.Sprintf("redis cache hit: %d, miss: %d", len(res), len(notFound)))
	return res, notFound, nil
}

func (c *client[V]) runUpdater(ctx context.Context) {
	for {
		select {
		case entity, ok := <-c.saveChan:
			if !ok {
				return
			}
			err := c.Set(ctx, entity.key, entity.value, 24*time.Hour)
			if err != nil {
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}
