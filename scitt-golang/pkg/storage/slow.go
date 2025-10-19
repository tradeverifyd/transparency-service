package storage

import (
	"sync"
	"time"
)

// SlowStorage simulates a slow storage backend (e.g., network storage, Azure Blob, S3)
// This is useful for testing cache performance benefits with realistic latencies
type SlowStorage struct {
	underlying Storage
	getLatency time.Duration // Latency added to each Get operation
	mu         sync.RWMutex
}

// NewSlowStorage creates a slow storage wrapper with configurable latency
func NewSlowStorage(underlying Storage, getLatency time.Duration) *SlowStorage {
	return &SlowStorage{
		underlying: underlying,
		getLatency: getLatency,
	}
}

// Get retrieves data by key with added latency
func (s *SlowStorage) Get(key string) ([]byte, error) {
	// Simulate network/disk latency
	time.Sleep(s.getLatency)
	return s.underlying.Get(key)
}

// Put stores data at the specified key
func (s *SlowStorage) Put(key string, data []byte) error {
	return s.underlying.Put(key, data)
}

// Delete removes data at the specified key
func (s *SlowStorage) Delete(key string) error {
	return s.underlying.Delete(key)
}

// Exists checks if a key exists
func (s *SlowStorage) Exists(key string) (bool, error) {
	return s.underlying.Exists(key)
}

// List returns all keys with the given prefix
func (s *SlowStorage) List(prefix string) ([]string, error) {
	return s.underlying.List(prefix)
}

// Clear removes all data
func (s *SlowStorage) Clear() error {
	switch store := s.underlying.(type) {
	case interface{ Clear() error }:
		return store.Clear()
	case interface{ Clear() }:
		store.Clear()
		return nil
	default:
		return nil
	}
}
