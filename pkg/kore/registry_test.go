package kore

import (
	"errors"
	"reflect"
	"sync"
	"testing"
)

func TestRegistry_RegisterAndFrom(t *testing.T) {
	r := New[string]()

	if err := Register(r, "a", 42); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, err := From[string, int](r, "a")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if got != 42 {
		t.Fatalf("From: got %d, want 42", got)
	}
}

func TestRegistry_FromNotFound(t *testing.T) {
	r := New[string]()

	_, err := From[string, int](r, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("From err = %v, want ErrNotFound", err)
	}
}

func TestRegistry_FromTypeMismatch(t *testing.T) {
	r := New[string]()
	if err := Register(r, "a", 42); err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, err := From[string, string](r, "a")
	if !errors.Is(err, ErrTypeMismatch) {
		t.Fatalf("From err = %v, want ErrTypeMismatch", err)
	}
}

func TestRegistry_RegisterDuplicateKey(t *testing.T) {
	r := New[string]()
	if err := Register(r, "a", 1); err != nil {
		t.Fatalf("Register: %v", err)
	}

	err := Register(r, "a", 2)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("Register err = %v, want ErrAlreadyExists", err)
	}

	// The first value must survive a failed duplicate registration.
	got, _ := From[string, int](r, "a")
	if got != 1 {
		t.Fatalf("From after failed duplicate Register: got %d, want 1", got)
	}
}

func TestRegistry_RegisterNilValue(t *testing.T) {
	r := New[string]()

	err := Register[string, *int](r, "a", nil)
	if !errors.Is(err, ErrNilValue) {
		t.Fatalf("Register err = %v, want ErrNilValue", err)
	}

	if _, err := From[string, *int](r, "a"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("From after failed nil Register: err = %v, want ErrNotFound", err)
	}
}

func TestRegistry_RegisterTypedNilValue(t *testing.T) {
	r := New[string]()
	var nilPtr *int // non-nil interface value, nil underlying pointer

	err := Register(r, "a", nilPtr)
	if !errors.Is(err, ErrNilValue) {
		t.Fatalf("Register err = %v, want ErrNilValue", err)
	}
}

func TestRegistry_RegisterValueTypeIsNeverNil(t *testing.T) {
	r := New[string]()

	// A plain struct value can never be nil, regardless of its fields.
	if err := Register(r, "a", struct{ N int }{N: 0}); err != nil {
		t.Fatalf("Register: %v", err)
	}
}

func TestRegistry_GenericOverReflectTypeKey(t *testing.T) {
	r := New[reflect.Type]()
	key := reflect.TypeFor[int]()

	if err := Register(r, key, "value for int"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	got, err := From[reflect.Type, string](r, key)
	if err != nil || got != "value for int" {
		t.Fatalf("From: got (%q, %v), want (\"value for int\", nil)", got, err)
	}
}

func TestRegistry_ConcurrentRegisterAndFrom(t *testing.T) {
	r := New[int]()

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() { _ = Register(r, i, i) })
	}
	wg.Wait()

	for i := range 50 {
		wg.Go(func() { _, _ = From[int, int](r, i) })
	}
	wg.Wait()
}
