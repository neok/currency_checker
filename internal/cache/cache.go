package cache

import "time"

type Cache[V any] interface {
	Get(key string) (V, bool)
	Set(key string, value V, ttl time.Duration)
}
