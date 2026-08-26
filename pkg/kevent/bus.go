// Package kevent provides a small, generic in-memory event bus.
//
// A Bus is scoped to a single event type T; compose multiple event types by
// holding multiple typed buses (see Registry) rather than routing through
// one type-erased dispatcher.
package kevent

import "context"

// Handler processes a single published event. It is a plain func type (not
// an interface) so it can be satisfied by a closure capturing outer state
// (e.g. a Kafka producer) or by a method value on a stateful type, without
// any adapter boilerplate.
type Handler[T any] func(ctx context.Context, event T) error

// Bus is satisfied by the value NewBus returns. The concrete type is
// unexported — callers construct and use it entirely through this
// interface, never by name.
type Bus[T any] interface {
	// Subscribe registers fn to run on every future Publish call and
	// returns a func to cancel that registration.
	Subscribe(fn Handler[T]) (unsubscribe func())

	// Publish runs every subscribed handler in subscription order and
	// joins their errors with errors.Join.
	Publish(ctx context.Context, event T) error
}
