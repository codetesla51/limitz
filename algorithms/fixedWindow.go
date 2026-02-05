package algorithms

import (
	"fmt"
	"sync"
	"time"
)

type FixedWindowBucket struct {
	count  int
	window int
}

type FixedWindow struct {
	Limit      int
	WindowSize time.Duration
	Buckets    map[string]*FixedWindowBucket
	mu         sync.Mutex
}

func (fw *FixedWindow) Allow(key string) bool {

	fw.mu.Lock()
	defer fw.mu.Unlock()
	bucket, exists := fw.Buckets[key]
	if !exists {
		bucket = &FixedWindowBucket{
			count:  0,
			window: 0,
		}
		fw.Buckets[key] = bucket
	}
	now := time.Now().Unix()
	currentWindow := int(now / int64(fw.WindowSize.Seconds()))
	if currentWindow != bucket.window {
		bucket.window = currentWindow
		bucket.count = 0
	}
	bucket.count++
	if bucket.count > fw.Limit {
		return false
	}
	return true
}
func (fw *FixedWindow) Reset(key string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	_, exists := fw.Buckets[key]
	if !exists {
		return fmt.Errorf("bucket for key %s does not exist", key)
	}
	bucket := &FixedWindowBucket{
		count:  0,
		window: 0,
	}
	fw.Buckets[key] = bucket
	return nil
}
