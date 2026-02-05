package algorithms

import (
	"testing"
	"time"

	"github.com/codetesla51/limitz/store"
)

// Test that a request is allowed when bucket has tokens
func TestAllowWithTokens(t *testing.T) {
	s := store.NewMemoryStore()
	limiter := NewTokenBucket(5, 2, s)

	result, err := limiter.Allow("user-a")
	if err != nil {
		t.Fatalf("Allow returned error: %v", err)
	}

	if !result.Allowed {
		t.Error("Expected request to be allowed, but it was denied")
	}

	bucketData, _ := s.Get("user-a")
	bucket := bucketData.(*Buckets)
	if bucket.tokens != 4 {
		t.Errorf("Expected 4 tokens left, got %d", bucket.tokens)
	}
}

// Test that a request is denied when bucket is empty
func TestDenyWithNoTokens(t *testing.T) {
	s := store.NewMemoryStore()
	limiter := NewTokenBucket(5, 2, s)

	// Consume all tokens first
	for i := 0; i < 5; i++ {
		_, _ = limiter.Allow("user-a")
	}

	result, err := limiter.Allow("user-a")
	if err != nil {
		t.Fatalf("Allow returned error: %v", err)
	}

	if result.Allowed {
		t.Error("Expected request to be denied, but it was allowed")
	}

	bucketData, _ := s.Get("user-a")
	bucket := bucketData.(*Buckets)
	if bucket.tokens != 0 {
		t.Errorf("Expected 0 tokens, got %d", bucket.tokens)
	}
}

// Test consuming all tokens one by one
func TestConsumeAllTokens(t *testing.T) {
	s := store.NewMemoryStore()
	limiter := NewTokenBucket(3, 2, s)

	// First 3 requests should pass
	result1, err1 := limiter.Allow("user-a")
	if err1 != nil || !result1.Allowed {
		t.Error("Request 1 should be allowed")
	}
	result2, err2 := limiter.Allow("user-a")
	if err2 != nil || !result2.Allowed {
		t.Error("Request 2 should be allowed")
	}
	result3, err3 := limiter.Allow("user-a")
	if err3 != nil || !result3.Allowed {
		t.Error("Request 3 should be allowed")
	}

	// 4th request should fail
	result4, err4 := limiter.Allow("user-a")
	if err4 != nil || result4.Allowed {
		t.Error("Request 4 should be denied")
	}
}

// Test that tokens refill over time
func TestTokenRefill(t *testing.T) {
	s := store.NewMemoryStore()
	limiter := NewTokenBucket(5, 1, s)

	// Consume all tokens
	for i := 0; i < 5; i++ {
		_, _ = limiter.Allow("user-a")
	}

	// Wait 1 second, should add 1 token
	time.Sleep(1 * time.Second)
	result, err := limiter.Allow("user-a")
	if err != nil {
		t.Fatalf("Allow returned error: %v", err)
	}

	if !result.Allowed {
		t.Error("Expected request to be allowed after refill")
	}

	bucketData, _ := s.Get("user-a")
	bucket := bucketData.(*Buckets)
	if bucket.tokens != 0 {
		t.Errorf("Expected 0 tokens left, got %d", bucket.tokens)
	}
}

// Test that refill doesn't exceed capacity
func TestRefillCappedAtCapacity(t *testing.T) {
	s := store.NewMemoryStore()
	limiter := NewTokenBucket(5, 10, s)

	// Create bucket with 3 tokens
	_, _ = limiter.Allow("user-a")
	_, _ = limiter.Allow("user-a")

	// Wait 1 second, would add 10 tokens but capped at 5
	time.Sleep(1 * time.Second)
	_, _ = limiter.Allow("user-a")

	bucketData, _ := s.Get("user-a")
	bucket := bucketData.(*Buckets)
	if bucket.tokens > limiter.Capacity {
		t.Errorf("Tokens (%d) exceeded capacity (%d)", bucket.tokens, limiter.Capacity)
	}
}

// Test reset function
func TestReset(t *testing.T) {
	s := store.NewMemoryStore()
	limiter := NewTokenBucket(5, 2, s)

	// Consume all tokens
	for i := 0; i < 5; i++ {
		_, _ = limiter.Allow("user-a")
	}

	// Reset
	err := limiter.Reset("user-a")

	if err != nil {
		t.Errorf("Reset should not return error: %v", err)
	}

	// After reset, user-a should not exist in store
	_, err = s.Get("user-a")
	if err == nil {
		t.Error("after reset, user-a should not exist in store")
	}
}

// Test with burst of requests
func TestBurstRequests(t *testing.T) {
	s := store.NewMemoryStore()
	limiter := NewTokenBucket(10, 1, s)

	// Fire 10 requests rapidly
	for i := 0; i < 10; i++ {
		result, err := limiter.Allow("user-a")
		if err != nil || !result.Allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 11th should fail
	result11, err11 := limiter.Allow("user-a")
	if err11 != nil || result11.Allowed {
		t.Error("11th request should be denied")
	}

	bucketData, _ := s.Get("user-a")
	bucket := bucketData.(*Buckets)
	if bucket.tokens != 0 {
		t.Errorf("Expected 0 tokens, got %d", bucket.tokens)
	}
}

// Test realistic scenario: 5 token capacity, 1 token per second refill
func TestRealisticRateLimiting(t *testing.T) {
	s := store.NewMemoryStore()
	limiter := NewTokenBucket(5, 1, s)

	// Can do 5 requests immediately
	for i := 0; i < 5; i++ {
		result, err := limiter.Allow("user-a")
		if err != nil || !result.Allowed {
			t.Errorf("Request %d should pass", i+1)
		}
	}

	// 6th request fails (no tokens)
	result6, err6 := limiter.Allow("user-a")
	if err6 != nil || result6.Allowed {
		t.Error("6th request should fail")
	}

	// Wait 2 seconds (2 tokens refill)
	time.Sleep(2 * time.Second)

	// Can now do 2 more requests
	result7, err7 := limiter.Allow("user-a")
	if err7 != nil || !result7.Allowed {
		t.Error("Request after 2sec wait should pass")
	}
	result8, err8 := limiter.Allow("user-a")
	if err8 != nil || !result8.Allowed {
		t.Error("2nd request after wait should pass")
	}

	// 3rd request after wait should fail
	result9, err9 := limiter.Allow("user-a")
	if err9 != nil || result9.Allowed {
		t.Error("3rd request after wait should fail")
	}
}

// Test separate keys have separate Buckets
func TestSeparateKeysHaveSeparateBuckets(t *testing.T) {
	s := store.NewMemoryStore()
	limiter := NewTokenBucket(3, 1, s)

	// User A consumes all tokens
	for i := 0; i < 3; i++ {
		_, _ = limiter.Allow("user-a")
	}

	// User A should be rate limited
	resultA, errA := limiter.Allow("user-a")
	if errA != nil || resultA.Allowed {
		t.Error("User A should be rate limited")
	}

	// But user B should still have full capacity
	resultB, errB := limiter.Allow("user-b")
	if errB != nil || !resultB.Allowed {
		t.Error("User B should not be rate limited")
	}

	bucketData, _ := s.Get("user-b")
	bucket := bucketData.(*Buckets)
	if bucket.tokens != 2 {
		t.Errorf("User B should have 2 tokens, got %d", bucket.tokens)
	}
}
