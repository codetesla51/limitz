package store

import (
	"testing"
	"time"
)

// RedisStore test - requires Redis running on localhost:6379
func TestRedisStoreBasic(t *testing.T) {
	// Connect to Redis
	rs, err := NewRedisStore("localhost:6379")
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	defer rs.Close()

	// Test Set and Get
	testData := map[string]interface{}{"count": 5, "name": "test"}
	err = rs.Set("testkey", testData, 10*time.Second)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get it back
	val, err := rs.Get("testkey")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if val == nil {
		t.Error("expected value, got nil")
	}
}

func TestRedisStoreDelete(t *testing.T) {
	rs, err := NewRedisStore("localhost:6379")
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	defer rs.Close()

	// Set a key
	rs.Set("delkey", "value", 10*time.Second)

	// Verify it exists
	if !rs.Exists("delkey") {
		t.Error("key should exist after Set")
	}

	// Delete it
	err = rs.Delete("delkey")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	if rs.Exists("delkey") {
		t.Error("key should not exist after Delete")
	}
}

func TestRedisStoreTTL(t *testing.T) {
	rs, err := NewRedisStore("localhost:6379")
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	defer rs.Close()

	// Set with 500ms TTL
	rs.Set("ttlkey", "value", 500*time.Millisecond)

	// Should exist now
	if !rs.Exists("ttlkey") {
		t.Error("key should exist immediately after Set")
	}

	// Wait for it to expire
	time.Sleep(600 * time.Millisecond)

	// Should be gone
	if rs.Exists("ttlkey") {
		t.Error("key should have expired after TTL")
	}
}

func TestRedisStoreDoesNotExist(t *testing.T) {
	rs, err := NewRedisStore("localhost:6379")
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	defer rs.Close()

	// Try to get non-existent key
	_, err = rs.Get("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent key")
	}

	// Exists should return false
	if rs.Exists("nonexistent") {
		t.Error("Exists should return false for non-existent key")
	}
}
