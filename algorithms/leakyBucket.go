package algorithms

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/codetesla51/limitz/store"
)

type LeakyBucketUser struct {
	Queue    int
	LastLeak time.Time
}

type LeakyBucket struct {
	Capacity int
	Rate     int
	store    store.Store
	mu       sync.Mutex
}

func NewLeakyBucket(capacity, rate int, s store.Store) *LeakyBucket {
	return &LeakyBucket{
		Capacity: capacity,
		Rate:     rate,
		store:    s,
	}
}

func (lb *LeakyBucket) Allow(key string) (Result, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	now := time.Now()

	// GET bucket from store
	bucketData, err := lb.store.Get(key)
	var bucket *LeakyBucketUser
	if err != nil {
		bucket = &LeakyBucketUser{
			Queue:    0,
			LastLeak: now,
		}
	} else {
		// Handle both MemoryStore (returns struct) and RedisStore (returns JSON string)
		switch v := bucketData.(type) {
		case *LeakyBucketUser:
			bucket = v
		case string:
			bucket = &LeakyBucketUser{}
			if err := json.Unmarshal([]byte(v), bucket); err != nil {
				bucket = &LeakyBucketUser{Queue: 0, LastLeak: now}
			}
		default:
			bucket = &LeakyBucketUser{Queue: 0, LastLeak: now}
		}
	}

	// Calculate leakage
	elapsed := now.Sub(bucket.LastLeak).Seconds()
	bucket.LastLeak = now
	leaked := int(elapsed * float64(lb.Rate))
	bucket.Queue -= leaked
	if bucket.Queue < 0 {
		bucket.Queue = 0
	}

	// Check capacity
	if bucket.Queue < lb.Capacity {
		bucket.Queue++
		err := lb.store.Set(key, bucket, 1*time.Hour)
		if err != nil {
			return Result{}, fmt.Errorf("failed to save bucket state: %v", err)
		}
		return Result{
			Allowed:    true,
			Limit:      lb.Capacity,
			Remaining:  lb.Capacity - bucket.Queue,
			RetryAfter: 0,
		}, nil
	}

	err = lb.store.Set(key, bucket, 1*time.Hour)
	if err != nil {
		return Result{}, fmt.Errorf("failed to save bucket state: %v", err)
	}
	return Result{
		Allowed:    false,
		Limit:      lb.Capacity,
		Remaining:  0,
		RetryAfter: time.Duration(float64(time.Second) / float64(lb.Rate)),
	}, nil
}

func (lb *LeakyBucket) Reset(key string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	exists, err := lb.store.Exists(key)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("bucket for key %s does not exist", key)
	}

	return lb.store.Delete(key)
}
