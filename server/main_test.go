package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devallisamuel/Rate-limiter/limiter"
)

func TestHealthzIsNeverRateLimited(t *testing.T) {
	handler := newHandler(limiter.NewStore(1, time.Minute))

	status := func(path string) int {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec.Code
	}

	if got := status("/"); got != http.StatusOK {
		t.Fatalf("first request to /: status = %d, want 200", got)
	}
	if got := status("/"); got != http.StatusTooManyRequests {
		t.Fatalf("second request to /: status = %d, want 429 (limiter should be exhausted)", got)
	}

	for i := range 20 {
		if got := status("/healthz"); got != http.StatusOK {
			t.Fatalf("healthz request %d: status = %d, want 200 even while rate limited", i+1, got)
		}
	}
}
