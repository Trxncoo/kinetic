package kevent_test

import (
	"context"
	"fmt"

	"github.com/Trxncoo/kinetic/pkg/kevent"
)

type OrderPlaced struct {
	ID    string
	Total int
}

type UserSignedUp struct {
	ID string
}

func ExampleBus() {
	orders := kevent.NewBus[OrderPlaced]()

	unsubscribe := orders.Subscribe(func(_ context.Context, e OrderPlaced) error {
		fmt.Println("emailing receipt for", e.ID)
		return nil
	})
	orders.Subscribe(func(_ context.Context, e OrderPlaced) error {
		fmt.Println("recording metrics for", e.ID)
		return nil
	})

	ctx := context.Background()
	_ = orders.Publish(ctx, OrderPlaced{ID: "o1", Total: 42})

	unsubscribe()
	_ = orders.Publish(ctx, OrderPlaced{ID: "o2", Total: 10})

	// Output:
	// emailing receipt for o1
	// recording metrics for o1
	// recording metrics for o2
}

func ExampleRegistry() {
	reg := kevent.NewRegistry()

	// Wired once at startup, per event type.
	kevent.Register(reg, kevent.NewBus[OrderPlaced]())
	kevent.Register(reg, kevent.NewBus[UserSignedUp]())

	// Elsewhere, code that only knows the event type gets the right bus —
	// no need to thread individual bus fields through every constructor.
	orders := kevent.From[OrderPlaced](reg)
	orders.Subscribe(func(_ context.Context, e OrderPlaced) error {
		fmt.Println("emailing receipt for", e.ID)
		return nil
	})
	_ = orders.Publish(context.Background(), OrderPlaced{ID: "o4", Total: 7})

	// Output:
	// emailing receipt for o4
}
