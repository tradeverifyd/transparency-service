package storage

import (
	"sync"
)

// CachedStorage wraps a Storage implementation with an in-memory read-through cache
// Writes are synchronously persisted to the underlying storage before updating the cache
// This ensures durability while providing fast reads for repeated access patterns
type CachedStorage struct {
	underlying Storage
	cache      map[string][]byte
	mu         sync.RWMutex
}

// NewCachedStorage creates a new cached storage wrapper
func NewCachedStorage(underlying Storage) *CachedStorage {
	return &CachedStorage{
		underlying: underlying,
		cache:      make(map[string][]byte),
	}
}

// Get retrieves data by key, using cache if available, otherwise fetching from underlying storage
func (c *CachedStorage) Get(key string) ([]byte, error) {
	// Try cache first with read lock
	c.mu.RLock()
	if data, found := c.cache[key]; found {
		c.mu.RUnlock()
		// Return a copy to prevent external modifications
		result := make([]byte, len(data))
		copy(result, data)
		return result, nil
	}
	c.mu.RUnlock()

	// Cache miss - fetch from underlying storage
	data, err := c.underlying.Get(key)
	if err != nil {
		return nil, err
	}

	// If data exists, cache it
	if data != nil {
		c.mu.Lock()
		// Store a copy in cache to prevent external modifications
		cached := make([]byte, len(data))
		copy(cached, data)
		c.cache[key] = cached
		c.mu.Unlock()

		// Return a copy
		result := make([]byte, len(data))
		copy(result, data)
		return result, nil
	}

	return nil, nil
}

// Put stores data at the specified key
// IMPORTANT: Writes to underlying storage FIRST, then updates cache
// This ensures durability - if the underlying write fails, cache is not updated
func (c *CachedStorage) Put(key string, data []byte) error {
	// Write to underlying storage first (durability)
	if err := c.underlying.Put(key, data); err != nil {
		return err
	}

	// Only update cache after successful write
	c.mu.Lock()
	// Store a copy in cache to prevent external modifications
	cached := make([]byte, len(data))
	copy(cached, data)
	c.cache[key] = cached
	c.mu.Unlock()

	return nil
}

// Delete removes data at the specified key
func (c *CachedStorage) Delete(key string) error {
	// Delete from underlying storage first
	if err := c.underlying.Delete(key); err != nil {
		return err
	}

	// Remove from cache
	c.mu.Lock()
	delete(c.cache, key)
	c.mu.Unlock()

	return nil
}

// Exists checks if a key exists
func (c *CachedStorage) Exists(key string) (bool, error) {
	// Check cache first
	c.mu.RLock()
	if _, found := c.cache[key]; found {
		c.mu.RUnlock()
		return true, nil
	}
	c.mu.RUnlock()

	// Fall back to underlying storage
	return c.underlying.Exists(key)
}

// List returns all keys with the given prefix
func (c *CachedStorage) List(prefix string) ([]string, error) {
	// List operations are not cached - delegate to underlying storage
	return c.underlying.List(prefix)
}

// Clear removes all data (delegates to underlying storage and clears cache)
func (c *CachedStorage) Clear() error {
	// Clear underlying storage first
	var err error
	switch store := c.underlying.(type) {
	case interface{ Clear() error }:
		err = store.Clear()
	case interface{ Clear() }:
		store.Clear()
	default:
		// No Clear method on underlying storage - just clear cache
		c.mu.Lock()
		c.cache = make(map[string][]byte)
		c.mu.Unlock()
		return nil
	}

	// Clear cache
	c.mu.Lock()
	c.cache = make(map[string][]byte)
	c.mu.Unlock()

	return err
}

// ClearCache clears only the in-memory cache without affecting underlying storage
// This is useful for testing or when you want to force a reload from storage
func (c *CachedStorage) ClearCache() {
	c.mu.Lock()
	c.cache = make(map[string][]byte)
	c.mu.Unlock()
}

// CacheStats returns cache statistics
func (c *CachedStorage) CacheStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := CacheStats{
		Entries: len(c.cache),
	}

	for _, data := range c.cache {
		stats.SizeBytes += int64(len(data))
	}

	return stats
}

// CacheStats contains cache statistics
type CacheStats struct {
	Entries   int   // Number of cached entries
	SizeBytes int64 // Total size of cached data in bytes
}
