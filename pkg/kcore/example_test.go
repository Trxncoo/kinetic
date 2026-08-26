package kcore_test

import (
	"errors"
	"fmt"

	"github.com/Trxncoo/kinetic/pkg/kcore"
)

func ExampleRegistry() {
	r := kcore.NewRegistry[string]()

	if err := kcore.Register(r, "greeting", "hello"); err != nil {
		fmt.Println("register failed:", err)
	}

	if v, err := kcore.From[string, string](r, "greeting"); err == nil {
		fmt.Println(v)
	}

	if _, err := kcore.From[string, string](r, "missing"); errors.Is(err, kcore.ErrNotFound) {
		fmt.Println("missing is not registered")
	}

	// Output:
	// hello
	// missing is not registered
}

func ExampleRegistry_errors() {
	r := kcore.NewRegistry[string]()
	_ = kcore.Register(r, "greeting", "hello")

	if err := kcore.Register(r, "greeting", "bonjour"); errors.Is(err, kcore.ErrAlreadyExists) {
		fmt.Println("greeting is already registered")
	}

	if _, err := kcore.From[string, int](r, "greeting"); errors.Is(err, kcore.ErrTypeMismatch) {
		fmt.Println("greeting was registered with a different type")
	}

	// Output:
	// greeting is already registered
	// greeting was registered with a different type
}
