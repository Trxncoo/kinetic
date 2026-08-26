package kevent

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/Trxncoo/kinetic/pkg/kore"
)

// Registry holds one Bus per event type, so application wiring code can
// pass a single Registry around instead of one bus field per event type.
// Register and From are free functions rather than methods, because Go
// doesn't allow a method to introduce a new type parameter — Registry
// itself isn't generic over any single T, since it holds many. Built on
// kore.Registry[reflect.Type], keyed by event type since one event
// type has one canonical bus.
//
// Registry only matters at startup: Register and From are meant to be
// called once per event type while wiring an application, to obtain the
// concrete Bus[T] that's then held directly (e.g. in a struct field) and
// published to from then on. Publish never goes through the Registry, so
// it adds no overhead to the hot path.
type Registry struct {
	inner *kore.Registry[reflect.Type]
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{inner: kore.New[reflect.Type]()}
}

// Register adds bus as the Bus for event type T. It panics if bus is nil
// (including a typed nil pointer boxed in a Bus[T] value — that's not a
// nil interface, so it would otherwise be stored successfully and then
// panic with a confusing nil-pointer error the first time it's used) or if
// a bus for T is already registered — like http.ServeMux, both are wiring
// bugs to catch at startup, not runtime conditions to handle.
func Register[T any](r *Registry, bus Bus[T]) {
	t := reflect.TypeFor[T]()

	switch err := kore.Register(r.inner, t, bus); {
	case errors.Is(err, kore.ErrNilValue):
		panic("kevent: Register called with a nil bus")
	case errors.Is(err, kore.ErrAlreadyExists):
		panic(fmt.Sprintf("kevent: bus for %s already registered", t))
	}
}

// From returns the Bus registered for event type T. It panics if none was
// registered — like Register's checks, a missing bus is always a wiring
// bug (the set of registered event types is fixed at startup, so there's
// no runtime condition to gracefully branch on), so there's no
// non-panicking variant to fall back to.
func From[T any](r *Registry) Bus[T] {
	bus, err := kore.From[reflect.Type, Bus[T]](r.inner, reflect.TypeFor[T]())
	if err != nil {
		panic(fmt.Sprintf("kevent: no bus registered for %s", reflect.TypeFor[T]()))
	}
	return bus
}
