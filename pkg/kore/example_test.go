package kore_test

import (
	"errors"
	"fmt"

	"github.com/Trxncoo/kinetic/pkg/kore"
)

func ExampleRegistry() {
	r := kore.New[string]()

	if err := kore.Register(r, "greeting", "hello"); err != nil {
		fmt.Println("register failed:", err)
	}

	if v, err := kore.From[string, string](r, "greeting"); err == nil {
		fmt.Println(v)
	}

	if _, err := kore.From[string, string](r, "missing"); errors.Is(err, kore.ErrNotFound) {
		fmt.Println("missing is not registered")
	}

	// Output:
	// hello
	// missing is not registered
}
