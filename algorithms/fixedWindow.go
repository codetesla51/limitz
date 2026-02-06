package algorithms

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/codetesla51/limitz/store"
)

type FixedWindowBucket struct {
	Count  int
	Window int
}

type FixedWindow struct {
	Limit      int
	WindowSize time.Duration
	store      store.Store
	mu         sync.Mutex
}

func NewFixedWindow(limit int, windowSize time.Duration, s store.Store) *FixedWindow {
	if windowSize <= 0 {
		panic("windowSize must be greater than 0")
	}
	return &FixedWindow{
		Limit:      limit,
		WindowSize: windowSize,
		store:      s,
	}
}

func (fw *FixedWindow) Allow(key string) (Result, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	nowNanos := time.Now().UnixNano()
	windowSizeNanos := fw.WindowSize.Nanoseconds()

	fixedWindowData, err := fw.store.Get(key)
	var bucket *FixedWindowBucket
	if err != nil {
		bucket = &FixedWindowBucket{
			Count:  0,
			Window: 0,
		}
	} else {
		switch v := fixedWindowData.(type) {
		case *FixedWindowBucket:
			bucket = v
		case string:
			bucket = &FixedWindowBucket{}
			if err := json.Unmarshal([]byte(v), bucket); err != nil {
				bucket = &FixedWindowBucket{Count: 0, Window: 0}
			}
		default:
			bucket = &FixedWindowBucket{Count: 0, Window: 0}
		}
	}
	currentWindow := int(nowNanos / windowSizeNanos)
	if currentWindow != bucket.Window {
		bucket.Window = currentWindow
		bucket.Count = 0
	}
	bucket.Count++
	if bucket.Count > fw.Limit {
		nextWindowStart := int64(currentWindow+1) * windowSizeNanos
		retryAfter := time.Duration(nextWindowStart-nowNanos) * time.Nanosecond

		err = fw.store.Set(key, bucket, fw.WindowSize)
		if err != nil {
			return Result{}, fmt.Errorf("failed to save bucket state: %v", err)
		}
		return Result{
			Allowed:    false,
			Limit:      fw.Limit,
			Remaining:  0,
			RetryAfter: retryAfter,
		}, nil
	}
	err = fw.store.Set(key, bucket, fw.WindowSize)
	if err != nil {
		return Result{}, fmt.Errorf("failed to save bucket state: %v", err)
	}

	return Result{
		Allowed:    true,
		Limit:      fw.Limit,
		Remaining:  fw.Limit - bucket.Count,
		RetryAfter: 0,
	}, nil
}
func (fw *FixedWindow) Reset(key string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	exists, err := fw.store.Exists(key)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("bucket for key %s does not exist", key)
	}
	return fw.store.Delete(key)
}
