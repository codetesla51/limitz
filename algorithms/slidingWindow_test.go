package algorithms

import (
	"testing"
	"time"

	"github.com/codetesla51/limitz/store"
)

func TestSlidingWindowAllow(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		requests int
		expected int
	}{
		{
			name:     "basic allow within limit",
			limit:    5,
			requests: 5,
			expected: 5,
		},
		{
			name:     "deny when limit exceeded",
			limit:    3,
			requests: 5,
			expected: 3,
		},
		{
			name:     "single request",
			limit:    10,
			requests: 1,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.NewMemoryStore()
			sw := NewSlidingWindow(tt.limit, 1*time.Second, s)

			allowed := 0
			for i := 0; i < tt.requests; i++ {
				if sw.Allow("user1") {
					allowed++
				}
			}

			if allowed != tt.expected {
				t.Errorf("got %d, want %d", allowed, tt.expected)
			}
		})
	}
}

func TestSlidingWindowMultipleUsers(t *testing.T) {
	s := store.NewMemoryStore()
	sw := NewSlidingWindow(3, 1*time.Second, s)

	// User 1 makes 3 requests
	for i := 0; i < 3; i++ {
		if !sw.Allow("user1") {
			t.Errorf("user1 request %d should be allowed", i+1)
		}
	}

	// User 2 makes 3 requests (different user, separate bucket)
	for i := 0; i < 3; i++ {
		if !sw.Allow("user2") {
			t.Errorf("user2 request %d should be allowed", i+1)
		}
	}

	// Both should have 3 timestamps in their bucket
	bucket1Data, _ := s.Get("user1")
	bucket2Data, _ := s.Get("user2")
	bucket1 := bucket1Data.(*SlidingWindowBucket)
	bucket2 := bucket2Data.(*SlidingWindowBucket)

	if len(bucket1.Timestamps) != 3 {
		t.Errorf("user1 timestamps: got %d, want 3", len(bucket1.Timestamps))
	}
	if len(bucket2.Timestamps) != 3 {
		t.Errorf("user2 timestamps: got %d, want 3", len(bucket2.Timestamps))
	}
}

func TestSlidingWindowWindowSlide(t *testing.T) {
	s := store.NewMemoryStore()
	sw := NewSlidingWindow(5, 1*time.Second, s)

	// Make 5 requests (at limit)
	for i := 0; i < 5; i++ {
		if !sw.Allow("user1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 6th request denied (at limit)
	if sw.Allow("user1") {
		t.Error("6th request should be denied")
	}

	bucketData, _ := s.Get("user1")
	bucket := bucketData.(*SlidingWindowBucket)
	if len(bucket.Timestamps) != 5 {
		t.Errorf("timestamps after denial: got %d, want 5", len(bucket.Timestamps))
	}

	time.Sleep(1 * time.Second)

	if !sw.Allow("user1") {
		t.Error("request after 1 second wait should be allowed")
	}

	bucketData, _ = s.Get("user1")
	bucket = bucketData.(*SlidingWindowBucket)
	if len(bucket.Timestamps) != 1 {
		t.Errorf("timestamps after slide: got %d, want 1", len(bucket.Timestamps))
	}
}

func TestSlidingWindowPartialSlide(t *testing.T) {
	s := store.NewMemoryStore()
	sw := NewSlidingWindow(3, 1*time.Second, s)

	// Make 3 requests at the start
	for i := 0; i < 3; i++ {
		if !sw.Allow("user1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	if sw.Allow("user1") {
		t.Error("4th request should be denied (at limit)")
	}

	time.Sleep(1500 * time.Millisecond)

	if !sw.Allow("user1") {
		t.Error("request after full window slide should be allowed")
	}

	bucketData, _ := s.Get("user1")
	bucket := bucketData.(*SlidingWindowBucket)
	// Only the new request should be there
	if len(bucket.Timestamps) != 1 {
		t.Errorf("timestamps after full slide: got %d, expected 1", len(bucket.Timestamps))
	}
}

func TestSlidingWindowReset(t *testing.T) {
	s := store.NewMemoryStore()
	sw := NewSlidingWindow(5, 1*time.Second, s)

	// Make some requests
	for i := 0; i < 3; i++ {
		sw.Allow("user1")
	}

	bucketData, _ := s.Get("user1")
	bucket := bucketData.(*SlidingWindowBucket)
	if len(bucket.Timestamps) != 3 {
		t.Errorf("timestamps before reset: got %d, want 3", len(bucket.Timestamps))
	}

	// Reset
	err := sw.Reset("user1")
	if err != nil {
		t.Errorf("reset failed: %v", err)
	}

	_, err = s.Get("user1")
	if err == nil {
		t.Error("after reset, user1 should not exist in store")
	}
}

func TestSlidingWindowResetNonexistent(t *testing.T) {
	s := store.NewMemoryStore()
	sw := NewSlidingWindow(5, 1*time.Second, s)

	err := sw.Reset("nonexistent")
	if err == nil {
		t.Error("reset nonexistent should return error")
	}
}

func TestSlidingWindowConcurrency(t *testing.T) {
	s := store.NewMemoryStore()
	sw := NewSlidingWindow(100, 1*time.Second, s)

	// Launch 10 goroutines, each making 10 requests
	done := make(chan int, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			count := 0
			for j := 0; j < 10; j++ {
				if sw.Allow("concurrent_user") {
					count++
				}
			}
			done <- count
		}(i)
	}

	// Count total allowed
	totalAllowed := 0
	for i := 0; i < 10; i++ {
		totalAllowed += <-done
	}

	if totalAllowed != 100 {
		t.Errorf("concurrent total: got %d, want 100", totalAllowed)
	}
}

func TestSlidingWindowFairness(t *testing.T) {
	// This test shows why SlidingWindow is fairer than FixedWindow
	s := store.NewMemoryStore()
	sw := NewSlidingWindow(2, 100*time.Millisecond, s)

	// Time T=0ms: Make 2 requests (at limit)
	sw.Allow("user1")
	sw.Allow("user1")

	// Time T=50ms: Wait 50ms (halfway through window)
	time.Sleep(50 * time.Millisecond)

	// Time T=50ms: Try 2 more requests (should be denied - still in same window)
	denied1 := !sw.Allow("user1")
	denied2 := !sw.Allow("user1")

	if !denied1 || !denied2 {
		t.Error("requests at 50ms should be denied (in same window)")
	}

	// Time T=100ms: Wait another 50ms (total 100ms = full window)
	time.Sleep(50 * time.Millisecond)

	// Time T=100ms: Now old requests have slid out, new ones allowed
	allowed1 := sw.Allow("user1")
	allowed2 := sw.Allow("user1")

	if !allowed1 || !allowed2 {
		t.Error("requests at 100ms should be allowed (window slid)")
	}

	// This shows fairness: you get 2 requests per 100ms on a sliding basis
	// NOT 2 at 0ms, then blocked until 100ms boundary (FixedWindow problem)
}
