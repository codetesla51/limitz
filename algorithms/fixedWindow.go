package algorithms

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/codetesla51/limitz/store"
)

type FixedWindowBucket struct {
	count  int
	window int
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

	now := time.Now().Unix()
	fixedWindowData, err := fw.store.Get(key)
	var bucket *FixedWindowBucket
	if err != nil {
		bucket = &FixedWindowBucket{
			count:  0,
			window: 0,
		}
	} else {
		// Handle both MemoryStore (returns struct) and RedisStore (returns JSON string)
		switch v := fixedWindowData.(type) {
		case *FixedWindowBucket:
			bucket = v
		case string:
			bucket = &FixedWindowBucket{}
			if err := json.Unmarshal([]byte(v), bucket); err != nil {
				bucket = &FixedWindowBucket{count: 0, window: 0}
			}
		default:
			bucket = &FixedWindowBucket{count: 0, window: 0}
		}
	}
	currentWindow := int(now / int64(fw.WindowSize.Seconds()))
	if currentWindow != bucket.window {
		bucket.window = currentWindow
		bucket.count = 0
	}
	bucket.count++
	if bucket.count > fw.Limit {
		fw.store.Set(key, bucket, fw.WindowSize)
		return Result{
			Allowed:    false,
			Limit:      fw.Limit,
			Remaining:  0,
			RetryAfter: fw.WindowSize,
		}, nil
	}
	err = fw.store.Set(key, bucket, fw.WindowSize)
	if err != nil {
		return Result{}, fmt.Errorf("failed to save bucket state: %v", err)
	}

	return Result{
		Allowed:    true,
		Limit:      fw.Limit,
		Remaining:  fw.Limit - bucket.count,
		RetryAfter: 0,
	}, nil
}
func (fw *FixedWindow) Reset(key string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if !fw.store.Exists(key) {
		return fmt.Errorf("bucket for key %s does not exist", key)
	}
	return fw.store.Delete(key)
}
