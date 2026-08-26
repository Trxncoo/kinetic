package kretry

import (
	"testing"
	"time"
)

// A light sanity check, not a comparative deep dive like kcache's — this
// package's value is correctness and composability, not raw ns/op.
func BenchmarkBackoff_Next(b *testing.B) {
	backoff := NewExponential(time.Millisecond).
		WithMaxRetries(1_000_000).
		WithCappedDuration(time.Minute).
		WithFullJitter()

	b.ReportAllocs()
	for b.Loop() {
		backoff.Next()
	}
}
