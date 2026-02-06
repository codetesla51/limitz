package algorithms

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/codetesla51/limitz/store"
)

type SlidingWindowCounterBucket struct {
	PreviousCount int
	CurrentCount  int
	CurrentWindow int
}

type SlidingWindowCounter struct {
	Limit      int
	WindowSize time.Duration
	store      store.Store
	mu         sync.Mutex
}

func NewSlidingWindowCounter(limit int, windowSize time.Duration, s store.Store) *SlidingWindowCounter {
	if windowSize <= 0 {
		panic("windowSize must be greater than 0")
	}
	return &SlidingWindowCounter{
		Limit:      limit,
		WindowSize: windowSize,
		store:      s,
	}
}

func (swc *SlidingWindowCounter) Allow(key string) (Result, error) {
	swc.mu.Lock()
	defer swc.mu.Unlock()

	nowNanos := time.Now().UnixNano()
	windowSizeNanos := swc.WindowSize.Nanoseconds()

	currentWindow := int(nowNanos / windowSizeNanos)

	bucketData, err := swc.store.Get(key)
	var bucket *SlidingWindowCounterBucket
	if err != nil {
		bucket = &SlidingWindowCounterBucket{
			PreviousCount: 0,
			CurrentCount:  0,
			CurrentWindow: currentWindow,
		}
	} else {
		switch v := bucketData.(type) {
		case *SlidingWindowCounterBucket:
			bucket = v
		case string:
			bucket = &SlidingWindowCounterBucket{}
			if err := json.Unmarshal([]byte(v), bucket); err != nil {
				bucket = &SlidingWindowCounterBucket{
					PreviousCount: 0,
					CurrentCount:  0,
					CurrentWindow: currentWindow,
				}
			}
		default:
			bucket = &SlidingWindowCounterBucket{
				PreviousCount: 0,
				CurrentCount:  0,
				CurrentWindow: currentWindow,
			}
		}
	}

	// Did we move to a new window?
	if currentWindow != bucket.CurrentWindow {
		bucket.PreviousCount = bucket.CurrentCount
		bucket.CurrentCount = 0
		bucket.CurrentWindow = currentWindow
	}

	// How far into current window are we?
	timeIntoWindow := nowNanos % windowSizeNanos

	// How much of previous window overlaps with our sliding window?
	overlap := windowSizeNanos - timeIntoWindow
	overlapPercentage := float64(overlap) / float64(windowSizeNanos)

	// Estimate total requests in the sliding window
	estimate := float64(bucket.PreviousCount)*overlapPercentage + float64(bucket.CurrentCount)

	// Check if allowed
	if estimate < float64(swc.Limit) {
		bucket.CurrentCount++
		err := swc.store.Set(key, bucket, swc.WindowSize*2) // Store for 2 windows
		if err != nil {
			return Result{}, fmt.Errorf("failed to save bucket state: %v", err)
		}

		return Result{
			Allowed:    true,
			Limit:      swc.Limit,
			Remaining:  swc.Limit - int(estimate),
			RetryAfter: 0,
		}, nil
	}

	nextWindowStart := int64(currentWindow+1) * windowSizeNanos
	retryAfter := time.Duration(nextWindowStart-nowNanos) * time.Nanosecond

	err = swc.store.Set(key, bucket, swc.WindowSize*2)
	if err != nil {
		return Result{}, fmt.Errorf("failed to save bucket state: %v", err)
	}

	return Result{
		Allowed:    false,
		Limit:      swc.Limit,
		Remaining:  0,
		RetryAfter: retryAfter,
	}, nil
}

func (swc *SlidingWindowCounter) Reset(key string) error {
	swc.mu.Lock()
	defer swc.mu.Unlock()

	exists, err := swc.store.Exists(key)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("bucket for key %s does not exist", key)
	}

	return swc.store.Delete(key)
}
