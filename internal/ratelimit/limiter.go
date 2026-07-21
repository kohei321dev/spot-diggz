package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	mu         sync.Mutex
	ratePerSec float64
	burst      float64
	tokens     float64
	last       time.Time
	now        func() time.Time
}

func New(requestsPerMinute int, burst int, now func() time.Time) *Limiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 1
	}
	if burst <= 0 {
		burst = 1
	}
	if now == nil {
		now = time.Now
	}
	current := now()
	return &Limiter{
		ratePerSec: float64(requestsPerMinute) / 60,
		burst:      float64(burst),
		tokens:     float64(burst),
		last:       current,
		now:        now,
	}
}

func (limiter *Limiter) Allow() bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	current := limiter.now()
	elapsed := current.Sub(limiter.last).Seconds()
	if elapsed > 0 {
		limiter.tokens = min(limiter.burst, limiter.tokens+elapsed*limiter.ratePerSec)
		limiter.last = current
	}
	if limiter.tokens < 1 {
		return false
	}
	limiter.tokens--
	return true
}
