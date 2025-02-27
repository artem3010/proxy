package local_lru_cache

import (
	"container/list"
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"sync"
)

type CacheItem[K comparable, V any] struct {
	Key      K
	Value    V
	Priority int
}

/*
*
Cache based on map of elements, linked list, map of priorities
when touch an elem move it on the top of linked list
when need to find an element to delete - searches for elem with max priority
*/
type LRUCache[K comparable, V any] struct {
	capacity      int
	items         map[K]*list.Element
	order         *list.List
	priorityCount map[int]int
	maxPriority   int
	mu            sync.RWMutex
	saveChan      chan CacheItem[K, V]
}

func NewLRUCache[K comparable, V any](ctx context.Context, capacity int, lruChanSize int) *LRUCache[K, V] {
	cache := &LRUCache[K, V]{
		capacity:      capacity,
		items:         make(map[K]*list.Element, capacity),
		order:         list.New(),
		priorityCount: make(map[int]int),
		maxPriority:   0,
		saveChan:      make(chan CacheItem[K, V], lruChanSize),
	}

	//goroutine to async save in cache
	go cache.runUpdater(ctx)
	return cache
}

// edit field of max priority
func (c *LRUCache[K, V]) updateMaxPriorityOnRemoval(removedPriority int) {
	c.priorityCount[removedPriority]--
	if c.priorityCount[removedPriority] == 0 {
		delete(c.priorityCount, removedPriority)
		if removedPriority == c.maxPriority {
			newMax := 0
			for prio := range c.priorityCount {
				if prio > newMax {
					newMax = prio
				}
			}
			c.maxPriority = newMax
		}
	}
}

// edit priority map
func (c *LRUCache[K, V]) updatePriorityCountOnAddition(priority int) {
	c.priorityCount[priority]++
	if priority > c.maxPriority {
		c.maxPriority = priority
	}
}

// get an elem
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	elem, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		var zero V
		return zero, false
	}
	c.mu.Lock()
	c.order.MoveToFront(elem)
	c.mu.Unlock()

	return elem.Value.(*CacheItem[K, V]).Value, true
}

// get all values
func (c *LRUCache[K, V]) GetValues() []V {
	result := make([]V, 0, len(c.items))
	c.mu.RLock()
	for _, v := range c.items {
		result = append(result, v.Value.(*CacheItem[K, V]).Value)
	}
	c.mu.RUnlock()
	return result
}

// async set value
func (c *LRUCache[K, V]) Update(rows []CacheItem[K, V]) {
	appCtx := context.Background()
	for i := range rows {
		select {
		case c.saveChan <- rows[i]:
		default:
			//if blocked make new goroutine to save
			go func(r CacheItem[K, V]) {
				select {
				case c.saveChan <- r:
				case <-appCtx.Done():
					return
				}
			}(rows[i])
		}
	}
}

// sync set value
func (c *LRUCache[K, V]) Set(key K, value V, priority int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		item := elem.Value.(*CacheItem[K, V])

		//if Priority changed we need to change our Priority map and max Priority
		if item.Priority != priority {
			c.updateMaxPriorityOnRemoval(item.Priority)
			item.Priority = priority
			c.updatePriorityCountOnAddition(priority)
		}
		item.Value = value
		c.order.MoveToFront(elem)
		return
	}

	//if our linked list fulfilled need to find an elem to delete
	if c.order.Len() == c.capacity {
		var candidate *list.Element
		for e := c.order.Back(); e != nil; e = e.Prev() {
			item := e.Value.(*CacheItem[K, V])
			//todo use spread of Priority for optimization
			if item.Priority == c.maxPriority {
				candidate = e
				break
			}
		}
		if candidate != nil {
			remItem := candidate.Value.(*CacheItem[K, V])
			delete(c.items, remItem.Key)
			c.order.Remove(candidate)
			c.updateMaxPriorityOnRemoval(remItem.Priority)
		}
	}

	newItem := &CacheItem[K, V]{
		Key:      key,
		Value:    value,
		Priority: priority}
	elem := c.order.PushFront(newItem)
	c.items[key] = elem
	c.updatePriorityCountOnAddition(priority)
}

// delete an elem
func (c *LRUCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, ok := c.items[key]; ok {
		item := elem.Value.(*CacheItem[K, V])
		c.order.Remove(elem)
		delete(c.items, key)
		c.priorityCount[item.Priority]--
		if c.priorityCount[item.Priority] == 0 {
			delete(c.priorityCount, item.Priority)
			if item.Priority == c.maxPriority {
				newMax := 0
				for prio := range c.priorityCount {
					if prio > newMax {
						newMax = prio
					}
				}
				c.maxPriority = newMax
			}
		}
	}
}

func (c *LRUCache[K, V]) BatchGet(keys []K) ([]V, []K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]V, 0, len(keys))
	notFound := make([]K, 0)
	for _, key := range keys {
		if elem, ok := c.items[key]; ok {
			c.order.MoveToFront(elem)
			result = append(result, elem.Value.(*CacheItem[K, V]).Value)
			continue
		}
		notFound = append(notFound, key)
	}
	//TODO change by writing metrics
	log.Info().Msg(fmt.Sprintf("lru cache hit: %d, miss: %d", len(result), len(notFound)))
	return result, notFound
}

func (c *LRUCache[K, V]) runUpdater(ctx context.Context) {
	for {
		select {
		case row, ok := <-c.saveChan:
			if !ok {
				// Канал закрыт, выходим из метода.
				return
			}
			c.Set(row.Key, row.Value, row.Priority)
		case <-ctx.Done():
			return
		}
	}
}
