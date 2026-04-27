package cache

import (
	"sync"
	"time"
)

type entry[V any] struct {
	value     V
	expiresAt time.Time
}

type InMemoryCache[V any] struct {
	mu      sync.RWMutex
	entries map[string]entry[V]
	now     func() time.Time
}

func NewInMemoryCache[V any]() *InMemoryCache[V] {
	return &InMemoryCache[V]{
		entries: make(map[string]entry[V]),
		now:     time.Now,
	}
}

func (c *InMemoryCache[V]) Get(key string) (V, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok || c.now().After(e.expiresAt) {
		var zero V
		return zero, false
	}
	return e.value, true
}

func (c *InMemoryCache[V]) Set(key string, value V, ttl time.Duration) {
	c.mu.Lock()
	c.entries[key] = entry[V]{value: value, expiresAt: c.now().Add(ttl)}
	c.mu.Unlock()
}
