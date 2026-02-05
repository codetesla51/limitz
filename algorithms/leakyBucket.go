package algorithms

import (
	"fmt"
	"sync"
	"time"
)

type LeakyBucketUser struct {
	Queue    int
	LastLeak time.Time
}

type LeakyBucket struct {
	Capacity int
	Rate     int
	Buckets  map[string]*LeakyBucketUser
	mu       sync.Mutex
}

func (lb *LeakyBucket) Allow(key string) bool {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	now := time.Now()
	bucket, exists := lb.Buckets[key]
	if !exists {
		bucket = &LeakyBucketUser{
			Queue:    0,
			LastLeak: now,
		}
		lb.Buckets[key] = bucket
	}

	elapsed := now.Sub(bucket.LastLeak).Seconds()
	bucket.LastLeak = now
	leaked := int(elapsed * float64(lb.Rate))
	bucket.Queue -= leaked
	if bucket.Queue < 0 {
		bucket.Queue = 0
	}

	if bucket.Queue < lb.Capacity {
		bucket.Queue++
		return true
	}
	return false
}

func (lb *LeakyBucket) Reset(key string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if _, exists := lb.Buckets[key]; !exists {
		return fmt.Errorf("bucket for key %s does not exist", key)
	} else {
		bucket := &LeakyBucketUser{
			Queue:    0,
			LastLeak: time.Now(),
		}
		lb.Buckets[key] = bucket
	}
	return nil
}
