package algorithms

import (
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

func (fw *FixedWindow) Allow(key string) bool {

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
		bucket = fixedWindowData.(*FixedWindowBucket)
	}
	currentWindow := int(now / int64(fw.WindowSize.Seconds()))
	if currentWindow != bucket.window {
		bucket.window = currentWindow
		bucket.count = 0
	}
	bucket.count++
	if bucket.count > fw.Limit {
		fw.store.Set(key, bucket, fw.WindowSize)
		return false
	}
	fw.store.Set(key, bucket, fw.WindowSize)

	return true
}
func (fw *FixedWindow) Reset(key string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if !fw.store.Exists(key) {
		return fmt.Errorf("bucket for key %s does not exist", key)
	}
	return fw.store.Delete(key)
}
