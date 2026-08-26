package kretry

import "time"

// NewConstant returns a Backoff with a fixed delay that never stops on
// its own — pair it with WithMaxRetries if you want a bound.
func NewConstant(delay time.Duration) Backoff {
	return Backoff{
		next: func() (time.Duration, bool) {
			return delay, false
		},
	}
}

// NewExponential returns a Backoff that doubles the delay on every call:
// base, 2*base, 4*base, and so on. Like NewConstant, it never stops on
// its own — pair it with WithCappedDuration too, not just WithMaxRetries:
// time.Duration is an int64 count of nanoseconds, and doubling
// indefinitely will eventually overflow it (~63 doublings from 1ms) if
// nothing caps the delay first. WithFullJitter guards against the
// resulting non-positive delay rather than panicking, but backoff that's
// silently stopped backing off is still a real degradation, not a safe
// substitute for capping.
func NewExponential(base time.Duration) Backoff {
	delay := base
	started := false
	return Backoff{
		next: func() (time.Duration, bool) {
			if !started {
				started = true
				return delay, false
			}
			delay *= 2
			return delay, false
		},
	}
}
