package algorithms

import (
	"testing"
	"time"

	"github.com/codetesla51/limitz/store"
)

func TestLeakyBucketAllow(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		rate     int
		requests int
		expected int
	}{
		{
			name:     "basic allow within capacity",
			capacity: 10,
			rate:     5,
			requests: 5,
			expected: 5,
		},
		{
			name:     "reject when capacity exceeded",
			capacity: 3,
			rate:     5,
			requests: 5,
			expected: 3,
		},
		{
			name:     "single request",
			capacity: 10,
			rate:     5,
			requests: 1,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.NewMemoryStore()
			lb := NewLeakyBucket(tt.capacity, tt.rate, s)

			allowed := 0
			for i := 0; i < tt.requests; i++ {
				result, err := lb.Allow("user1")
				if err == nil && result.Allowed {
					allowed++
				}
			}

			if allowed != tt.expected {
				t.Errorf("got %d, want %d", allowed, tt.expected)
			}
		})
	}
}

func TestLeakyBucketMultipleUsers(t *testing.T) {
	s := store.NewMemoryStore()
	lb := NewLeakyBucket(5, 2, s)

	// User 1 makes 3 requests
	for i := 0; i < 3; i++ {
		result, err := lb.Allow("user1")
		if err != nil || !result.Allowed {
			t.Errorf("user1 request %d should be allowed", i+1)
		}
	}

	// User 2 makes 2 requests (different user, separate bucket)
	for i := 0; i < 2; i++ {
		result, err := lb.Allow("user2")
		if err != nil || !result.Allowed {
			t.Errorf("user2 request %d should be allowed", i+1)
		}
	}

	// User 1 and 2 have independent queues (check via store)
	bucket1Data, _ := s.Get("user1")
	bucket2Data, _ := s.Get("user2")
	bucket1 := bucket1Data.(*LeakyBucketUser)
	bucket2 := bucket2Data.(*LeakyBucketUser)

	if bucket1.Queue != 3 {
		t.Errorf("user1 queue: got %d, want 3", bucket1.Queue)
	}
	if bucket2.Queue != 2 {
		t.Errorf("user2 queue: got %d, want 2", bucket2.Queue)
	}
}

func TestLeakyBucketLeakage(t *testing.T) {
	s := store.NewMemoryStore()
	lb := NewLeakyBucket(10, 10, s) // 10 requests per second

	// Fill the bucket with 5 requests
	for i := 0; i < 5; i++ {
		_, _ = lb.Allow("user1")
	}

	bucketData, _ := s.Get("user1")
	bucket := bucketData.(*LeakyBucketUser)
	if bucket.Queue != 5 {
		t.Errorf("queue before leak: got %d, want 5", bucket.Queue)
	}

	// Wait 1 second (should leak 10 requests, but only 5 exist, so goes to 0)
	time.Sleep(1 * time.Second)

	_, _ = lb.Allow("user1")

	bucketData, _ = s.Get("user1")
	bucket = bucketData.(*LeakyBucketUser)
	if bucket.Queue != 1 {
		t.Errorf("queue after 1s leak: got %d, want 1", bucket.Queue)
	}
}

func TestLeakyBucketReset(t *testing.T) {
	s := store.NewMemoryStore()
	lb := NewLeakyBucket(5, 2, s)

	// Add some requests
	for i := 0; i < 3; i++ {
		_, _ = lb.Allow("user1")
	}

	bucketData, _ := s.Get("user1")
	bucket := bucketData.(*LeakyBucketUser)
	if bucket.Queue != 3 {
		t.Errorf("queue before reset: got %d, want 3", bucket.Queue)
	}

	// Reset
	err := lb.Reset("user1")
	if err != nil {
		t.Errorf("reset failed: %v", err)
	}

	// After reset, user1 should not exist in store
	_, err = s.Get("user1")
	if err == nil {
		t.Error("after reset, user1 should not exist in store")
	}
}

func TestLeakyBucketResetNonexistent(t *testing.T) {
	s := store.NewMemoryStore()
	lb := NewLeakyBucket(5, 2, s)

	// Try to reset nonexistent user
	err := lb.Reset("nonexistent")
	if err == nil {
		t.Error("reset nonexistent should return error")
	}
}

func TestLeakyBucketConcurrency(t *testing.T) {
	s := store.NewMemoryStore()
	lb := NewLeakyBucket(100, 10, s)

	// Launch 10 goroutines, each making 10 requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				_, _ = lb.Allow("concurrent_user")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have exactly 100 in queue (at capacity)
	bucketData, _ := s.Get("concurrent_user")
	bucket := bucketData.(*LeakyBucketUser)
	if bucket.Queue != 100 {
		t.Errorf("concurrent queue: got %d, want 100", bucket.Queue)
	}
}
