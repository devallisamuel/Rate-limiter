package limiter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Many goroutines call Allow() at the same time. Exactly `limit` of them
// may be allowed, no matter how the goroutines interleave.
func TestAllow_ConcurrentCallsRespectLimit(t *testing.T) {
	const limit = 100
	const goroutines = 1000

	l := New(limit, time.Minute)

	var allowed int64
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.Allow() {
				atomic.AddInt64(&allowed, 1)
			}
		}()
	}

	wg.Wait()

	if allowed != limit {
		t.Errorf("allowed %d requests, want exactly %d", allowed, limit)
	}
}
