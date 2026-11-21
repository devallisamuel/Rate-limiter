package limiter

import (
	"sync"
	"time"
)

type Limiter struct {
	mu         sync.Mutex // guards counter and startTime
	limit      int
	windowSize time.Duration
	counter    int
	startTime  time.Time
}

func New(limit int, windowSize time.Duration) *Limiter {
	return &Limiter{
		limit:      limit,
		windowSize: windowSize,
		startTime:  time.Now(),
	}
}

func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.hasElapsed() {
		l.counter = 0
		l.startTime = time.Now()
	}
	if l.counter < l.limit {
		l.counter++
		return true
	}
	return false
}

func (l *Limiter) hasElapsed() bool {
	return time.Since(l.startTime) > l.windowSize
}
