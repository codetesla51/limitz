package algorithms

import (
	"testing"
	"time"

	"github.com/codetesla51/limitz/store"
)

func TestSlidingWindowCounterAllow(t *testing.T) {
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
		{
			name:     "zero requests",
			limit:    5,
			requests: 0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.NewMemoryStore()
			swc := NewSlidingWindowCounter(tt.limit, 1*time.Second, s)

			allowed := 0
			for i := 0; i < tt.requests; i++ {
				result, err := swc.Allow("user1")
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

func TestSlidingWindowCounterMultipleUsers(t *testing.T) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(3, 1*time.Second, s)

	for i := 0; i < 3; i++ {
		result, err := swc.Allow("user1")
		if err != nil || !result.Allowed {
			t.Errorf("user1 request %d should be allowed", i+1)
		}
	}

	for i := 0; i < 3; i++ {
		result, err := swc.Allow("user2")
		if err != nil || !result.Allowed {
			t.Errorf("user2 request %d should be allowed", i+1)
		}
	}

	bucket1Data, _ := s.Get("user1")
	bucket2Data, _ := s.Get("user2")
	bucket1 := bucket1Data.(*SlidingWindowCounterBucket)
	bucket2 := bucket2Data.(*SlidingWindowCounterBucket)

	if bucket1.CurrentCount != 3 {
		t.Errorf("user1 count: got %d, want 3", bucket1.CurrentCount)
	}
	if bucket2.CurrentCount != 3 {
		t.Errorf("user2 count: got %d, want 3", bucket2.CurrentCount)
	}
}

func TestSlidingWindowCounterWindowReset(t *testing.T) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(5, 100*time.Millisecond, s)

	for i := 0; i < 5; i++ {
		result, err := swc.Allow("user1")
		if err != nil || !result.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	result, err := swc.Allow("user1")
	if err != nil || result.Allowed {
		t.Errorf("6th request should be denied")
	}

	time.Sleep(150 * time.Millisecond)

	result, err = swc.Allow("user1")
	if err != nil || !result.Allowed {
		t.Errorf("request after window reset should be allowed")
	}
}

func TestSlidingWindowCounterRemaining(t *testing.T) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(10, 1*time.Second, s)

	for i := 0; i < 3; i++ {
		swc.Allow("user1")
	}

	result, err := swc.Allow("user1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Remaining < 5 || result.Remaining > 7 {
		t.Errorf("remaining: got %d, want between 5-7", result.Remaining)
	}
}

func TestSlidingWindowCounterRetryAfter(t *testing.T) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(2, 1*time.Second, s)

	swc.Allow("user1")
	swc.Allow("user1")

	result, err := swc.Allow("user1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Allowed {
		t.Errorf("request should be denied")
	}

	if result.RetryAfter == 0 {
		t.Errorf("RetryAfter should be greater than 0, got %v", result.RetryAfter)
	}

	if result.RetryAfter > 1*time.Second {
		t.Errorf("RetryAfter: got %v, want <= 1s", result.RetryAfter)
	}
}

func TestSlidingWindowCounterResultFields(t *testing.T) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(5, 1*time.Second, s)

	result, err := swc.Allow("user1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Allowed {
		t.Errorf("first request should be allowed")
	}

	if result.Limit != 5 {
		t.Errorf("Limit: got %d, want 5", result.Limit)
	}

	if result.Remaining < 4 || result.Remaining > 5 {
		t.Errorf("Remaining: got %d, want between 4-5", result.Remaining)
	}

	if result.RetryAfter != 0 {
		t.Errorf("RetryAfter should be 0 for allowed request, got %v", result.RetryAfter)
	}
}

func TestSlidingWindowCounterReset(t *testing.T) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(3, 1*time.Second, s)

	// Make 3 requests
	for i := 0; i < 3; i++ {
		swc.Allow("user1")
	}

	// Reset the bucket
	err := swc.Reset("user1")
	if err != nil {
		t.Errorf("unexpected error during reset: %v", err)
	}

	// Next request should be allowed (bucket was reset)
	result, err := swc.Allow("user1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Allowed {
		t.Errorf("request after reset should be allowed")
	}
}

func TestSlidingWindowCounterResetNonExistent(t *testing.T) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(3, 1*time.Second, s)

	err := swc.Reset("nonexistent")
	if err == nil {
		t.Errorf("expected error for non-existent key, got nil")
	}
}

func TestSlidingWindowCounterBucketData(t *testing.T) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(5, 1*time.Second, s)

	for i := 0; i < 2; i++ {
		swc.Allow("user1")
	}

	bucketData, err := s.Get("user1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	bucket := bucketData.(*SlidingWindowCounterBucket)

	if bucket.CurrentCount != 2 {
		t.Errorf("CurrentCount: got %d, want 2", bucket.CurrentCount)
	}

	if bucket.PreviousCount != 0 {
		t.Errorf("PreviousCount: got %d, want 0", bucket.PreviousCount)
	}

	if bucket.CurrentWindow == 0 {
		t.Errorf("CurrentWindow should not be 0")
	}
}

func TestSlidingWindowCounterConcurrency(t *testing.T) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(100, 1*time.Second, s)

	done := make(chan bool)
	allowed := 0

	for i := 0; i < 150; i++ {
		go func() {
			result, err := swc.Allow("user1")
			if err == nil && result.Allowed {
				allowed++
			}
			done <- true
		}()
	}

	for i := 0; i < 150; i++ {
		<-done
	}

	if allowed > 100 {
		t.Errorf("allowed requests exceeded limit: got %d, want <= 100", allowed)
	}
}

func TestSlidingWindowCounterPanicOnInvalidWindow(t *testing.T) {
	s := store.NewMemoryStore()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for non-positive window size")
		}
	}()

	NewSlidingWindowCounter(5, 0, s)
}

func TestSlidingWindowCounterSliding(t *testing.T) {
	s := store.NewMemoryStore()
	swc := NewSlidingWindowCounter(10, 1*time.Second, s)

	for i := 0; i < 10; i++ {
		result, err := swc.Allow("user1")
		if err != nil || !result.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	time.Sleep(500 * time.Millisecond)

	result, err := swc.Allow("user1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.RetryAfter > 0 && result.Allowed {
		t.Errorf("inconsistent result: allowed but has RetryAfter")
	}

	time.Sleep(600 * time.Millisecond)

	result, err = swc.Allow("user1")
	if err != nil || !result.Allowed {
		t.Errorf("request after window expires should be allowed")
	}
}
