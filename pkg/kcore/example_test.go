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
