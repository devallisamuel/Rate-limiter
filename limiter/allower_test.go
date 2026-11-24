package limiter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var implementations = map[string]func(c *fakeClock) Allower{
	"fixed window": func(c *fakeClock) Allower { return newWithClock(3, time.Minute, c.Now) },
	"token bucket": func(c *fakeClock) Allower { return newTokenBucket(3, 0.001, c.Now) },
}

func TestAllower_AllowsBurstThenDenies(t *testing.T) {
	for name, build := range implementations {
		t.Run(name, func(t *testing.T) {
			a := build(newFakeClock())

			for i := 1; i <= 3; i++ {
				if !a.Allow() {
					t.Fatalf("request %d should be allowed", i)
				}
			}
			if a.Allow() {
				t.Error("request 4 should be denied")
			}
		})
	}
}

func TestAllower_LastSeenTracksCalls(t *testing.T) {
	for name, build := range implementations {
		t.Run(name, func(t *testing.T) {
			clock := newFakeClock()
			a := build(clock)

			clock.Advance(5 * time.Second)
			a.Allow()

			if got := a.LastSeen(); !got.Equal(clock.Now()) {
				t.Errorf("LastSeen() = %v, want %v", got, clock.Now())
			}
		})
	}
}

func TestAllower_ConcurrentCallsRespectBurst(t *testing.T) {
	for name, build := range implementations {
		t.Run(name, func(t *testing.T) {
			a := build(newFakeClock())

			var allowed atomic.Int64
			var wg sync.WaitGroup
			for range 100 {
				wg.Go(func() {
					if a.Allow() {
						allowed.Add(1)
					}
				})
			}
			wg.Wait()

			if got := allowed.Load(); got != 3 {
				t.Errorf("allowed %d, want exactly 3", got)
			}
		})
	}
}

// countAllowed makes n calls and returns how many were allowed.
func countAllowed(a Allower, n int) int {
	count := 0
	for range n {
		if a.Allow() {
			count++
		}
	}
	return count
}

// TestBurstAtWindowBoundary shows WHY the token bucket exists.
//
// Both limiters are configured for "5 requests per 10 seconds". A client
// sends 5 requests at second 9 (the end of the first window), then 5 more at
// second 11 (the start of the next one).
//
//   - Fixed window: the counter reset between the two bursts, so all 10 get
//     through in 2 seconds, double the intended rate.
//   - Token bucket: only 2s x 0.5 tokens/s = 1 token refilled in between, so
//     just 6 get through in total.
func TestBurstAtWindowBoundary(t *testing.T) {
	t.Run("fixed window lets 2x the limit through", func(t *testing.T) {
		clock := newFakeClock()
		a := newWithClock(5, 10*time.Second, clock.Now)

		clock.Advance(9 * time.Second)
		first := countAllowed(a, 5)
		clock.Advance(2 * time.Second) // now at 11s: a new window has begun
		second := countAllowed(a, 5)

		if first+second != 10 {
			t.Errorf("allowed %d+%d, want 10 total (the known boundary burst)", first, second)
		}
	})

	t.Run("token bucket does not", func(t *testing.T) {
		clock := newFakeClock()
		a := newTokenBucket(5, 0.5, clock.Now) // 5 per 10s = 0.5 tokens/s

		clock.Advance(9 * time.Second)
		first := countAllowed(a, 5)
		clock.Advance(2 * time.Second)
		second := countAllowed(a, 5)

		if first+second != 6 {
			t.Errorf("allowed %d+%d, want 6 total (full bucket + 1 refilled token)", first, second)
		}
	})
}
