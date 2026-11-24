package limiter

import (
	"sync"
	"time"
)

// TokenBucket is a rate limiter with a bucket that holds up to `capacity`
// tokens and refills continuously at `rate` tokens per second. Each allowed
// request spends one token.
//
// WHY it exists alongside the fixed window: a fixed window resets all at once,
// so a client can send `limit` requests at the end of one window and `limit`
// more at the start of the next, up to 2x the limit in a short burst. A token
// bucket refills smoothly, so that boundary burst can't happen (see
// TestBurstAtWindowBoundary).
type TokenBucket struct {
	mu       sync.Mutex // guards tokens, last and lastSeen
	capacity float64
	rate     float64   // tokens added per second
	tokens   float64   // tokens currently in the bucket (fractions allowed)
	last     time.Time // when tokens was last refilled
	lastSeen time.Time
	now      func() time.Time // injectable clock, same idea as Limiter.now
}

// NewTokenBucket creates a full bucket that uses the real clock.
func NewTokenBucket(capacity, ratePerSecond float64) *TokenBucket {
	return newTokenBucket(capacity, ratePerSecond, time.Now)
}

func newTokenBucket(capacity, ratePerSecond float64, now func() time.Time) *TokenBucket {
	t := now()
	return &TokenBucket{
		capacity: capacity,
		rate:     ratePerSecond,
		tokens:   capacity, // start full so a new client isn't punished
		last:     t,
		lastSeen: t,
		now:      now,
	}
}

func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := b.now()
	b.lastSeen = now

	// Lazy refill: instead of a background goroutine adding tokens every
	// tick, work out how many tokens SHOULD have arrived since the last call.
	// This costs nothing while the client is quiet and needs no timers.
	if elapsed := now.Sub(b.last).Seconds(); elapsed > 0 {
		// Never exceed capacity: a client idle for an hour must not be able
		// to save up an hour's worth of tokens and spend them all at once.
		// (min is a built-in function since Go 1.21.)
		b.tokens = min(b.capacity, b.tokens+elapsed*b.rate)
		b.last = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// LastSeen is locked for the same reason as Limiter.LastSeen.
func (b *TokenBucket) LastSeen() time.Time {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastSeen
}
