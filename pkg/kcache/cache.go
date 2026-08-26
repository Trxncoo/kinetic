// Package kcache provides a small, generic, sharded in-memory cache.
//
// Compose multiple named caches by holding multiple typed Cache values,
// or via Registry — unlike kevent's Registry, kcache's is keyed by name
// rather than by type, since a (K, V) pair isn't a unique identity for a
// cache the way an event type is for a Bus.
package kcache

import "time"

// Cache is satisfied by the value NewCache returns. The concrete type is
// unexported — callers construct and use it entirely through this
// interface, never by name.
type Cache[K comparable, V any] interface {
	// Get returns the value stored for key, and whether it was found. A
	// key whose TTL has passed is treated as a miss.
	Get(key K) (V, bool)

	// Set stores value for key. ttl <= 0 means the entry never expires —
	// the zero value is the "no special behavior" case, matching Go's
	// usual zero-value conventions.
	Set(key K, value V, ttl time.Duration)

	// Delete removes key, if present. Deleting a missing key is a no-op.
	Delete(key K)

	// Len returns the number of entries currently stored, including any
	// that have expired but haven't been touched (by Get or Delete) yet.
	// Under concurrent writes this is a point-in-time approximation, not
	// a snapshot — it sums per-shard counts taken one shard at a time.
	Len() int
}
