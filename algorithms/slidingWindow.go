package algorithms

import (
	"fmt"
	"sync"
	"time"

	"github.com/codetesla51/limitz/store"
)

// SlidingWindowBucket stores a list of request timestamps for a user
type SlidingWindowBucket struct {
	Timestamps []int64
}

type SlidingWindow struct {
	Limit      int           // Max requests allowed
	WindowSize time.Duration // How long to track (e.g., 1 minute)
	store      store.Store   // Where to persist request timestamps
	mu         sync.Mutex
}

func NewSlidingWindow(limit int, windowSize time.Duration, s store.Store) *SlidingWindow {
	if windowSize <= 0 {
		panic("WindowSize must be greater than 0")
	}
	return &SlidingWindow{
		Limit:      limit,
		WindowSize: windowSize,
		store:      s,
	}
}

// Allow checks if a request is allowed under sliding window rate limit
func (sw *SlidingWindow) Allow(key string) bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now().UnixNano()
	windowStart := now - sw.WindowSize.Nanoseconds()

	bucketData, err := sw.store.Get(key)
	var bucket *SlidingWindowBucket
	if err != nil {
		bucket = &SlidingWindowBucket{
			Timestamps: []int64{},
		}
	} else {
		bucket = bucketData.(*SlidingWindowBucket)
	}

	validTimestamps := []int64{}
	for _, ts := range bucket.Timestamps {
		if ts > windowStart {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	bucket.Timestamps = validTimestamps

	if len(bucket.Timestamps) < sw.Limit {
		bucket.Timestamps = append(bucket.Timestamps, now)
		sw.store.Set(key, bucket, sw.WindowSize)
		return true
	}

	sw.store.Set(key, bucket, sw.WindowSize)
	return false
}

func (sw *SlidingWindow) Reset(key string) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if !sw.store.Exists(key) {
		return fmt.Errorf("bucket for key %s does not exist", key)
	}

	return sw.store.Delete(key)
}
