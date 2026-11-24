package limiter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenBucket_StartsFull(t *testing.T) {
	clock := newFakeClock()
	b := newTokenBucket(3, 1, clock.Now)

	want := []bool{true, true, true, false}
	for i, expected := range want {
		if got := b.Allow(); got != expected {
			t.Errorf("call %d: Allow() = %v, want %v", i+1, got, expected)
		}
	}
}

func TestTokenBucket_RefillsOverTime(t *testing.T) {
	clock := newFakeClock()
	b := newTokenBucket(2, 1, clock.Now)

	b.Allow()
	b.Allow()
	if b.Allow() {
		t.Fatal("bucket should be empty after spending its 2 tokens")
	}

	clock.Advance(time.Second)
	if !b.Allow() {
		t.Error("expected one request after 1s of refill")
	}
	if b.Allow() {
		t.Error("only one token should have been refilled")
	}
}

func TestTokenBucket_FractionalRefill(t *testing.T) {
	clock := newFakeClock()
	b := newTokenBucket(1, 0.5, clock.Now)

	b.Allow() // empty the bucket

	clock.Advance(time.Second) // 0.5 tokens: not enough
	if b.Allow() {
		t.Error("half a token should not allow a request")
	}

	clock.Advance(time.Second)
	if !b.Allow() {
		t.Error("a full token should allow a request")
	}
}

func TestTokenBucket_RefillIsCappedAtCapacity(t *testing.T) {
	clock := newFakeClock()
	b := newTokenBucket(3, 1, clock.Now)

	for range 3 {
		b.Allow()
	}
	clock.Advance(time.Hour)

	allowed := 0
	for range 10 {
		if b.Allow() {
			allowed++
		}
	}
	if allowed != 3 {
		t.Errorf("allowed %d after a long idle, want exactly the capacity (3)", allowed)
	}
}

func TestTokenBucket_ConcurrentCalls(t *testing.T) {
	clock := newFakeClock() // frozen, so no refill can muddy the count
	b := newTokenBucket(100, 1, clock.Now)

	var allowed atomic.Int64
	var wg sync.WaitGroup
	for range 1000 {
		wg.Go(func() {
			if b.Allow() {
				allowed.Add(1)
			}
		})
	}
	wg.Wait()

	if got := allowed.Load(); got != 100 {
		t.Errorf("allowed %d, want exactly 100", got)
	}
}
