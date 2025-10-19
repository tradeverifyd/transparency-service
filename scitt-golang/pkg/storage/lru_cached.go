package storage

import (
	"container/list"
	"strings"
	"sync"
)

// LRUCachedStorage wraps a Storage implementation with an LRU (Least Recently Used) cache
// Features:
// - Configurable maximum cache size (in bytes)
// - Evicts least recently used tiles when cache is full
// - Prefers caching full tiles over partial tiles
// - Thread-safe with RWMutex
// - Write-through for durability
type LRUCachedStorage struct {
	underlying Storage
	maxBytes   int64 // Maximum cache size in bytes (0 = unlimited)

	// LRU tracking
	cache   map[string]*list.Element // key -> list element
	lruList *list.List               // doubly-linked list for LRU order
	mu      sync.RWMutex

	// Statistics
	hits   int64
	misses int64
	evictions int64
}

// cacheEntry represents a cached item
type cacheEntry struct {
	key       string
	data      []byte
	isPartial bool // true if this is a partial tile (lower priority for caching)
}

// LRUCacheConfig configures the LRU cache
type LRUCacheConfig struct {
	MaxBytes int64 // Maximum cache size in bytes (0 = unlimited)
}

// DefaultLRUCacheConfig returns default cache configuration
// Default: 100MB cache (can hold ~12,800 full tiles)
func DefaultLRUCacheConfig() LRUCacheConfig {
	return LRUCacheConfig{
		MaxBytes: 100 * 1024 * 1024, // 100MB
	}
}

// NewLRUCachedStorage creates a new LRU cached storage wrapper
func NewLRUCachedStorage(underlying Storage, config LRUCacheConfig) *LRUCachedStorage {
	return &LRUCachedStorage{
		underlying: underlying,
		maxBytes:   config.MaxBytes,
		cache:      make(map[string]*list.Element),
		lruList:    list.New(),
	}
}

// Get retrieves data by key, using cache if available
func (c *LRUCachedStorage) Get(key string) ([]byte, error) {
	// Try cache first with read lock
	c.mu.RLock()
	if elem, found := c.cache[key]; found {
		c.mu.RUnlock()

		// Move to front (most recently used) with write lock
		c.mu.Lock()
		c.lruList.MoveToFront(elem)
		c.hits++
		entry := elem.Value.(*cacheEntry)
		c.mu.Unlock()

		// Return a copy to prevent external modifications
		result := make([]byte, len(entry.data))
		copy(result, entry.data)
		return result, nil
	}
	c.mu.RUnlock()

	// Cache miss - fetch from underlying storage
	c.mu.Lock()
	c.misses++
	c.mu.Unlock()

	data, err := c.underlying.Get(key)
	if err != nil {
		return nil, err
	}

	// If data exists, cache it
	if data != nil {
		c.put(key, data)

		// Return a copy
		result := make([]byte, len(data))
		copy(result, data)
		return result, nil
	}

	return nil, nil
}

// Put stores data at the specified key
// IMPORTANT: Writes to underlying storage FIRST, then updates cache
func (c *LRUCachedStorage) Put(key string, data []byte) error {
	// Write to underlying storage first (durability)
	if err := c.underlying.Put(key, data); err != nil {
		return err
	}

	// Update cache after successful write
	c.put(key, data)

	return nil
}

// PutDeferred stores data in cache and optionally writes through to storage
// If writeThrough is true, writes to storage immediately
// If writeThrough is false, keeps in cache only (for incomplete entry tiles)
// This supports batching writes for incomplete tiles that will be completed later
func (c *LRUCachedStorage) PutDeferred(key string, data []byte, writeThrough bool) error {
	if writeThrough {
		// Write through to storage immediately (for complete tiles)
		if err := c.underlying.Put(key, data); err != nil {
			return err
		}
	}

	// Update cache regardless
	c.put(key, data)

	return nil
}

// Flush writes all cached data to underlying storage
// This should be called on server shutdown or periodically
func (c *LRUCachedStorage) Flush() error {
	c.mu.RLock()
	entries := make([]*cacheEntry, 0, len(c.cache))
	for _, elem := range c.cache {
		entry := elem.Value.(*cacheEntry)
		entries = append(entries, entry)
	}
	c.mu.RUnlock()

	// Write each cached entry to storage
	for _, entry := range entries {
		if err := c.underlying.Put(entry.key, entry.data); err != nil {
			return err
		}
	}

	return nil
}

// put adds an item to the cache (internal, must not hold lock)
func (c *LRUCachedStorage) put(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Make a copy for cache
	cached := make([]byte, len(data))
	copy(cached, data)

	// Determine if this is a partial tile
	isPartial := strings.Contains(key, ".p/")

	// Check if key already exists
	if elem, found := c.cache[key]; found {
		// Update existing entry and move to front
		c.lruList.MoveToFront(elem)
		entry := elem.Value.(*cacheEntry)
		entry.data = cached
		entry.isPartial = isPartial
		return
	}

	// Add new entry
	entry := &cacheEntry{
		key:       key,
		data:      cached,
		isPartial: isPartial,
	}
	elem := c.lruList.PushFront(entry)
	c.cache[key] = elem

	// Evict if over limit
	if c.maxBytes > 0 {
		c.evictIfNeeded()
	}
}

// evictIfNeeded evicts least recently used items until cache is under limit
// Prefers evicting partial tiles over full tiles
// Must be called with lock held
func (c *LRUCachedStorage) evictIfNeeded() {
	for c.currentSize() > c.maxBytes {
		// First pass: try to evict partial tiles from the back (LRU)
		evicted := false
		for elem := c.lruList.Back(); elem != nil; elem = elem.Prev() {
			entry := elem.Value.(*cacheEntry)
			if entry.isPartial {
				c.lruList.Remove(elem)
				delete(c.cache, entry.key)
				c.evictions++
				evicted = true
				break
			}
		}

		// If no partial tiles to evict, evict least recently used (back of list)
		if !evicted {
			elem := c.lruList.Back()
			if elem == nil {
				break // Should never happen, but safety check
			}
			entry := elem.Value.(*cacheEntry)
			c.lruList.Remove(elem)
			delete(c.cache, entry.key)
			c.evictions++
		}
	}
}

// currentSize returns the current cache size in bytes
// Must be called with lock held
func (c *LRUCachedStorage) currentSize() int64 {
	var size int64
	for _, elem := range c.cache {
		entry := elem.Value.(*cacheEntry)
		size += int64(len(entry.data))
	}
	return size
}

// Delete removes data at the specified key
func (c *LRUCachedStorage) Delete(key string) error {
	// Delete from underlying storage first
	if err := c.underlying.Delete(key); err != nil {
		return err
	}

	// Remove from cache
	c.mu.Lock()
	if elem, found := c.cache[key]; found {
		c.lruList.Remove(elem)
		delete(c.cache, key)
	}
	c.mu.Unlock()

	return nil
}

// Exists checks if a key exists
func (c *LRUCachedStorage) Exists(key string) (bool, error) {
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
func (c *LRUCachedStorage) List(prefix string) ([]string, error) {
	// List operations are not cached - delegate to underlying storage
	return c.underlying.List(prefix)
}

// Clear removes all data (delegates to underlying storage and clears cache)
func (c *LRUCachedStorage) Clear() error {
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
		c.cache = make(map[string]*list.Element)
		c.lruList.Init()
		c.mu.Unlock()
		return nil
	}

	// Clear cache
	c.mu.Lock()
	c.cache = make(map[string]*list.Element)
	c.lruList.Init()
	c.hits = 0
	c.misses = 0
	c.evictions = 0
	c.mu.Unlock()

	return err
}

// ClearCache clears only the in-memory cache without affecting underlying storage
func (c *LRUCachedStorage) ClearCache() {
	c.mu.Lock()
	c.cache = make(map[string]*list.Element)
	c.lruList.Init()
	c.hits = 0
	c.misses = 0
	c.evictions = 0
	c.mu.Unlock()
}

// CacheStats returns cache statistics
func (c *LRUCachedStorage) CacheStats() LRUCacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := LRUCacheStats{
		Entries:    len(c.cache),
		SizeBytes:  c.currentSize(),
		MaxBytes:   c.maxBytes,
		Hits:       c.hits,
		Misses:     c.misses,
		Evictions:  c.evictions,
	}

	// Count full vs partial tiles
	for _, elem := range c.cache {
		entry := elem.Value.(*cacheEntry)
		if entry.isPartial {
			stats.PartialTiles++
		} else {
			stats.FullTiles++
		}
	}

	// Calculate hit rate
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}

	return stats
}

// LRUCacheStats contains LRU cache statistics
type LRUCacheStats struct {
	Entries      int     // Number of cached entries
	FullTiles    int     // Number of full tiles cached
	PartialTiles int     // Number of partial tiles cached
	SizeBytes    int64   // Current cache size in bytes
	MaxBytes     int64   // Maximum cache size in bytes
	Hits         int64   // Cache hits
	Misses       int64   // Cache misses
	Evictions    int64   // Number of evictions
	HitRate      float64 // Hit rate (hits / (hits + misses))
}
