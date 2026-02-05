package algorithms

import (
	"testing"
	"time"
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
			lb := &LeakyBucket{
				Capacity: tt.capacity,
				Rate:     tt.rate,
				Buckets:  make(map[string]*LeakyBucketUser),
			}

			allowed := 0
			for i := 0; i < tt.requests; i++ {
				if lb.Allow("user1") {
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
	lb := &LeakyBucket{
		Capacity: 5,
		Rate:     2,
		Buckets:  make(map[string]*LeakyBucketUser),
	}

	// User 1 makes 3 requests
	for i := 0; i < 3; i++ {
		if !lb.Allow("user1") {
			t.Errorf("user1 request %d should be allowed", i+1)
		}
	}

	// User 2 makes 2 requests (different user, separate bucket)
	for i := 0; i < 2; i++ {
		if !lb.Allow("user2") {
			t.Errorf("user2 request %d should be allowed", i+1)
		}
	}

	// User 1 and 2 have independent queues
	bucket1 := lb.Buckets["user1"]
	bucket2 := lb.Buckets["user2"]

	if bucket1.Queue != 3 {
		t.Errorf("user1 queue: got %d, want 3", bucket1.Queue)
	}
	if bucket2.Queue != 2 {
		t.Errorf("user2 queue: got %d, want 2", bucket2.Queue)
	}
}

func TestLeakyBucketLeakage(t *testing.T) {
	lb := &LeakyBucket{
		Capacity: 10,
		Rate:     10, // 10 requests per second
		Buckets:  make(map[string]*LeakyBucketUser),
	}

	// Fill the bucket with 5 requests
	for i := 0; i < 5; i++ {
		lb.Allow("user1")
	}

	if lb.Buckets["user1"].Queue != 5 {
		t.Errorf("queue before leak: got %d, want 5", lb.Buckets["user1"].Queue)
	}

	// Wait 1 second (should leak 10 requests, but only 5 exist, so goes to 0)
	time.Sleep(1 * time.Second)

	lb.Allow("user1")

	if lb.Buckets["user1"].Queue != 1 {
		t.Errorf("queue after 1s leak: got %d, want 1", lb.Buckets["user1"].Queue)
	}
}

func TestLeakyBucketReset(t *testing.T) {
	lb := &LeakyBucket{
		Capacity: 5,
		Rate:     2,
		Buckets:  make(map[string]*LeakyBucketUser),
	}

	// Add some requests
	for i := 0; i < 3; i++ {
		lb.Allow("user1")
	}

	if lb.Buckets["user1"].Queue != 3 {
		t.Errorf("queue before reset: got %d, want 3", lb.Buckets["user1"].Queue)
	}

	// Reset
	err := lb.Reset("user1")
	if err != nil {
		t.Errorf("reset failed: %v", err)
	}

	if lb.Buckets["user1"].Queue != 0 {
		t.Errorf("queue after reset: got %d, want 0", lb.Buckets["user1"].Queue)
	}
}

func TestLeakyBucketResetNonexistent(t *testing.T) {
	lb := &LeakyBucket{
		Capacity: 5,
		Rate:     2,
		Buckets:  make(map[string]*LeakyBucketUser),
	}

	// Try to reset nonexistent user
	err := lb.Reset("nonexistent")
	if err == nil {
		t.Error("reset nonexistent should return error")
	}
}

func TestLeakyBucketConcurrency(t *testing.T) {
	lb := &LeakyBucket{
		Capacity: 100,
		Rate:     10,
		Buckets:  make(map[string]*LeakyBucketUser),
	}

	// Launch 10 goroutines, each making 10 requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				lb.Allow("concurrent_user")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have exactly 100 in queue (at capacity)
	if lb.Buckets["concurrent_user"].Queue != 100 {
		t.Errorf("concurrent queue: got %d, want 100", lb.Buckets["concurrent_user"].Queue)
	}
}
