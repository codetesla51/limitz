package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisStore(addr, username, password string) (*RedisStore, error) {
	if addr == "" {
		return nil, fmt.Errorf("Redis address cannot be empty")
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Username:     username,
		Password:     password,
		MaxRetries:   3,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{
		client: client,
		ctx:    context.Background(),
	}, nil
}

func (r *RedisStore) Get(key string) (interface{}, error) {
	if key == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}

	ctx, cancel := context.WithTimeout(r.ctx, 2*time.Second)
	defer cancel()

	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("Redis Get error: %w", err)
	}

	return val, nil
}

func (r *RedisStore) Set(key string, value interface{}, ttl time.Duration) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	if value == nil {
		return fmt.Errorf("value cannot be nil")
	}

	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	ctx, cancel := context.WithTimeout(r.ctx, 2*time.Second)
	defer cancel()

	if err := r.client.Set(ctx, key, string(jsonData), ttl).Err(); err != nil {
		return fmt.Errorf("Redis Set error: %w", err)
	}
	return nil
}

func (r *RedisStore) Delete(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	ctx, cancel := context.WithTimeout(r.ctx, 2*time.Second)
	defer cancel()

	deleted, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("Redis Delete error: %w", err)
	}
	if deleted == 0 {
		return fmt.Errorf("key not found")
	}
	return nil
}

func (r *RedisStore) Exists(key string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("key cannot be empty")
	}

	ctx, cancel := context.WithTimeout(r.ctx, 2*time.Second)
	defer cancel()

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("Redis Exists error: %w", err)
	}
	return exists > 0, nil
}

func (r *RedisStore) Close() error {
	return r.client.Close()
}
