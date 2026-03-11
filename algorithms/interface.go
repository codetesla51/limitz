package algorithms

import (
	"context"
	"time"
)

type Result struct {
	Allowed    bool
	Limit      int
	Remaining  int
	RetryAfter time.Duration
}
type RateLimiter interface {
	Allow(ctx context.Context, key string) (Result, error)
	Reset(ctx context.Context, key string) error
}
