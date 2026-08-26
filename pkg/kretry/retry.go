package kretry

import (
	"context"
	"errors"
	"time"
)

// RetryFunc is an operation Do retries on failure.
type RetryFunc func(ctx context.Context) error

// RetryFuncValue is an operation DoValue retries on failure, producing a
// value on success.
type RetryFuncValue[T any] func(ctx context.Context) (T, error)

type retryable struct {
	err error
}

func (r *retryable) Error() string { return r.err.Error() }
func (r *retryable) Unwrap() error { return r.err }

// RetryableError marks err as worth retrying. Do and DoValue treat any
// error NOT wrapped this way as permanent and stop immediately — opt-in
// to retry, not opt-out, so a caller can't accidentally retry a
// non-idempotent or genuinely permanent failure just because they forgot
// to special-case it. errors.Is and errors.As still reach the original
// err through the wrapper.
func RetryableError(err error) error {
	if err == nil {
		return nil
	}
	return &retryable{err: err}
}

// Do calls f, retrying it with b's delays for as long as f returns an
// error wrapped with RetryableError. It returns nil on the first success;
// the most recent error once f returns a non-retryable error or b stops
// retrying; or ctx's error if ctx is canceled before an attempt or during
// a retry wait (waiting never blocks past ctx's cancellation). If f
// panics, the panic is not recovered — it propagates to Do's caller, same
// as kevent.Bus.Publish: a panic means something in f is actually broken,
// not a condition to retry past.
func Do(ctx context.Context, b Backoff, f RetryFunc) error {
	if f == nil {
		panic("kretry: Do called with a nil f")
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := f(ctx)
		if err == nil {
			return nil
		}

		if _, ok := errors.AsType[*retryable](err); !ok {
			return err
		}

		delay, stop := b.Next()
		if stop {
			return err
		}

		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	}
}

// DoValue is Do for an operation that produces a value on success.
func DoValue[T any](ctx context.Context, b Backoff, f RetryFuncValue[T]) (T, error) {
	if f == nil {
		panic("kretry: DoValue called with a nil f")
	}

	var result T
	err := Do(ctx, b, func(ctx context.Context) error {
		v, err := f(ctx)
		if err != nil {
			return err
		}
		result = v
		return nil
	})
	return result, err
}
