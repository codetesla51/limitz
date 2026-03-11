package store

import (
	"context"
	"time"
)

type Store interface {
	// Get retrieves bucket data for a key
	Get(ctx context.Context, key string) (interface{}, error)

	// Set stores bucket data with TTL (auto-expiration)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a key
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists
	Exists(ctx context.Context, key string) (bool, error)
}
