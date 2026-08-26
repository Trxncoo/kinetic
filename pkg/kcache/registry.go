package kcache

import (
	"errors"
	"fmt"

	"github.com/Trxncoo/kinetic/pkg/kore"
)

// Registry holds named caches, so application wiring code can pass a
// single Registry around instead of one cache field per purpose. Built on
// kore.Registry[string].
//
// Unlike kevent.Registry, this is keyed by name, not by type: a Cache's
// (K, V) type pair isn't a unique identity the way an event type is for a
// Bus — it's entirely normal to want two different Cache[string, int]
// instances for two different purposes (say, session hit counts and
// rate-limit counters). The name is what disambiguates them; Register and
// From both also check that the stored value's type matches the (K, V)
// the caller asked for, since Go can't verify that statically across a
// map[string]any.
//
// Like kevent.Registry, this only matters at startup: Register and From
// are meant to be called once per named cache while wiring an
// application, to obtain the concrete Cache[K, V] that's then held
// directly (e.g. in a struct field) and used from then on. Get/Set/Delete
// never go through the Registry, so it adds no overhead to the hot path.
type Registry struct {
	inner *kore.Registry[string]
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{inner: kore.New[string]()}
}

// Register adds cache under name. It panics if cache is nil (including a
// typed nil pointer boxed in a Cache[K, V] value — that's not a nil
// interface, so it would otherwise be stored successfully and then panic
// with a confusing nil-pointer error the first time it's used) or if name
// is already registered — like http.ServeMux, both are wiring bugs to
// catch at startup, not runtime conditions to handle.
func Register[K comparable, V any](r *Registry, name string, cache Cache[K, V]) {
	switch err := kore.Register(r.inner, name, cache); {
	case errors.Is(err, kore.ErrNilValue):
		panic("kcache: Register called with a nil cache")
	case errors.Is(err, kore.ErrAlreadyExists):
		panic(fmt.Sprintf("kcache: cache %q already registered", name))
	}
}

// From returns the Cache registered under name. It panics if no cache was
// registered under that name, or if it was registered with a different
// (K, V) — a missing or mismatched cache is always a wiring bug (the set
// of named caches is fixed at startup, so there's no runtime condition to
// gracefully branch on), so there's no non-panicking variant.
func From[K comparable, V any](r *Registry, name string) Cache[K, V] {
	cache, err := kore.From[string, Cache[K, V]](r.inner, name)
	switch {
	case errors.Is(err, kore.ErrNotFound):
		panic(fmt.Sprintf("kcache: no cache registered for %q", name))
	case errors.Is(err, kore.ErrTypeMismatch):
		panic(fmt.Sprintf("kcache: cache %q was registered with a different type", name))
	}
	return cache
}
