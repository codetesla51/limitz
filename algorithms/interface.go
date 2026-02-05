package algorithms

import "time"

type Result struct {
	Allowed    bool
	Limit      int
	Remaining  int
	RetryAfter time.Duration
}
type RateLimiter interface {
	Allow(key string) (Result, error)
	Reset(key string) error
}
