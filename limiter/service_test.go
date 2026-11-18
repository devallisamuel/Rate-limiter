package limiter

import (
	"testing"
	"time"
)

func TestAllow_RespectsLimit(t *testing.T) {
	l := New(3, time.Minute)

	want := []bool{true, true, true, false}

	for i, expected := range want {
		got := l.Allow()
		if got != expected {
			t.Errorf("call %d: Allow() = %v, want %v", i+1, got, expected)
		}
	}
}

// After the window expires, a full limiter should start accepting again.
func TestAllow_ResetsAfterWindow(t *testing.T) {
	l := New(2, 50*time.Millisecond)

	// Fill the window.
	l.Allow()
	l.Allow()
	if l.Allow() {
		t.Fatal("3rd call in the same window should be denied")
	}

	// Wait for the window to expire.
	time.Sleep(60 * time.Millisecond)

	if !l.Allow() {
		t.Error("first call after the window expired should be allowed")
	}
}

// The request that triggers a reset must be counted too,
// so the new window allows exactly `limit` requests, not limit+1.
func TestAllow_CountsFirstRequestAfterReset(t *testing.T) {
	l := New(2, 50*time.Millisecond)

	l.Allow()
	l.Allow()
	time.Sleep(60 * time.Millisecond)

	want := []bool{true, true, false}

	for i, expected := range want {
		got := l.Allow()
		if got != expected {
			t.Errorf("call %d after reset: Allow() = %v, want %v", i+1, got, expected)
		}
	}
}
