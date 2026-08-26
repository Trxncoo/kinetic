package kevent

import (
	"context"
	"sync"
	"testing"
)

type orderPlaced struct{ ID string }
type userSignedUp struct{ ID string }

// fakeBus is a value-type Bus[T] implementation, used to prove Bus[T] isn't
// secretly tied to bus and to exercise isNilBus's non-pointer branch (a
// struct value can never be nil, so it should never be treated as one).
type fakeBus[T any] struct{}

func (fakeBus[T]) Subscribe(Handler[T]) func()      { return func() {} }
func (fakeBus[T]) Publish(context.Context, T) error { return nil }

func TestRegistry_RegisterAndFrom(t *testing.T) {
	reg := NewRegistry()
	orders := NewBus[orderPlaced]()

	Register(reg, orders)

	got := From[orderPlaced](reg)
	if got != Bus[orderPlaced](orders) {
		t.Fatal("From returned a different bus than was registered")
	}
}

func TestRegistry_FromPanicsWhenMissing(t *testing.T) {
	reg := NewRegistry()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic from From on missing type")
		}
	}()
	From[orderPlaced](reg)
}

func TestRegistry_DistinctTypesDontCollide(t *testing.T) {
	reg := NewRegistry()
	Register(reg, NewBus[orderPlaced]())
	Register(reg, NewBus[userSignedUp]())

	From[orderPlaced](reg) // must not panic
	From[userSignedUp](reg)
}

func TestRegistry_RegisterPanicsOnDuplicate(t *testing.T) {
	reg := NewRegistry()
	Register(reg, NewBus[orderPlaced]())

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate Register")
		}
	}()
	Register(reg, NewBus[orderPlaced]())
}

func TestRegistry_RegisterNilBusPanics(t *testing.T) {
	reg := NewRegistry()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on Register with a nil bus")
		}
	}()
	Register[orderPlaced](reg, nil)
}

func TestRegistry_RegisterTypedNilBusPanics(t *testing.T) {
	reg := NewRegistry()
	var nilBus *bus[orderPlaced] // non-nil interface value, nil underlying pointer

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on Register with a typed-nil bus")
		}
	}()
	Register[orderPlaced](reg, nilBus)
}

func TestRegistry_RegisterValueTypeBus(t *testing.T) {
	reg := NewRegistry()
	Register[orderPlaced](reg, fakeBus[orderPlaced]{})

	From[orderPlaced](reg) // must not panic
}

func TestRegistry_ConcurrentRegisterDistinctTypes(t *testing.T) {
	reg := NewRegistry()

	type a struct{ N int }
	type b struct{ N int }
	type c struct{ N int }

	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); Register(reg, NewBus[a]()) }()
	go func() { defer wg.Done(); Register(reg, NewBus[b]()) }()
	go func() { defer wg.Done(); Register(reg, NewBus[c]()) }()
	wg.Wait()

	From[a](reg) // must not panic
	From[b](reg)
	From[c](reg)
}

func TestRegistry_ConcurrentFrom(t *testing.T) {
	reg := NewRegistry()
	Register(reg, NewBus[orderPlaced]())

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = From[orderPlaced](reg)
		}()
	}
	wg.Wait()
}
