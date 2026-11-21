package limiter

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

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

// Many goroutines asking for the same key must all get the same *Limiter.
// Run with -race to confirm the Store's mutex protects the map.
func TestStore_GetConcurrentSameKey(t *testing.T) {
	s := NewStore(5, time.Minute)

	const goroutines = 200
	results := make([]*Limiter, goroutines)

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = s.Get("same-key")
		}(i)
	}
	wg.Wait()

	for i, l := range results {
		if l != results[0] {
			t.Fatalf("goroutine %d got a different Limiter than goroutine 0", i)
		}
	}
}
