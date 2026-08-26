package kevent

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestBus_DeliversToSingleSubscriber(t *testing.T) {
	bus := NewBus[int]()

	var got int
	bus.Subscribe(func(_ context.Context, e int) error {
		got = e
		return nil
	})

	if err := bus.Publish(context.Background(), 42); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if got != 42 {
		t.Fatalf("got %d, want 42", got)
	}
}

func TestBus_DeliversToMultipleSubscribersInOrder(t *testing.T) {
	bus := NewBus[int]()

	var order []int
	for i := range 3 {
		bus.Subscribe(func(_ context.Context, e int) error {
			order = append(order, i)
			return nil
		})
	}

	if err := bus.Publish(context.Background(), 1); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	want := []int{0, 1, 2}
	if len(order) != len(want) {
		t.Fatalf("got %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("got %v, want %v", order, want)
		}
	}
}

func TestBus_JoinsHandlerErrors(t *testing.T) {
	bus := NewBus[int]()
	errA := errors.New("a")
	errB := errors.New("b")

	bus.Subscribe(func(_ context.Context, e int) error { return errA })
	bus.Subscribe(func(_ context.Context, e int) error { return nil })
	bus.Subscribe(func(_ context.Context, e int) error { return errB })

	err := bus.Publish(context.Background(), 1)
	if !errors.Is(err, errA) || !errors.Is(err, errB) {
		t.Fatalf("Publish err = %v, want to wrap %v and %v", err, errA, errB)
	}
}

func TestBus_UnsubscribeStopsDelivery(t *testing.T) {
	bus := NewBus[int]()

	var calls int
	unsubscribe := bus.Subscribe(func(_ context.Context, e int) error {
		calls++
		return nil
	})

	if err := bus.Publish(context.Background(), 1); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	unsubscribe()
	if err := bus.Publish(context.Background(), 1); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestBus_PublishWithNoSubscribers(t *testing.T) {
	bus := NewBus[int]()

	if err := bus.Publish(context.Background(), 1); err != nil {
		t.Fatalf("Publish: %v, want nil", err)
	}
}

func TestBus_SubscribeNilPanics(t *testing.T) {
	bus := NewBus[int]()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic from Subscribe(nil)")
		}
	}()
	bus.Subscribe(nil)
}

func TestBus_UnsubscribeIsIdempotent(t *testing.T) {
	bus := NewBus[int]()

	var calls int
	unsubscribe := bus.Subscribe(func(_ context.Context, e int) error {
		calls++
		return nil
	})

	unsubscribe()
	unsubscribe() // must not panic, and must not disturb other subscribers

	var otherCalls int
	bus.Subscribe(func(_ context.Context, e int) error {
		otherCalls++
		return nil
	})

	if err := bus.Publish(context.Background(), 1); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if calls != 0 {
		t.Fatalf("calls = %d, want 0 (unsubscribed before any publish)", calls)
	}
	if otherCalls != 1 {
		t.Fatalf("otherCalls = %d, want 1", otherCalls)
	}
}

func TestBus_SubscribeDuringPublishNotCalledForCurrentEvent(t *testing.T) {
	bus := NewBus[int]()

	var lateCalls int
	bus.Subscribe(func(_ context.Context, e int) error {
		bus.Subscribe(func(_ context.Context, e int) error {
			lateCalls++
			return nil
		})
		return nil
	})

	if err := bus.Publish(context.Background(), 1); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if lateCalls != 0 {
		t.Fatalf("lateCalls = %d, want 0 (subscribed mid-publish, shouldn't see the in-flight event)", lateCalls)
	}

	if err := bus.Publish(context.Background(), 2); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if lateCalls != 1 {
		t.Fatalf("lateCalls = %d, want 1 (should see the next event)", lateCalls)
	}
}

func TestBus_UnsubscribeDuringPublish(t *testing.T) {
	bus := NewBus[int]()

	var selfCalls int
	var unsubscribe func()
	unsubscribe = bus.Subscribe(func(_ context.Context, e int) error {
		selfCalls++
		unsubscribe() // must not deadlock
		return nil
	})

	var otherCalls int
	bus.Subscribe(func(_ context.Context, e int) error {
		otherCalls++
		return nil
	})

	if err := bus.Publish(context.Background(), 1); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if err := bus.Publish(context.Background(), 2); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if selfCalls != 1 {
		t.Fatalf("selfCalls = %d, want 1 (unsubscribed itself after the first publish)", selfCalls)
	}
	if otherCalls != 2 {
		t.Fatalf("otherCalls = %d, want 2", otherCalls)
	}
}

func TestBus_SurvivesHandlerPanicWhenCallerRecovers(t *testing.T) {
	bus := NewBus[int]()

	unsubscribePanicker := bus.Subscribe(func(_ context.Context, e int) error {
		panic("boom")
	})

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_ = bus.Publish(context.Background(), 1)
	}()
	if recovered == nil {
		t.Fatal("expected the handler's panic to propagate to the caller")
	}
	unsubscribePanicker()

	// Publish never mutates or locks bus state, so a panic mid-dispatch
	// can't corrupt it — Subscribe/Publish work normally afterward.
	var calls int
	bus.Subscribe(func(_ context.Context, e int) error {
		calls++
		return nil
	})
	if err := bus.Publish(context.Background(), 2); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestBus_ConcurrentSubscribePublishUnsubscribe(t *testing.T) {
	bus := NewBus[int]()

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(3)
		go func() {
			defer wg.Done()
			unsubscribe := bus.Subscribe(func(_ context.Context, e int) error { return nil })
			unsubscribe()
		}()
		go func() {
			defer wg.Done()
			_ = bus.Publish(context.Background(), 1)
		}()
		go func() {
			defer wg.Done()
			bus.Subscribe(func(_ context.Context, e int) error { return nil })
		}()
	}
	wg.Wait()
}
