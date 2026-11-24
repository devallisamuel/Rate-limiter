package limiter

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMiddleware_Returns429WhenOverLimit(t *testing.T) {
	s := NewStore(2, time.Minute)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := s.Middleware(next)

	want := []int{
		http.StatusOK,
		http.StatusOK,
		http.StatusTooManyRequests,
	}

	for i, expected := range want {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != expected {
			t.Errorf("request %d: status = %d, want %d", i+1, rec.Code, expected)
		}
	}
}

func TestMiddleware_DoesNotCallNextWhenDenied(t *testing.T) {
	s := NewStore(1, time.Minute)

	calls := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
	})
	handler := s.Middleware(next)

	for i := 0; i < 3; i++ {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	}

	if calls != 1 {
		t.Errorf("next was called %d times, want 1", calls)
	}
}
