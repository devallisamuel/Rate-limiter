package limiter

import (
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestFakeClock_OnlyMovesWhenAdvanced(t *testing.T) {
	c := newFakeClock()
	start := c.Now()

	if !c.Now().Equal(start) {
		t.Fatal("clock moved by itself")
	}
	c.Advance(time.Hour)
	if got := c.Now().Sub(start); got != time.Hour {
		t.Errorf("after Advance(1h), clock moved by %v", got)
	}
}
