package store

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// RateLimitEntry represents a row in the database
type RateLimitEntry struct {
	Key       string `gorm:"primaryKey"`
	Value     string
	ExpiresAt time.Time
}

type DatabaseStore struct {
	db *gorm.DB
}

func NewDatabaseStore(dsn string) (*DatabaseStore, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-create table if needed
	if err := db.AutoMigrate(&RateLimitEntry{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Create index on ExpiresAt for cleanup queries
	if !db.Migrator().HasIndex(&RateLimitEntry{}, "expires_at") {
		db.Migrator().CreateIndex(&RateLimitEntry{}, "expires_at")
	}

	return &DatabaseStore{db: db}, nil
}

// Get retrieves a value from database
func (ds *DatabaseStore) Get(key string) (interface{}, error) {
	var entry RateLimitEntry

	// Query and check if expired
	result := ds.db.Where("key = ? AND expires_at > ?", key, time.Now()).First(&entry)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("key not found")
	}
	if result.Error != nil {
		return nil, result.Error
	}

	// Return the JSON string (caller will unmarshal)
	return entry.Value, nil
}

// Set stores a value in database with TTL
func (ds *DatabaseStore) Set(key string, value interface{}, ttl time.Duration) error {
	// Convert value to JSON string
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	entry := RateLimitEntry{
		Key:       key,
		Value:     string(jsonData),
		ExpiresAt: time.Now().Add(ttl),
	}

	// Upsert (insert or update)
	return ds.db.Save(&entry).Error
}

// Delete removes a key from database
func (ds *DatabaseStore) Delete(key string) error {
	return ds.db.Delete(&RateLimitEntry{}, "key = ?", key).Error
}

// Exists checks if a key exists in database and hasn't expired
func (ds *DatabaseStore) Exists(key string) bool {
	var count int64
	ds.db.Model(&RateLimitEntry{}).
		Where("key = ? AND expires_at > ?", key, time.Now()).
		Count(&count)

	return count > 0
}

func (ds *DatabaseStore) CleanupExpired() error {
	return ds.db.Delete(&RateLimitEntry{}, "expires_at <= ?", time.Now()).Error
}

// Close closes the database connection
func (ds *DatabaseStore) Close() error {
	sqlDB, err := ds.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
