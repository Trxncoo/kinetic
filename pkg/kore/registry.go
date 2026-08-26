// Package kore provides a small, generic, concurrency-safe registry:
// one value per key, looked up by its static type. It's the shared
// mechanics behind kevent.Registry and kcache.Registry — each wraps a
// Registry keyed on whatever makes sense for it (kevent: reflect.Type,
// since one event type has one canonical bus; kcache: string, since a
// (K, V) pair isn't a unique identity for a cache) and layers its own
// panic-on-misuse policy on top of the errors this package returns.
package kore

import (
	"errors"
	"reflect"
	"sync"
)

var (
	// ErrNilValue is returned by Register when value is nil, including a
	// typed nil pointer boxed in a non-nil interface — that's not a nil
	// interface, so it would otherwise be stored successfully and then
	// fail with a confusing nil-pointer error the first time it's used.
	ErrNilValue = errors.New("kore: nil value")

	// ErrAlreadyExists is returned by Register when key is already
	// registered.
	ErrAlreadyExists = errors.New("kore: key already registered")

	// ErrNotFound is returned by From when key has nothing registered.
	ErrNotFound = errors.New("kore: key not found")

	// ErrTypeMismatch is returned by From when key has a value
	// registered, but not of the requested type V.
	ErrTypeMismatch = errors.New("kore: value has a different type")
)

// Registry holds one value per key, safe for concurrent use. Meant for
// startup-time wiring, not a hot path: Register and From take a lock, and
// From does a type assertion on every call. Callers are expected to call
// From once per key and hold onto the concrete value from then on.
type Registry[Key comparable] struct {
	mu    sync.RWMutex
	items map[Key]any
}

// New creates an empty Registry.
func New[Key comparable]() *Registry[Key] {
	return &Registry[Key]{items: make(map[Key]any)}
}

// Register adds value under key. It returns ErrNilValue or
// ErrAlreadyExists without modifying the Registry if either applies;
// both checks happen inside one critical section, so a concurrent
// Register for the same key can't make a plain nil-value call
// misreport as a duplicate (or vice versa).
func Register[Key comparable, V any](r *Registry[Key], key Key, value V) error {
	if isNil(value) {
		return ErrNilValue
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[key]; exists {
		return ErrAlreadyExists
	}
	r.items[key] = value
	return nil
}

// From returns the value registered under key. It returns ErrNotFound if
// key has nothing registered, or ErrTypeMismatch if it has a value of a
// different type than V.
func From[Key comparable, V any](r *Registry[Key], key Key) (V, error) {
	r.mu.RLock()
	v, exists := r.items[key]
	r.mu.RUnlock()

	var zero V
	if !exists {
		return zero, ErrNotFound
	}
	val, ok := v.(V)
	if !ok {
		return zero, ErrTypeMismatch
	}
	return val, nil
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
