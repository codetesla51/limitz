package store

import (
	"fmt"
	"sync"
	"time"
)

type entry struct {
	value      interface{}
	expiration time.Time
}

type MemoryStore struct {
	data map[string]*entry
	mu   sync.Mutex
}

func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		data: make(map[string]*entry),
	}

	// Background cleanup of expired entries
	go store.cleanupExpired()

	return store
}

func (ms *MemoryStore) Get(key string) (interface{}, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	entry, exists := ms.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// Check if expired
	if time.Now().After(entry.expiration) {
		delete(ms.data, key)
		return nil, fmt.Errorf("key expired: %s", key)
	}

	return entry.value, nil
}

func (ms *MemoryStore) Set(key string, value interface{}, ttl time.Duration) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.data[key] = &entry{
		value:      value,
		expiration: time.Now().Add(ttl),
	}

	return nil
}

func (ms *MemoryStore) Delete(key string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if _, exists := ms.data[key]; !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	delete(ms.data, key)
	return nil
}

func (ms *MemoryStore) Exists(key string) bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	entry, exists := ms.data[key]
	if !exists {
		return false
	}

	// Check if expired
	if time.Now().After(entry.expiration) {
		delete(ms.data, key)
		return false
	}

	return true
}

// Background cleanup of expired entries (runs every 5 minutes)
func (ms *MemoryStore) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ms.mu.Lock()
		now := time.Now()
		for key, entry := range ms.data {
			if now.After(entry.expiration) {
				delete(ms.data, key)
			}
		}
		ms.mu.Unlock()
	}
}
