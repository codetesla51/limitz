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
	stop chan struct{}
}

func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		data: make(map[string]*entry),
		stop: make(chan struct{}),
	}

	go store.cleanupExpired()

	return store
}

func (ms *MemoryStore) Close() {
	close(ms.stop)
}

func (ms *MemoryStore) Get(key string) (interface{}, error) {
	if key == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}

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
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	if value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	if ttl <= 0 {
		return fmt.Errorf("TTL must be greater than 0")
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.data[key] = &entry{
		value:      value,
		expiration: time.Now().Add(ttl),
	}

	return nil
}

func (ms *MemoryStore) Delete(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	if _, exists := ms.data[key]; !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	delete(ms.data, key)
	return nil
}

func (ms *MemoryStore) Exists(key string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("key cannot be empty")
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	entry, exists := ms.data[key]
	if !exists {
		return false, nil
	}

	// Check if expired
	if time.Now().After(entry.expiration) {
		delete(ms.data, key)
		return false, nil
	}

	return true, nil
}

func (ms *MemoryStore) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ms.stop:
			return
		case <-ticker.C:
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
}
