package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterConsumesBurstAndRefills(t *testing.T) {
	current := time.Date(2026, time.July, 19, 0, 0, 0, 0, time.UTC)
	limiter := New(60, 2, func() time.Time { return current })
	if !limiter.Allow() || !limiter.Allow() {
		t.Fatal("initial burst was rejected")
	}
	if limiter.Allow() {
		t.Fatal("request beyond burst was allowed")
	}
	current = current.Add(time.Second)
	if !limiter.Allow() {
		t.Fatal("refilled token was rejected")
	}
}

func TestLimiterIsSafeWhenClockMovesBackward(t *testing.T) {
	current := time.Date(2026, time.July, 19, 0, 0, 1, 0, time.UTC)
	limiter := New(60, 1, func() time.Time { return current })
	if !limiter.Allow() {
		t.Fatal("initial request was rejected")
	}
	current = current.Add(-time.Second)
	if limiter.Allow() {
		t.Fatal("backward clock movement refilled a token")
	}
}
