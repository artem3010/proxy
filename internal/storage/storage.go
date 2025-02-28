package storage

import (
	"context"
	"proxy/internal/schema"
	"proxy/internal/storage/lru_cache"
	"time"
)

type storage struct {
	lruLocalCache   lruLocalCache[string, schema.Row]
	redisCache      redisCache[schema.Row]
	emissionService emissionClient
}

type redisRes struct {
	found    []schema.Row
	notFound []schema.Row
}

// New return storage of emissions
func New(ctx context.Context,
	lruLocalCache lruLocalCache[string, schema.Row],
	redisCache redisCache[schema.Row],
	emissionService emissionClient,
	periodHour time.Duration) *storage {
	storage := &storage{
		lruLocalCache:   lruLocalCache,
		redisCache:      redisCache,
		emissionService: emissionService,
	}

	//updater, that updates emissionsBreakdown in caches
	go storage.runDailyUpdater(ctx, periodHour)

	return storage
}

// Get return emissions by ids
func (s *storage) Get(ctx context.Context, inventoryIds map[string]schema.Row) ([]schema.Row, error) {

	keys := extractKeys(inventoryIds)

	result := make([]schema.Row, 0, len(inventoryIds))

	//getting from local in memory cache
	foundInLru, notFoundInLru := s.lruLocalCache.BatchGet(keys)
	result = append(result, foundInLru...)
	//if we found all keys returns it
	if len(notFoundInLru) == 0 {
		return result, nil
	}

	//use channels to cancel context and keep SLI
	redisCh := make(chan redisRes, 1)
	emissionCh := make(chan []schema.Row, 1)
	//if we didn't find some ok keys - gets from redis
	go s.fetchFromRedis(ctx, getRows(notFoundInLru, inventoryIds), redisCh, inventoryIds)

	var rRes redisRes
	select {
	case rRes = <-redisCh:
		result = append(result, rRes.found...)
	case <-ctx.Done():
		//async update cache for keys we didn't get
		go s.asyncUpdateCache(getRows(notFoundInLru, inventoryIds), inventoryIds)
		return result, nil
	}

	//if we found all keys returns it
	if len(rRes.notFound) == 0 {
		return result, nil
	}

	//if we didn't find some keys in redis - gets from api
	go s.fetchFromEmission(ctx, rRes.notFound, emissionCh)
	select {
	case foundEmissions := <-emissionCh:
		if foundEmissions == nil {
			//async update cache for keys we didn't get
			go s.asyncUpdateCache(rRes.notFound, inventoryIds)
			return result, nil
		}
		result = append(result, foundEmissions...)

	case <-ctx.Done():
		//async update cache for keys we didn't get
		go s.asyncUpdateCache(rRes.notFound, inventoryIds)
		return result, nil
	}
	return result, nil
}

func extractKeys(ids map[string]schema.Row) []string {
	result := make([]string, 0, len(ids))
	for k := range ids {
		result = append(result, k)
	}

	return result
}

func (s *storage) asyncUpdateCache(notFound []schema.Row, ids map[string]schema.Row) {
	//got elems to save in cache
	found, err := s.emissionService.GetEmissions(context.Background(), notFound)
	if err != nil {
		return
	}
	for i, v := range found {
		row, ok := ids[v.InventoryId]
		if !ok {
			continue
		}
		found[i].Priority = row.Priority
	}
	keys := extractKeysFromSlice(found)
	//save elems in both caches
	s.redisCache.Update(keys, found)
	s.lruLocalCache.Update(toCacheEntities(found))
}

func toCacheEntities(found []schema.Row) []lru.CacheItem[string, schema.Row] {
	result := make([]lru.CacheItem[string, schema.Row], 0, len(found))

	for _, val := range found {
		result = append(result, lru.CacheItem[string, schema.Row]{
			Key:      val.InventoryId,
			Value:    val,
			Priority: val.Priority,
		})
	}
	return result
}

func extractKeysFromSlice(found []schema.Row) []string {
	result := make([]string, 0, len(found))

	for _, val := range found {
		result = append(result, val.InventoryId)
	}

	return result
}

// gets from redis and send to chan
func (s *storage) fetchFromRedis(ctx context.Context, ids []schema.Row, out chan<- redisRes, inventoryIds map[string]schema.Row) {
	var res redisRes
	select {
	case <-ctx.Done():
		res = redisRes{found: nil, notFound: ids}
	default:
		found, notFound, err := s.redisCache.BatchGet(ctx, extractKeysFromSlice(ids))
		if err != nil {
			res = redisRes{found: nil, notFound: ids}
		} else {
			res = redisRes{found: found, notFound: getRows(notFound, inventoryIds)}
		}
	}
	out <- res
	if len(res.found) != 0 {
		update := make([]lru.CacheItem[string, schema.Row], 0, len(res.found))
		for _, val := range res.found {
			update = append(update, lru.CacheItem[string, schema.Row]{
				Key:      val.InventoryId,
				Priority: inventoryIds[val.InventoryId].Priority,
				Value: schema.Row{
					InventoryId:        val.InventoryId,
					Priority:           inventoryIds[val.InventoryId].Priority,
					EmissionsBreakdown: val.EmissionsBreakdown,
				},
			})
		}
		s.lruLocalCache.Update(update)
	}
}

func getRows(notFound []string, ids map[string]schema.Row) []schema.Row {
	result := make([]schema.Row, 0, len(notFound))

	for _, v := range notFound {
		result = append(result, ids[v])
	}

	return result
}

// gets from api and send to chan
func (s *storage) fetchFromEmission(ctx context.Context, ids []schema.Row, out chan<- []schema.Row) {
	var res []schema.Row
	select {
	case <-ctx.Done():
		res = nil
	default:
		found, err := s.emissionService.GetEmissions(ctx, ids)
		if err != nil {
			res = nil
		} else {
			res = found
		}
	}
	out <- res
}

// updater, that updates values in caches
func (s *storage) runDailyUpdater(ctx context.Context, periodHour time.Duration) {
	ticker := time.NewTicker(periodHour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			keys := s.lruLocalCache.GetValues()
			emissions, _ := s.emissionService.GetEmissions(ctx, keys)

			s.lruLocalCache.Update(toCacheEntities(emissions))
			s.redisCache.Update(extractKeysFromSlice(emissions), emissions)

		case <-ctx.Done():
			return
		}
	}
}
