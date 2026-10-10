package handler

import (
	"sync"
	"time"
)

// rateLimiter counts requests per client in fixed windows of one period.
// It is safe for concurrent use.
type rateLimiter struct {
	rate   int
	period time.Duration

	mu     sync.Mutex
	window time.Time // start of the current window
	counts map[string]int
}

func newRateLimiter(rate int, period time.Duration) *rateLimiter {
	return &rateLimiter{rate: rate, period: period, counts: make(map[string]int)}
}

// allow counts a request from client at now and reports whether it may be served:
// a client is limited once it made more than rate requests in the current window
func (l *rateLimiter) allow(client string, now time.Time) bool {
	window := now.Truncate(l.period)
	l.mu.Lock()
	defer l.mu.Unlock()
	if !window.Equal(l.window) {
		l.window = window
		clear(l.counts)
	}
	if l.counts[client] > l.rate {
		return false
	}
	l.counts[client]++
	return true
}
