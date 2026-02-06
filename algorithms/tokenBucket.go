package algorithms

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/codetesla51/limitz/store"
)

type Buckets struct {
	Tokens       int
	LastRefillTs time.Time
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

// Allow checks if a request is allowed using token bucket rate limiting.
func (tb *TokenBucket) Allow(key string) (Result, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	tokenBucketData, err := tb.store.Get(key)
	var bucket *Buckets
	if err != nil {
		bucket = &Buckets{
			Tokens:       tb.Capacity,
			LastRefillTs: now,
		}
	} else {
		switch v := tokenBucketData.(type) {
		case *Buckets:
			bucket = v
		case string:
			bucket = &Buckets{}
			if err := json.Unmarshal([]byte(v), bucket); err != nil {
				bucket = &Buckets{Tokens: tb.Capacity, LastRefillTs: now}
			}
		default:
			bucket = &Buckets{Tokens: tb.Capacity, LastRefillTs: now}
		}
	}

	timePassed := now.Sub(bucket.LastRefillTs)
	tokensToAdd := int(timePassed.Seconds()) * tb.RefillRate
	tokensToAdd = min(tokensToAdd, tb.Capacity-bucket.Tokens)
	bucket.Tokens += tokensToAdd
	bucket.LastRefillTs = now

	if bucket.Tokens > 0 {
		bucket.Tokens--
		err := tb.store.Set(key, bucket, 1*time.Hour)
		if err != nil {
			return Result{}, fmt.Errorf("failed to save bucket state: %v", err)
		}
		return Result{
			Allowed:    true,
			Limit:      tb.Capacity,
			Remaining:  bucket.Tokens,
			RetryAfter: 0,
		}, nil
	}

	err = tb.store.Set(key, bucket, 1*time.Hour)
	if err != nil {
		return Result{}, fmt.Errorf("failed to save bucket state: %v", err)
	}
	return Result{
		Allowed:    false,
		Limit:      tb.Capacity,
		Remaining:  bucket.Tokens,
		RetryAfter: time.Duration(float64(time.Second) / float64(tb.RefillRate)),
	}, nil
}
func (tb *TokenBucket) Reset(key string) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	exists, err := tb.store.Exists(key)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("bucket for key %s does not exist", key)
	}
	return tb.store.Delete(key)
}
