package limiter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// newTestStore builds a fixed-window store whose limiters AND cleanup logic
// all read the same fake clock, so tests control time completely.
func newTestStore(limit int, window time.Duration, clock *fakeClock) *Store {
	return newStore(
		func() Allower { return newWithClock(limit, window, clock.Now) },
		clock.Now,
	)
}

// Each client IP gets its own limit: exhausting one must not affect another.
func TestStore_LimitsPerClient(t *testing.T) {
	s := NewStore(2, time.Minute)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := s.Middleware(next)

	// send makes one request from the given address and returns the status code.
	send := func(remoteAddr string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = remoteAddr
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	steps := []struct {
		name string
		ip   string
		want int
	}{
		{"client A request 1", "1.1.1.1:1234", http.StatusOK},
		{"client A request 2", "1.1.1.1:1234", http.StatusOK},
		{"client A request 3 (over limit)", "1.1.1.1:1234", http.StatusTooManyRequests},
		{"client B is unaffected", "2.2.2.2:1234", http.StatusOK},
	}

	for _, step := range steps {
		if got := send(step.ip); got != step.want {
			t.Errorf("%s: status = %d, want %d", step.name, got, step.want)
		}
	}
}

// The same client IP may come from different source ports; it is still one client.
func TestStore_SameIPDifferentPortsShareLimit(t *testing.T) {
	s := NewStore(1, time.Minute)
	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	send := func(remoteAddr string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = remoteAddr
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if got := send("1.1.1.1:1111"); got != http.StatusOK {
		t.Errorf("first request: status = %d, want %d", got, http.StatusOK)
	}
	if got := send("1.1.1.1:2222"); got != http.StatusTooManyRequests {
		t.Errorf("same IP, new port: status = %d, want %d", got, http.StatusTooManyRequests)
	}
}

// Many goroutines asking for the same key must all get the same limiter.
// Run with -race to confirm the Store's mutex protects the map.
func TestStore_GetConcurrentSameKey(t *testing.T) {
	s := NewStore(5, time.Minute)

	const goroutines = 200
	results := make([]Allower, goroutines)

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = s.Get("same-key")
		}(i)
	}
	wg.Wait()

	for i, l := range results {
		if l != results[0] {
			t.Fatalf("goroutine %d got a different limiter than goroutine 0", i)
		}
	}
}

// The Store works with any algorithm, because it only talks to Allower.
func TestStore_WorksWithTokenBucket(t *testing.T) {
	clock := newFakeClock()
	s := newStore(
		func() Allower { return newTokenBucket(2, 1, clock.Now) },
		clock.Now,
	)

	a := s.Get("client")
	for i := 1; i <= 2; i++ {
		if !a.Allow() {
			t.Fatalf("request %d: a fresh token bucket should allow its capacity (2)", i)
		}
	}
	if a.Allow() {
		t.Error("third request should be denied")
	}
}

// ---- cleanup ----

// Idle clients are removed, so the map shrinks. One pass, fake clock: no
// goroutine, no waiting.
func TestStore_RemoveIdle_RemovesIdleClients(t *testing.T) {
	clock := newFakeClock()
	s := newTestStore(5, time.Minute, clock)
	s.Get("a")
	s.Get("b")
	if got := s.Len(); got != 2 {
		t.Fatalf("Len() = %d before cleanup, want 2", got)
	}

	clock.Advance(11 * time.Minute)
	s.removeIdle(10 * time.Minute)

	if got := s.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0 after both clients went idle", got)
	}
}

// A client that is still making requests must NOT be removed.
func TestStore_RemoveIdle_KeepsActiveClients(t *testing.T) {
	clock := newFakeClock()
	s := newTestStore(1000, time.Hour, clock)
	s.Get("idle")
	active := s.Get("active")

	clock.Advance(40 * time.Second)
	active.Allow() // refreshes lastSeen at t=40s
	clock.Advance(40 * time.Second)

	// At t=80s: "idle" has been silent for 80s, "active" for only 40s.
	s.removeIdle(60 * time.Second)

	if got := s.Len(); got != 1 {
		t.Errorf("Len() = %d, want 1 (only the active client should remain)", got)
	}
	if s.Get("active") != active {
		t.Error("active client was removed and recreated")
	}
}

// waitFor polls cond until it is true or the timeout passes.
//
// Used only by the ticker test below. It polls with a ticker instead of
// sleeping for a fixed time, so it returns as soon as the condition holds.
func waitFor(timeout time.Duration, cond func() bool) bool {
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	deadline := time.After(timeout)
	for {
		if cond() {
			return true
		}
		select {
		case <-tick.C:
		case <-deadline:
			return cond()
		}
	}
}

// The ONE test that uses the real clock and a real ticker: it proves the
// Cleanup loop actually calls removeIdle on each tick. All the logic of what
// gets removed is covered deterministically by the tests above.
func TestStore_CleanupRunsOnTicker(t *testing.T) {
	s := NewStore(5, time.Minute)
	s.Get("a")

	// t.Context() is cancelled automatically when the test ends, so the
	// goroutine can't leak even if the test fails.
	go s.Cleanup(t.Context(), 5*time.Millisecond, 10*time.Millisecond)

	if !waitFor(time.Second, func() bool { return s.Len() == 0 }) {
		t.Errorf("Len() = %d, want 0: Cleanup never removed the idle client", s.Len())
	}
}

// Cancelling the context must make Cleanup return (no leaked goroutine).
func TestStore_CleanupStopsOnCancel(t *testing.T) {
	s := NewStore(5, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		s.Cleanup(ctx, 10*time.Millisecond, time.Minute)
		close(done) // runs only once Cleanup has returned
	}()

	cancel()

	select {
	case <-done:
		// good: Cleanup returned
	case <-time.After(time.Second):
		t.Fatal("Cleanup did not return after the context was cancelled")
	}
}
