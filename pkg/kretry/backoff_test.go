package kretry

import (
	"math"
	"testing"
	"time"
)

func TestNewConstant(t *testing.T) {
	b := NewConstant(50 * time.Millisecond)

	for i := range 5 {
		delay, stop := b.Next()
		if stop {
			t.Fatalf("call %d: stop = true, want false", i)
		}
		if delay != 50*time.Millisecond {
			t.Fatalf("call %d: delay = %v, want 50ms", i, delay)
		}
	}
}

func TestNewExponential(t *testing.T) {
	b := NewExponential(10 * time.Millisecond)

	want := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		40 * time.Millisecond,
		80 * time.Millisecond,
	}
	for i, w := range want {
		delay, stop := b.Next()
		if stop {
			t.Fatalf("call %d: stop = true, want false", i)
		}
		if delay != w {
			t.Fatalf("call %d: delay = %v, want %v", i, delay, w)
		}
	}
}

func TestBackoff_WithMaxRetries(t *testing.T) {
	b := NewConstant(time.Millisecond).WithMaxRetries(3)

	for i := range 3 {
		if _, stop := b.Next(); stop {
			t.Fatalf("call %d: stop = true, want false", i)
		}
	}
	if _, stop := b.Next(); !stop {
		t.Fatal("call 3: stop = false, want true")
	}
}

func TestBackoff_WithMaxRetries_Zero(t *testing.T) {
	b := NewConstant(time.Millisecond).WithMaxRetries(0)

	if _, stop := b.Next(); !stop {
		t.Fatal("WithMaxRetries(0): stop = false on first call, want true")
	}
}

func TestBackoff_WithCappedDuration(t *testing.T) {
	b := NewExponential(10 * time.Millisecond).WithCappedDuration(25 * time.Millisecond)

	want := []time.Duration{
		10 * time.Millisecond, // uncapped
		20 * time.Millisecond, // uncapped
		25 * time.Millisecond, // would be 40ms, capped
		25 * time.Millisecond, // would be 80ms, capped
	}
	for i, w := range want {
		delay, _ := b.Next()
		if delay != w {
			t.Fatalf("call %d: delay = %v, want %v", i, delay, w)
		}
	}
}

func TestBackoff_WithCappedDuration_PropagatesStop(t *testing.T) {
	b := NewConstant(time.Millisecond).WithMaxRetries(1).WithCappedDuration(time.Second)

	if _, stop := b.Next(); stop {
		t.Fatal("call 0: stop = true, want false")
	}
	if _, stop := b.Next(); !stop {
		t.Fatal("call 1: stop = false, want true")
	}
}

func TestBackoff_WithFullJitter_StaysInRange(t *testing.T) {
	const delay = 100 * time.Millisecond

	for range 1000 {
		b := NewConstant(delay).WithFullJitter()
		got, stop := b.Next()
		if stop {
			t.Fatal("stop = true, want false")
		}
		if got < 0 || got >= delay {
			t.Fatalf("jittered delay = %v, want in [0, %v)", got, delay)
		}
	}
}

func TestBackoff_WithFullJitter_PropagatesStop(t *testing.T) {
	b := NewConstant(time.Millisecond).WithMaxRetries(0).WithFullJitter()

	if _, stop := b.Next(); !stop {
		t.Fatal("stop = false, want true")
	}
}

func TestBackoff_WithFullJitter_ZeroDelay(t *testing.T) {
	b := NewConstant(0).WithFullJitter()

	delay, stop := b.Next()
	if stop {
		t.Fatal("stop = true, want false")
	}
	if delay != 0 {
		t.Fatalf("delay = %v, want 0 (jittering a zero delay must not panic or go negative)", delay)
	}
}

func TestBackoff_WithFullJitter_SurvivesExponentialOverflow(t *testing.T) {
	// A base this close to time.Duration's max forces the second
	// doubling to overflow past math.MaxInt64 and wrap negative.
	// WithFullJitter must not panic (rand.Int64N panics on n <= 0).
	huge := time.Duration(math.MaxInt64/2 + 1)
	b := NewExponential(huge).WithFullJitter()

	b.Next() // huge — no overflow yet
	if _, stop := b.Next(); stop {
		t.Fatal("stop = true, want false")
	}
}

func TestBackoff_ChainOrderMatchesNesting(t *testing.T) {
	// NewExponential(...).WithMaxRetries(2).WithCappedDuration(15ms) must
	// compose the same as WithCappedDuration(15ms, WithMaxRetries(2,
	// NewExponential(...))): cap applies to every delay, max-retries
	// still stops after 2 attempts.
	b := NewExponential(10 * time.Millisecond).WithMaxRetries(2).WithCappedDuration(15 * time.Millisecond)

	delay, stop := b.Next()
	if stop || delay != 10*time.Millisecond {
		t.Fatalf("call 0: (delay, stop) = (%v, %v), want (10ms, false)", delay, stop)
	}
	delay, stop = b.Next()
	if stop || delay != 15*time.Millisecond {
		t.Fatalf("call 1: (delay, stop) = (%v, %v), want (15ms, false)", delay, stop)
	}
	if _, stop = b.Next(); !stop {
		t.Fatal("call 2: stop = false, want true")
	}
}
