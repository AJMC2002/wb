package cache

import (
	"context"
	"sync"

	"demo_service/internal/core/domain"
)

type MemoryCache struct {
	mu    sync.RWMutex
	store map[string]domain.Order
	stats *Stats
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		store: make(map[string]domain.Order),
		stats: NewStats(),
	}
}

func (c *MemoryCache) Get(_ context.Context, orderUID string) (domain.Order, bool) {
	c.mu.RLock()
	o, ok := c.store[orderUID]
	c.mu.RUnlock()

	if ok {
		c.stats.IncHit()
		return o, true
	}

	c.stats.IncMiss()
	return domain.Order{}, false
}

func (c *MemoryCache) Set(_ context.Context, order domain.Order) {
	if order.OrderUID == "" {
		return
	}
	c.mu.Lock()
	c.store[order.OrderUID] = order
	c.mu.Unlock()
}

func (c *MemoryCache) BulkSet(_ context.Context, orders []domain.Order) {
	c.mu.Lock()
	for _, o := range orders {
		if o.OrderUID == "" {
			continue
		}
		c.store[o.OrderUID] = o
	}
	c.mu.Unlock()
}

func (c *MemoryCache) Len(_ context.Context) int {
	c.mu.RLock()
	n := len(c.store)
	c.mu.RUnlock()
	return n
}

// Optional: use this later to expose cache metrics in logs/UI.
func (c *MemoryCache) Stats() (hits uint64, misses uint64) {
	return c.stats.Snapshot()
}
