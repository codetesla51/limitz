package store

import "time"

type Store interface {
	// Get retrieves bucket data for a key
	Get(key string) (interface{}, error)

	// Set stores bucket data with TTL (auto-expiration)
	Set(key string, value interface{}, ttl time.Duration) error

	// Delete removes a key
	Delete(key string) error

	// Exists checks if a key exists
	Exists(key string) bool
}
