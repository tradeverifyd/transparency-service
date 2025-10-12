package storage

import (
	"fmt"
	"strings"
	"sync"
)

// MemoryStorage is an in-memory storage implementation for testing
type MemoryStorage struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string][]byte),
	}
}

// Get retrieves data by key
func (s *MemoryStorage) Get(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.data[key]
	if !exists {
		return nil, nil
	}

	// Return a copy to prevent external modification
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// Put stores data at the specified key
func (s *MemoryStorage) Put(key string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store a copy to prevent external modification
	stored := make([]byte, len(data))
	copy(stored, data)
	s.data[key] = stored
	return nil
}

// Delete removes data at the specified key
func (s *MemoryStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return nil
}

// Exists checks if a key exists
func (s *MemoryStorage) Exists(key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.data[key]
	return exists, nil
}

// List returns all keys with the given prefix
func (s *MemoryStorage) List(prefix string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []string
	for key := range s.data {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// Size returns the number of items in storage (for testing)
func (s *MemoryStorage) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// Clear removes all data (for testing)
func (s *MemoryStorage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string][]byte)
}

// String returns a debug string representation
func (s *MemoryStorage) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fmt.Sprintf("MemoryStorage{items: %d}", len(s.data))
}
