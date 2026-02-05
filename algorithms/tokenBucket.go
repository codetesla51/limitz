package algorithms

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/codetesla51/limitz/store"
)

type Buckets struct {
	tokens       int
	lastRefillTs time.Time
}

type TokenBucket struct {
	Capacity   int
	RefillRate int
	store      store.Store
	mu         sync.Mutex
}

func NewTokenBucket(capacity, refillRate int, s store.Store) *TokenBucket {
	return &TokenBucket{
		Capacity:   capacity,
		RefillRate: refillRate,
		store:      s,
	}
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// allow checks if a request is allowed using token bucket rate limiting.
func (tb *TokenBucket) Allow(key string) (Result, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	tokenBucketData, err := tb.store.Get(key)
	var bucket *Buckets
	if err != nil {
		bucket = &Buckets{
			tokens:       tb.Capacity,
			lastRefillTs: now,
		}
	} else {
		// Handle both MemoryStore (returns struct) and RedisStore (returns JSON string)
		switch v := tokenBucketData.(type) {
		case *Buckets:
			bucket = v
		case string:
			bucket = &Buckets{}
			if err := json.Unmarshal([]byte(v), bucket); err != nil {
				bucket = &Buckets{tokens: tb.Capacity, lastRefillTs: now}
			}
		default:
			bucket = &Buckets{tokens: tb.Capacity, lastRefillTs: now}
		}
	}
	// Refill tokens based on elapsed time
	timePassed := now.Sub(bucket.lastRefillTs)
	tokensToAdd := int(timePassed.Seconds()) * tb.RefillRate
	tokensToAdd = min(tokensToAdd, tb.Capacity-bucket.tokens)
	bucket.tokens += tokensToAdd
	bucket.lastRefillTs = now

	// Check capacity
	if bucket.tokens > 0 {
		bucket.tokens--
		// SAVE back to store
		err := tb.store.Set(key, bucket, 1*time.Hour)
		if err != nil {
			return Result{}, fmt.Errorf("failed to save bucket state: %v", err)
		}
		return Result{
			Allowed:    true,
			Limit:      tb.Capacity,
			Remaining:  bucket.tokens,
			RetryAfter: 0,
		}, nil

	}

	// SAVE even if denied
	err = tb.store.Set(key, bucket, 1*time.Hour)
	if err != nil {
		return Result{}, fmt.Errorf("failed to save bucket state: %v", err)
	}
	return Result{
		Allowed:    false,
		Limit:      tb.Capacity,
		Remaining:  bucket.tokens,
		RetryAfter: time.Duration(float64(time.Second) / float64(tb.RefillRate)),
	}, nil
}
func (tb *TokenBucket) Reset(key string) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if !tb.store.Exists(key) {
		return fmt.Errorf("bucket for key %s does not exist", key)
	}
	return tb.store.Delete(key)
}
