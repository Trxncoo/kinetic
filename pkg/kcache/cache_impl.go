package kcache

import (
	"hash/maphash"
	"sync"
	"time"
)

// shardCount is the sweet spot measured at
// https://strebkov.dev/posts/shard-your-locks/: roughly a 9x win over a
// single mutex at 8 cores, with sharply diminishing returns past this
// (1024 shards only +13%, 4096 +18%, for 4-16x the memory) — so it's a
// fixed constant, not a constructor option, matching NewBus's zero-config
// shape in kevent.
const shardCount = 256

type entry[V any] struct {
	value     V
	expiresAt time.Time // zero value means "never expires"
}

// shard is padded to a full 64-byte cache line (sync.Mutex is 8 bytes,
// the map header is 8 bytes, so 48 bytes of padding) so that adjacent
// shards — sitting next to each other in cache.shards — don't false-share
// a cache line when different cores lock different shards concurrently.
type shard[K comparable, V any] struct {
	mu    sync.Mutex
	items map[K]entry[V]
	_     [48]byte
}

// cache implements Cache as shardCount independently-locked shards, so
// concurrent callers touching different keys rarely contend on the same
// mutex. Unexported — callers only ever see it through the Cache
// interface NewCache returns.
type cache[K comparable, V any] struct {
	seed   maphash.Seed
	shards [shardCount]shard[K, V]
}

// NewCache creates an empty Cache for keys of type K and values of type V.
func NewCache[K comparable, V any]() Cache[K, V] {
	c := &cache[K, V]{seed: maphash.MakeSeed()}
	for i := range c.shards {
		c.shards[i].items = make(map[K]entry[V])
	}
	return c
}

// shardFor hashes key with the same algorithm Go's runtime map uses
// internally (hash/maphash.Comparable, added in Go 1.24 specifically for
// this — hashing an arbitrary comparable value generically, no reflection
// needed), then masks it down to a shard index. shardCount is a power of
// two so &(shardCount-1) is equivalent to %shardCount but cheaper.
func (c *cache[K, V]) shardFor(key K) *shard[K, V] {
	h := maphash.Comparable(c.seed, key)
	return &c.shards[h&(shardCount-1)]
}

func (c *cache[K, V]) Get(key K) (V, bool) {
	s := c.shardFor(key)

	s.mu.Lock()
	e, ok := s.items[key]
	if ok && !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		delete(s.items, key)
		ok = false
	}
	s.mu.Unlock()

	if !ok {
		var zero V
		return zero, false
	}
	return e.value, true
}

func (c *cache[K, V]) Set(key K, value V, ttl time.Duration) {
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	s := c.shardFor(key)
	s.mu.Lock()
	s.items[key] = entry[V]{value: value, expiresAt: expiresAt}
	s.mu.Unlock()
}

func (c *cache[K, V]) Delete(key K) {
	s := c.shardFor(key)
	s.mu.Lock()
	delete(s.items, key)
	s.mu.Unlock()
}

func (c *cache[K, V]) Len() int {
	n := 0
	for i := range c.shards {
		c.shards[i].mu.Lock()
		n += len(c.shards[i].items)
		c.shards[i].mu.Unlock()
	}
	return n
}

var _ Cache[string, struct{}] = (*cache[string, struct{}])(nil)
