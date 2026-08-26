package kretry_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Trxncoo/kinetic/pkg/kretry"
)

func ExampleDo() {
	backoff := kretry.NewExponential(time.Millisecond).
		WithMaxRetries(5).
		WithCappedDuration(10 * time.Millisecond).
		WithFullJitter()

	attempts := 0
	err := kretry.Do(context.Background(), backoff, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return kretry.RetryableError(fmt.Errorf("attempt %d failed", attempts))
		}
		fmt.Println("succeeded on attempt", attempts)
		return nil
	})
	if err != nil {
		fmt.Println("failed:", err)
	}

	// Output:
	// succeeded on attempt 3
}

func ExampleRetryableError() {
	backoff := kretry.NewConstant(time.Millisecond).WithMaxRetries(3)

	attempts := 0
	err := kretry.Do(context.Background(), backoff, func(ctx context.Context) error {
		attempts++
		// A 4xx-shaped error: not wrapped with RetryableError, so Do
		// stops immediately instead of retrying it.
		return errors.New("400 bad request")
	})

	fmt.Println("attempts:", attempts)
	fmt.Println("error:", err)

	// Output:
	// attempts: 1
	// error: 400 bad request
}
