package kevent

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

type subscriber[T any] struct {
	id uint64
	fn Handler[T]
}

// bus implements Bus: Subscribe and Unsubscribe work at any time,
// concurrently with Publish. Publish reads a copy-on-write snapshot via
// atomic.Pointer, so it never blocks on or contends with Subscribe,
// Unsubscribe, or other Publish calls. Unexported — callers only ever see
// it through the Bus interface NewBus returns.
type bus[T any] struct {
	handlers atomic.Pointer[[]subscriber[T]]

	mu     sync.Mutex // serializes Subscribe/Unsubscribe writers only
	nextID uint64
}

// NewBus creates an empty Bus for events of type T.
func NewBus[T any]() Bus[T] {
	return &bus[T]{}
}

// Subscribe registers fn and returns a func that removes it. It panics if
// fn is nil — better a clear panic here than a confusing nil-call panic
// later, deep inside an unrelated Publish call.
func (b *bus[T]) Subscribe(fn Handler[T]) (unsubscribe func()) {
	if fn == nil {
		panic("kevent: Subscribe called with a nil handler")
	}

	b.mu.Lock()
	id := b.nextID
	b.nextID++

	old := b.handlers.Load()
	next := make([]subscriber[T], 0, len(deref(old))+1)
	next = append(next, deref(old)...)
	next = append(next, subscriber[T]{id: id, fn: fn})
	b.handlers.Store(&next)
	b.mu.Unlock()

	return func() { b.unsubscribe(id) }
}

// unsubscribe removes id, if present. Calling it a second time for the same
// id (or an id that was never registered) is a no-op: it doesn't allocate
// or touch the atomic pointer, so a double-unsubscribe costs one uncontended
// lock and a scan, not a wasted copy-on-write.
func (b *bus[T]) unsubscribe(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	old := deref(b.handlers.Load())
	idx := -1
	for i, s := range old {
		if s.id == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return
	}

	next := make([]subscriber[T], 0, len(old)-1)
	next = append(next, old[:idx]...)
	next = append(next, old[idx+1:]...)
	b.handlers.Store(&next)
}

// Publish runs every subscribed handler, in subscription order, against a
// wait-free snapshot of the handler set. Handlers run synchronously in the
// caller's goroutine and are all given the same event value — if T is a
// reference type (a pointer, slice, or map), one handler mutating it is
// visible to every handler that runs after it. A handler that panics is not
// recovered — it propagates to the caller, same as calling any other
// function. kevent doesn't guess at whether that's safe to paper over: a
// panic means something in your own code is actually broken, and the
// caller's goroutine is the one that owns the decision to recover (or not)
// at its own boundary, not this library.
func (b *bus[T]) Publish(ctx context.Context, event T) error {
	handlers := deref(b.handlers.Load())

	var errs []error
	for _, s := range handlers {
		if err := s.fn(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func deref[T any](p *[]subscriber[T]) []subscriber[T] {
	if p == nil {
		return nil
	}
	return *p
}

var _ Bus[struct{}] = (*bus[struct{}])(nil)
