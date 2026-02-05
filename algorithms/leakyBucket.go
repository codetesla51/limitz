package algorithms

import (
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

func (lb *LeakyBucket) Allow(key string) bool {
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
		bucket = bucketData.(*LeakyBucketUser)
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
		lb.store.Set(key, bucket, 1*time.Hour)
		return true
	}

	lb.store.Set(key, bucket, 1*time.Hour)
	return false
}

func (lb *LeakyBucket) Reset(key string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if !lb.store.Exists(key) {
		return fmt.Errorf("bucket for key %s does not exist", key)
	}

	return lb.store.Delete(key)
}
