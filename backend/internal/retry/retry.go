// Package retry decides how long to wait between attempts of an operation that
// can fail for temporary reasons, such as a network call.
//
// It uses exponential backoff with "full jitter": the longest possible wait
// doubles after each failure, and the actual wait is a random amount up to that
// limit. The randomness matters when many callers fail at the same moment.
// Without it they would all retry in lockstep and hit the recovering server
// together.
package retry

import (
	"context"
	"math/rand/v2"
	"time"
)

// Policy describes how many times to retry and how long to wait.
type Policy struct {
	MaxRetries int           // retries after the first attempt; 3 means up to 4 attempts
	BaseDelay  time.Duration // longest wait before the first retry
	MaxDelay   time.Duration // no single wait is ever longer than this

	// Rand returns a number in [0, 1). Tests replace it to make waits exact.
	Rand func() float64
}

// Default is the policy used for calls to CollegeFootballData (NFR-3).
func Default() Policy {
	return Policy{MaxRetries: 3, BaseDelay: 500 * time.Millisecond, MaxDelay: 8 * time.Second, Rand: rand.Float64}
}

// Delay returns how long to wait before retry number n (n starts at 1).
func (p Policy) Delay(n int) time.Duration {
	// A shift of 30 or more could overflow time.Duration's range, and any real
	// policy has hit MaxDelay long before then, so cap first.
	if n > 30 {
		return time.Duration(p.rand() * float64(p.MaxDelay))
	}
	// "<<" shifts the number's bits left; each shift by one doubles it. So the
	// limit is base, then 2x base, then 4x base, and so on for each retry.
	// With a 500 ms base: up to 0.5 s, 1 s, 2 s, then capped at MaxDelay.
	limit := p.BaseDelay << (n - 1)
	if limit > p.MaxDelay {
		limit = p.MaxDelay
	}
	return time.Duration(p.rand() * float64(limit))
}

func (p Policy) rand() float64 {
	if p.Rand == nil {
		return rand.Float64()
	}
	return p.Rand()
}

// Sleep waits for d, or returns early with the context's error if the context
// is canceled first, so a shutting-down service never waits out a backoff.
//
// Go note: a context.Context carries a deadline and a "cancel" signal through
// a chain of calls. When a request times out, or the service is told to shut
// down, the context is canceled and ctx.Done() is closed, so everything
// waiting on it can stop early. Almost every function that waits on something
// in Go takes a ctx as its first argument for this reason.
func Sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	// "defer" runs timer.Stop() when this function returns, however it
	// returns, so the timer is always cleaned up.
	defer timer.Stop()
	// "select" waits until one of its cases is ready and runs that one. Here:
	// whichever comes first, the cancel signal or the timer firing.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
