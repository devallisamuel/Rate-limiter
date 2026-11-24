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
	clock := newFakeClock()
	l := newWithClock(2, time.Minute, clock.Now)

	// Fill the window.
	l.Allow()
	l.Allow()
	if l.Allow() {
		t.Fatal("3rd call in the same window should be denied")
	}

	// A minute and a bit "passes" instantly: no time.Sleep needed.
	clock.Advance(time.Minute + time.Second)

	if !l.Allow() {
		t.Error("first call after the window expired should be allowed")
	}
}

// The window must NOT reset early.
func TestAllow_DoesNotResetBeforeWindowEnds(t *testing.T) {
	clock := newFakeClock()
	l := newWithClock(1, time.Minute, clock.Now)

	l.Allow()
	clock.Advance(59 * time.Second)

	if l.Allow() {
		t.Error("call 59s into a 60s window should still be denied")
	}
}

// The request that triggers a reset must be counted too,
// so the new window allows exactly `limit` requests, not limit+1.
func TestAllow_CountsFirstRequestAfterReset(t *testing.T) {
	clock := newFakeClock()
	l := newWithClock(2, time.Minute, clock.Now)

	l.Allow()
	l.Allow()
	clock.Advance(time.Minute + time.Second)

	want := []bool{true, true, false}

	for i, expected := range want {
		got := l.Allow()
		if got != expected {
			t.Errorf("call %d after reset: Allow() = %v, want %v", i+1, got, expected)
		}
	}
}
