package kevent

import (
	"context"
	"testing"
)

func benchHandler(_ context.Context, _ int) error { return nil }

func BenchmarkBus_Publish(b *testing.B) {
	bus := NewBus[int]()
	bus.Subscribe(benchHandler)
	bus.Subscribe(benchHandler)
	bus.Subscribe(benchHandler)
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		_ = bus.Publish(ctx, 1)
	}
}

func BenchmarkBus_Publish_Parallel(b *testing.B) {
	bus := NewBus[int]()
	bus.Subscribe(benchHandler)
	bus.Subscribe(benchHandler)
	bus.Subscribe(benchHandler)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = bus.Publish(ctx, 1)
		}
	})
}
