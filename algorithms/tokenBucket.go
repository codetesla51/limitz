package algorithms

import (
	"fmt"
	"sync"
	"time"
)

type Storage interface{}
type Buckets struct {
	tokens       int
	lastRefillTs time.Time
}

type TokenBucket struct {
	Capacity   int
	RefillRate int
	Buckets    map[string]*Buckets
	mu         sync.Mutex
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// allow checks if a request is allowed using token bucket rate limiting.
func (tb *TokenBucket) Allow(key string) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	bucket, exists := tb.Buckets[key]
	if !exists {
		bucket = &Buckets{
			tokens:       tb.Capacity,
			lastRefillTs: now,
		}
		tb.Buckets[key] = bucket
	}
	// Refill tokens based on elapsed time
	timePassed := now.Sub(bucket.lastRefillTs)
	tokensToAdd := int(timePassed.Seconds()) * tb.RefillRate
	tokensToAdd = min(tokensToAdd, tb.Capacity-bucket.tokens)
	bucket.tokens += tokensToAdd
	bucket.lastRefillTs = now
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}
	return false
}
func (tb *TokenBucket) Reset(key string) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	bucket, exists := tb.Buckets[key]
	if !exists {
		return fmt.Errorf("bucket for key %s does not exist", key)
	}
	bucket.tokens = tb.Capacity
	bucket.lastRefillTs = time.Now()
	return nil
}
