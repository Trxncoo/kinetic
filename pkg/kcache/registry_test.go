package kcache

import (
	"sync"
	"testing"
	"time"
)

// fakeCache is a value-type Cache[K, V] implementation, used to prove
// Cache[K, V] isn't secretly tied to the sharded impl and to exercise
// kcore.isNil's non-pointer branch (a struct value can never be nil, so it
// should never be treated as one).
type fakeCache[K comparable, V any] struct{}

func (fakeCache[K, V]) Get(K) (V, bool)         { var zero V; return zero, false }
func (fakeCache[K, V]) Set(K, V, time.Duration) {}
func (fakeCache[K, V]) Delete(K)                {}
func (fakeCache[K, V]) Len() int                { return 0 }

func TestRegistry_RegisterAndFrom(t *testing.T) {
	reg := NewRegistry()
	sessions := NewCache[string, string]()

	Register(reg, "sessions", sessions)

	got := From[string, string](reg, "sessions")
	if got != sessions {
		t.Fatal("From returned a different cache than was registered")
	}
}

func TestRegistry_FromPanicsWhenMissing(t *testing.T) {
	reg := NewRegistry()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic from From on missing name")
		}
	}()
	From[string, string](reg, "sessions")
}

func TestRegistry_FromPanicsOnTypeMismatch(t *testing.T) {
	reg := NewRegistry()
	Register(reg, "sessions", NewCache[string, int]())

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic from From with mismatched (K, V)")
		}
	}()
	From[string, string](reg, "sessions")
}

func TestRegistry_SameTypesDifferentNamesDontCollide(t *testing.T) {
	reg := NewRegistry()
	Register(reg, "sessions", NewCache[string, int]())
	Register(reg, "rate-limits", NewCache[string, int]())

	From[string, int](reg, "sessions") // must not panic
	From[string, int](reg, "rate-limits")
}

func TestRegistry_RegisterPanicsOnDuplicateName(t *testing.T) {
	reg := NewRegistry()
	Register(reg, "sessions", NewCache[string, int]())

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate Register")
		}
	}()
	Register(reg, "sessions", NewCache[string, int]())
}

func TestRegistry_RegisterNilCachePanics(t *testing.T) {
	reg := NewRegistry()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on Register with a nil cache")
		}
	}()
	Register[string, int](reg, "sessions", nil)
}

func TestRegistry_RegisterTypedNilCachePanics(t *testing.T) {
	reg := NewRegistry()
	var nilCache *cache[string, int] // non-nil interface value, nil underlying pointer

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on Register with a typed-nil cache")
		}
	}()
	Register(reg, "sessions", nilCache)
}

func TestRegistry_RegisterValueTypeCache(t *testing.T) {
	reg := NewRegistry()
	Register(reg, "sessions", fakeCache[string, int]{})

	From[string, int](reg, "sessions") // must not panic
}

func TestRegistry_ConcurrentRegisterDistinctNames(t *testing.T) {
	reg := NewRegistry()

	var wg sync.WaitGroup
	wg.Go(func() { Register(reg, "a", NewCache[string, int]()) })
	wg.Go(func() { Register(reg, "b", NewCache[string, int]()) })
	wg.Go(func() { Register(reg, "c", NewCache[string, int]()) })
	wg.Wait()

	From[string, int](reg, "a") // must not panic
	From[string, int](reg, "b")
	From[string, int](reg, "c")
}

func TestRegistry_ConcurrentFrom(t *testing.T) {
	reg := NewRegistry()
	Register(reg, "sessions", NewCache[string, int]())

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			From[string, int](reg, "sessions")
		})
	}
	wg.Wait()
}
