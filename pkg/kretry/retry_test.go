package kretry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo_SucceedsFirstTry(t *testing.T) {
	calls := 0
	err := Do(context.Background(), NewConstant(time.Millisecond), func(context.Context) error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestDo_NilFuncPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic from Do(nil)")
		}
	}()
	_ = Do(context.Background(), NewConstant(time.Millisecond), nil)
}

func TestDo_HandlerPanicPropagates(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected f's panic to propagate out of Do")
		}
	}()
	_ = Do(context.Background(), NewConstant(time.Millisecond), func(context.Context) error {
		panic("boom")
	})
}

func TestDo_PermanentErrorStopsImmediately(t *testing.T) {
	wantErr := errors.New("bad request")
	calls := 0
	err := Do(context.Background(), NewConstant(time.Millisecond), func(context.Context) error {
		calls++
		return wantErr // not wrapped with RetryableError
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Do err = %v, want %v", err, wantErr)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1 (permanent error must not retry)", calls)
	}
}

func TestDo_RetriesUntilSuccess(t *testing.T) {
	calls := 0
	err := Do(context.Background(), NewConstant(time.Millisecond), func(context.Context) error {
		calls++
		if calls < 3 {
			return RetryableError(errors.New("transient"))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}

func TestDo_ExhaustsMaxRetries(t *testing.T) {
	wantErr := errors.New("still failing")
	calls := 0
	err := Do(context.Background(), NewConstant(time.Millisecond).WithMaxRetries(2), func(context.Context) error {
		calls++
		return RetryableError(wantErr)
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Do err = %v, want to wrap %v", err, wantErr)
	}
	if calls != 3 { // first attempt + 2 retries
		t.Fatalf("calls = %d, want 3", calls)
	}
}

func TestDo_ContextCanceledBeforeFirstAttempt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	err := Do(ctx, NewConstant(time.Millisecond), func(context.Context) error {
		calls++
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Do err = %v, want context.Canceled", err)
	}
	if calls != 0 {
		t.Fatalf("calls = %d, want 0", calls)
	}
}

func TestDo_ContextCanceledDuringBackoffWaitReturnsPromptly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := Do(ctx, NewConstant(time.Hour), func(context.Context) error {
		return RetryableError(errors.New("transient"))
	})
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Do err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed > time.Second {
		t.Fatalf("Do took %v, want well under the 1h backoff delay", elapsed)
	}
}

func TestDoValue_NilFuncPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic from DoValue(nil)")
		}
	}()
	_, _ = DoValue[int](context.Background(), NewConstant(time.Millisecond), nil)
}

func TestDoValue_ReturnsValueOnSuccess(t *testing.T) {
	calls := 0
	v, err := DoValue(context.Background(), NewConstant(time.Millisecond), func(context.Context) (string, error) {
		calls++
		if calls < 2 {
			return "", RetryableError(errors.New("transient"))
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("DoValue: %v", err)
	}
	if v != "ok" {
		t.Fatalf("DoValue value = %q, want \"ok\"", v)
	}
}

func TestDoValue_ReturnsZeroValueOnPermanentError(t *testing.T) {
	wantErr := errors.New("bad request")
	v, err := DoValue(context.Background(), NewConstant(time.Millisecond), func(context.Context) (int, error) {
		return 42, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("DoValue err = %v, want %v", err, wantErr)
	}
	if v != 0 {
		t.Fatalf("DoValue value = %d, want 0 (zero value on failure, not the discarded 42)", v)
	}
}

func TestRetryableError_Nil(t *testing.T) {
	if err := RetryableError(nil); err != nil {
		t.Fatalf("RetryableError(nil) = %v, want nil", err)
	}
}

func TestRetryableError_UnwrapsToOriginal(t *testing.T) {
	original := errors.New("original")
	wrapped := RetryableError(original)

	if !errors.Is(wrapped, original) {
		t.Fatal("errors.Is(wrapped, original) = false, want true")
	}
	if wrapped.Error() != original.Error() {
		t.Fatalf("wrapped.Error() = %q, want %q", wrapped.Error(), original.Error())
	}
}
