package limiter

import (
	"sync"
	"time"
)

type Limiter struct {
	mu         sync.Mutex // guards counter, startTime and lastSeen
	limit      int
	windowSize time.Duration
	counter    int
	startTime  time.Time
	lastSeen   time.Time

	now func() time.Time
}

// New creates a fixed-window limiter that uses the real clock.
func New(limit int, windowSize time.Duration) *Limiter {
	return newWithClock(limit, windowSize, time.Now)
}

func newWithClock(limit int, windowSize time.Duration, now func() time.Time) *Limiter {
	t := now()
	return &Limiter{
		limit:      limit,
		windowSize: windowSize,
		startTime:  t,
		lastSeen:   t,
		now:        now,
	}
}

func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	l.lastSeen = now
	if l.hasElapsed(now) {
		l.counter = 0
		l.startTime = now
	}
	if l.counter < l.limit {
		l.counter++
		return true
	}
	return false
}

func (l *Limiter) LastSeen() time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lastSeen
}

// hasElapsed must be called with l.mu held. It takes `now` as an argument so
// Allow decides everything from a single reading of the clock.
func (l *Limiter) hasElapsed(now time.Time) bool {
	return now.Sub(l.startTime) > l.windowSize
}
