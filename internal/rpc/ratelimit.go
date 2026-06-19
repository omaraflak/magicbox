package rpc

import (
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter provides per-app rate limiting for gRPC requests.
// Each app gets its own rate.Limiter, lazily created on first access.
type RateLimiter struct {
	limiters sync.Map // map[string]*rate.Limiter keyed by app_id
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{}
}

// Allow checks whether a request from the given appID should be allowed.
// Each app is limited to 50 burst with 10 requests/second refill.
func (rl *RateLimiter) Allow(appID string) bool {
	limiter, _ := rl.limiters.LoadOrStore(appID, rate.NewLimiter(rate.Limit(10), 50))
	return limiter.(*rate.Limiter).Allow()
}
