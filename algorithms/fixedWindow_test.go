package algorithms

import (
	"testing"
	"time"
)

func TestFixedWindowAllow(t *testing.T) {
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
			fw := &FixedWindow{
				Limit:      tt.limit,
				WindowSize: 1 * time.Second,
				Buckets:    make(map[string]*FixedWindowBucket),
			}

			allowed := 0
			for i := 0; i < tt.requests; i++ {
				if fw.Allow("user1") {
					allowed++
				}
			}

			if allowed != tt.expected {
				t.Errorf("got %d, want %d", allowed, tt.expected)
			}
		})
	}
}

func TestFixedWindowMultipleUsers(t *testing.T) {
	fw := &FixedWindow{
		Limit:      3,
		WindowSize: 1 * time.Second,
		Buckets:    make(map[string]*FixedWindowBucket),
	}

	// User 1 makes 3 requests
	for i := 0; i < 3; i++ {
		if !fw.Allow("user1") {
			t.Errorf("user1 request %d should be allowed", i+1)
		}
	}

	// User 2 makes 3 requests (different user, separate bucket)
	for i := 0; i < 3; i++ {
		if !fw.Allow("user2") {
			t.Errorf("user2 request %d should be allowed", i+1)
		}
	}

	// Both should have count=3
	if fw.Buckets["user1"].count != 3 {
		t.Errorf("user1 count: got %d, want 3", fw.Buckets["user1"].count)
	}
	if fw.Buckets["user2"].count != 3 {
		t.Errorf("user2 count: got %d, want 3", fw.Buckets["user2"].count)
	}
}

func TestFixedWindowWindowReset(t *testing.T) {
	fw := &FixedWindow{
		Limit:      5,
		WindowSize: 1 * time.Second,
		Buckets:    make(map[string]*FixedWindowBucket),
	}

	// Fill the window with 5 requests
	for i := 0; i < 5; i++ {
		if !fw.Allow("user1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	if fw.Buckets["user1"].count != 5 {
		t.Errorf("count before reset: got %d, want 5", fw.Buckets["user1"].count)
	}

	// Wait for window to change
	time.Sleep(1 * time.Second)

	// Next request should be allowed with reset count
	if !fw.Allow("user1") {
		t.Error("first request in new window should be allowed")
	}

	if fw.Buckets["user1"].count != 1 {
		t.Errorf("count after reset: got %d, want 1", fw.Buckets["user1"].count)
	}
}

func TestFixedWindowReset(t *testing.T) {
	fw := &FixedWindow{
		Limit:      5,
		WindowSize: 1 * time.Second,
		Buckets:    make(map[string]*FixedWindowBucket),
	}

	// Add some requests
	for i := 0; i < 3; i++ {
		fw.Allow("user1")
	}

	if fw.Buckets["user1"].count != 3 {
		t.Errorf("count before reset: got %d, want 3", fw.Buckets["user1"].count)
	}

	// Reset
	err := fw.Reset("user1")
	if err != nil {
		t.Errorf("reset failed: %v", err)
	}

	if fw.Buckets["user1"].count != 0 {
		t.Errorf("count after reset: got %d, want 0", fw.Buckets["user1"].count)
	}
}

func TestFixedWindowResetNonexistent(t *testing.T) {
	fw := &FixedWindow{
		Limit:      5,
		WindowSize: 1 * time.Second,
		Buckets:    make(map[string]*FixedWindowBucket),
	}

	// Try to reset nonexistent user
	err := fw.Reset("nonexistent")
	if err == nil {
		t.Error("reset nonexistent should return error")
	}
}

func TestFixedWindowConcurrency(t *testing.T) {
	fw := &FixedWindow{
		Limit:      100,
		WindowSize: 1 * time.Second,
		Buckets:    make(map[string]*FixedWindowBucket),
	}

	// Launch 10 goroutines, each making 10 requests
	done := make(chan int, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			count := 0
			for j := 0; j < 10; j++ {
				if fw.Allow("concurrent_user") {
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

	// Should have exactly 100 allowed (at limit)
	if totalAllowed != 100 {
		t.Errorf("concurrent total: got %d, want 100", totalAllowed)
	}
}

func TestFixedWindowEdgeCase(t *testing.T) {
	fw := &FixedWindow{
		Limit:      1,
		WindowSize: 1 * time.Second,
		Buckets:    make(map[string]*FixedWindowBucket),
	}

	// First request allowed
	if !fw.Allow("user1") {
		t.Error("first request should be allowed")
	}

	// Second request denied (at limit)
	if fw.Allow("user1") {
		t.Error("second request should be denied")
	}

	// Wait for window change
	time.Sleep(1 * time.Second)

	// Next request allowed (new window)
	if !fw.Allow("user1") {
		t.Error("first request in new window should be allowed")
	}
}
