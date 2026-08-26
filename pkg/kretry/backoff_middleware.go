package kretry

import (
	"math/rand/v2"
	"time"
)

// WithMaxRetries allows at most max retries through b, not counting the
// first attempt — paired with Do, WithMaxRetries(2) means up to 2 retries
// after the first attempt fails, 3 total calls to the retried function.
func (b Backoff) WithMaxRetries(max int) Backoff {
	attempt := 0
	return Backoff{
		next: func() (time.Duration, bool) {
			if attempt >= max {
				return 0, true
			}
			attempt++
			return b.next()
		},
	}
}

// WithCappedDuration clamps every delay from b to at most cap.
func (b Backoff) WithCappedDuration(cap time.Duration) Backoff {
	return Backoff{
		next: func() (time.Duration, bool) {
			delay, stop := b.next()
			if delay > cap {
				delay = cap
			}
			return delay, stop
		},
	}
}

// WithFullJitter replaces every delay from b with a random duration in
// [0, delay) — AWS's "Full Jitter" formula, which their own simulations
// found gives the lowest server load and lowest completion time among
// the common jitter strategies. Put this last in a chain, after any
// WithCappedDuration, so the cap bounds what gets jittered.
func (b Backoff) WithFullJitter() Backoff {
	return Backoff{
		next: func() (time.Duration, bool) {
			delay, stop := b.next()
			if stop || delay <= 0 {
				return delay, stop
			}
			return time.Duration(rand.Int64N(int64(delay))), stop
		},
	}
}
