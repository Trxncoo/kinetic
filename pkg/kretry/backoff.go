// Package kretry provides retry-with-backoff for flaky operations, with
// jitter and cancellation built in.
//
// Two things worth knowing that aren't separate API surface because
// they're already covered by what's here:
//
//   - A per-attempt timeout, distinct from the overall ctx passed to Do,
//     is just context.WithTimeout(ctx, ...) inside f — f already gets ctx
//     and can derive its own bound from it.
//   - Retrying at more than one layer of a call chain (client, load
//     balancer, gateway all retrying the same request) multiplies load on
//     a struggling downstream service instead of backing off it — retry
//     at one layer, or share a retry budget across them.
package kretry

import "time"

// Backoff computes the delay before each retry attempt. It's a concrete
// struct, not an interface — unlike kevent.Bus or kcache.Cache, there's
// no plausible second "backend" to swap in here, just algorithm
// composition, so a closure-wrapped struct with chainable With* methods
// gets simpler call sites for free.
//
// A Backoff is stateful: each instance (and each link in a chain) closes
// over its own attempt count. Build one fresh per Do/DoValue call — it's
// not safe to share or reuse across concurrent retry loops.
type Backoff struct {
	next func() (delay time.Duration, stop bool)
}

// Next returns the delay before the next attempt, and whether to stop
// retrying.
func (b Backoff) Next() (delay time.Duration, stop bool) {
	return b.next()
}
