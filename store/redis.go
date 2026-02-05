package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements Store interface using Redis as backend
// This allows sharing rate limit state across multiple servers
type RedisStore struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisStore creates a new Redis-backed store
// addr format: "localhost:6379"
func NewRedisStore(addr string) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{
		client: client,
		ctx:    ctx,
	}, nil
}

// Get retrieves a value from Redis
// Since Redis stores strings, we JSON-encode/decode the value
func (r *RedisStore) Get(key string) (interface{}, error) {
	val, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		// Key doesn't exist
		return nil, fmt.Errorf("key not found")
	}
	if err != nil {
		return nil, err
	}

	// Redis stores it as JSON string, we get back the raw string
	// Caller will unmarshal/type-assert based on their needs
	return val, nil
}

// Set stores a value in Redis with TTL
func (r *RedisStore) Set(key string, value interface{}, ttl time.Duration) error {
	// Convert value to JSON string for storage
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// Store with TTL
	return r.client.Set(r.ctx, key, string(jsonData), ttl).Err()
}

// Delete removes a key from Redis
func (r *RedisStore) Delete(key string) error {
	return r.client.Del(r.ctx, key).Err()
}

// Exists checks if a key exists in Redis
func (r *RedisStore) Exists(key string) bool {
	exists, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		return false
	}
	return exists > 0
}

// Close closes the Redis connection
func (r *RedisStore) Close() error {
	return r.client.Close()
}
