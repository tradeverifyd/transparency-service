package storage

import (
	"bytes"
	"fmt"
	"testing"
)

func TestLRUCachedStorage_Basic(t *testing.T) {
	underlying := NewMemoryStorage()
	config := LRUCacheConfig{MaxBytes: 1024 * 1024} // 1MB
	cached := NewLRUCachedStorage(underlying, config)

	key := "test-key"
	data := []byte("test-data")

	err := cached.Put(key, data)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	retrieved, err := cached.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !bytes.Equal(retrieved, data) {
		t.Errorf("Retrieved data mismatch: got %s, want %s", retrieved, data)
	}
}

func TestLRUCachedStorage_Eviction(t *testing.T) {
	underlying := NewMemoryStorage()
	config := LRUCacheConfig{MaxBytes: 100} // Very small cache
	cached := NewLRUCachedStorage(underlying, config)

	// Put 3 items of 40 bytes each (total 120 bytes)
	// Cache can only hold 2 items (80 bytes), so oldest should be evicted
	data1 := make([]byte, 40)
	data2 := make([]byte, 40)
	data3 := make([]byte, 40)

	for i := range data1 {
		data1[i] = 1
		data2[i] = 2
		data3[i] = 3
	}

	// Put first item
	err := cached.Put("key1", data1)
	if err != nil {
		t.Fatalf("Put key1 failed: %v", err)
	}

	stats := cached.CacheStats()
	if stats.Entries != 1 {
		t.Errorf("Expected 1 entry after first put, got %d", stats.Entries)
	}

	// Put second item
	err = cached.Put("key2", data2)
	if err != nil {
		t.Fatalf("Put key2 failed: %v", err)
	}

	stats = cached.CacheStats()
	if stats.Entries != 2 {
		t.Errorf("Expected 2 entries after second put, got %d", stats.Entries)
	}

	// Put third item - should evict key1 (least recently used)
	err = cached.Put("key3", data3)
	if err != nil {
		t.Fatalf("Put key3 failed: %v", err)
	}

	stats = cached.CacheStats()
	if stats.Entries > 2 {
		t.Errorf("Expected at most 2 entries after eviction, got %d", stats.Entries)
	}

	if stats.Evictions < 1 {
		t.Errorf("Expected at least 1 eviction, got %d", stats.Evictions)
	}

	// key2 and key3 should be in cache, key1 should be evicted
	// All should still be retrievable from underlying storage
	retrieved1, err := cached.Get("key1")
	if err != nil {
		t.Fatalf("Get key1 failed: %v", err)
	}
	if !bytes.Equal(retrieved1, data1) {
		t.Error("key1 data mismatch (should be retrievable from underlying storage)")
	}
}

func TestLRUCachedStorage_PartialTileEviction(t *testing.T) {
	underlying := NewMemoryStorage()
	config := LRUCacheConfig{MaxBytes: 150} // Can hold ~3 items of 50 bytes
	cached := NewLRUCachedStorage(underlying, config)

	fullTileData := make([]byte, 50)
	partialTileData := make([]byte, 50)

	// Put full tile
	err := cached.Put("tile/entries/000", fullTileData)
	if err != nil {
		t.Fatalf("Put full tile failed: %v", err)
	}

	// Put partial tile
	err = cached.Put("tile/entries/001.p/128", partialTileData)
	if err != nil {
		t.Fatalf("Put partial tile failed: %v", err)
	}

	// Put another full tile
	err = cached.Put("tile/entries/002", fullTileData)
	if err != nil {
		t.Fatalf("Put second full tile failed: %v", err)
	}

	stats := cached.CacheStats()
	if stats.Entries != 3 {
		t.Errorf("Expected 3 entries, got %d", stats.Entries)
	}
	if stats.FullTiles != 2 {
		t.Errorf("Expected 2 full tiles, got %d", stats.FullTiles)
	}
	if stats.PartialTiles != 1 {
		t.Errorf("Expected 1 partial tile, got %d", stats.PartialTiles)
	}

	// Put one more full tile - should evict partial tile first
	err = cached.Put("tile/entries/003", fullTileData)
	if err != nil {
		t.Fatalf("Put third full tile failed: %v", err)
	}

	stats = cached.CacheStats()
	if stats.PartialTiles > 0 {
		t.Errorf("Expected partial tile to be evicted first, but found %d partial tiles", stats.PartialTiles)
	}
}

func TestLRUCachedStorage_LRUOrder(t *testing.T) {
	underlying := NewMemoryStorage()
	config := LRUCacheConfig{MaxBytes: 80} // Can hold 2 items of 40 bytes
	cached := NewLRUCachedStorage(underlying, config)

	data := make([]byte, 40)

	// Put key1 and key2
	cached.Put("key1", data)
	cached.Put("key2", data)

	// Access key1 (moves it to front)
	cached.Get("key1")

	// Put key3 - should evict key2 (least recently used), not key1
	cached.Put("key3", data)

	// Check that key1 and key3 are cached
	stats := cached.CacheStats()
	if stats.Entries != 2 {
		t.Errorf("Expected 2 entries, got %d", stats.Entries)
	}

	// Verify key1 is still in cache (should be a cache hit)
	initialHits := stats.Hits
	cached.Get("key1")
	stats = cached.CacheStats()
	if stats.Hits <= initialHits {
		t.Error("Expected cache hit for key1, but hit count did not increase")
	}
}

func TestLRUCachedStorage_HitRate(t *testing.T) {
	underlying := NewMemoryStorage()
	config := DefaultLRUCacheConfig()
	cached := NewLRUCachedStorage(underlying, config)

	data := []byte("test-data")

	// First access - cache miss
	cached.Put("key1", data)
	cached.Get("key1") // Cache hit

	// Second access - cache hit
	cached.Get("key1") // Cache hit

	// Access non-existent key - cache miss
	cached.Get("key2") // Cache miss

	stats := cached.CacheStats()

	// Should have 2 hits and 1 miss
	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	expectedHitRate := 2.0 / 3.0
	if stats.HitRate < expectedHitRate-0.01 || stats.HitRate > expectedHitRate+0.01 {
		t.Errorf("Expected hit rate ~%.2f, got %.2f", expectedHitRate, stats.HitRate)
	}
}

func TestLRUCachedStorage_UnlimitedCache(t *testing.T) {
	underlying := NewMemoryStorage()
	config := LRUCacheConfig{MaxBytes: 0} // Unlimited
	cached := NewLRUCachedStorage(underlying, config)

	// Put many items - none should be evicted
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		data := make([]byte, 1024) // 1KB each
		err := cached.Put(key, data)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	stats := cached.CacheStats()
	if stats.Entries != 100 {
		t.Errorf("Expected 100 entries with unlimited cache, got %d", stats.Entries)
	}
	if stats.Evictions != 0 {
		t.Errorf("Expected 0 evictions with unlimited cache, got %d", stats.Evictions)
	}
}

func TestLRUCachedStorage_ClearCache(t *testing.T) {
	underlying := NewMemoryStorage()
	config := DefaultLRUCacheConfig()
	cached := NewLRUCachedStorage(underlying, config)

	data := []byte("test-data")
	cached.Put("key1", data)
	cached.Put("key2", data)

	stats := cached.CacheStats()
	if stats.Entries != 2 {
		t.Errorf("Expected 2 entries before clear, got %d", stats.Entries)
	}

	// Clear cache
	cached.ClearCache()

	stats = cached.CacheStats()
	if stats.Entries != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", stats.Entries)
	}

	// Data should still be in underlying storage
	retrieved, err := underlying.Get("key1")
	if err != nil {
		t.Fatalf("Get from underlying failed: %v", err)
	}
	if !bytes.Equal(retrieved, data) {
		t.Error("Data not found in underlying storage after cache clear")
	}
}

func TestLRUCachedStorage_LargeScale(t *testing.T) {
	underlying := NewMemoryStorage()
	config := LRUCacheConfig{
		MaxBytes: 10 * 1024 * 1024, // 10MB - can hold ~1,280 full tiles
	}
	cached := NewLRUCachedStorage(underlying, config)

	// Simulate 10,000 entries (40 full tiles of 8KB each = 320KB)
	// This should easily fit in cache
	tileData := make([]byte, 8192) // 8KB tile

	for i := 0; i < 40; i++ {
		key := fmt.Sprintf("tile/entries/%03d", i)
		err := cached.Put(key, tileData)
		if err != nil {
			t.Fatalf("Put tile %d failed: %v", i, err)
		}
	}

	stats := cached.CacheStats()
	if stats.Entries != 40 {
		t.Errorf("Expected 40 entries, got %d", stats.Entries)
	}

	expectedSize := int64(40 * 8192)
	if stats.SizeBytes != expectedSize {
		t.Errorf("Expected cache size %d, got %d", expectedSize, stats.SizeBytes)
	}

	// Access all tiles multiple times - should all be cache hits
	initialHits := stats.Hits
	for i := 0; i < 40; i++ {
		for j := 0; j < 10; j++ {
			key := fmt.Sprintf("tile/entries/%03d", i)
			_, err := cached.Get(key)
			if err != nil {
				t.Fatalf("Get tile %d failed: %v", i, err)
			}
		}
	}

	stats = cached.CacheStats()
	expectedHits := initialHits + 400 // 40 tiles × 10 accesses
	if stats.Hits != expectedHits {
		t.Errorf("Expected %d hits, got %d", expectedHits, stats.Hits)
	}

	// Hit rate should be very high (all hits after initial population)
	if stats.HitRate < 0.95 {
		t.Errorf("Expected hit rate > 95%%, got %.2f%%", stats.HitRate*100)
	}
}

func BenchmarkLRUCachedStorage_Get(b *testing.B) {
	underlying := NewMemoryStorage()
	config := DefaultLRUCacheConfig()
	cached := NewLRUCachedStorage(underlying, config)

	data := make([]byte, 8192) // 8KB tile
	cached.Put("tile/entries/000", data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cached.Get("tile/entries/000")
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}

func BenchmarkLRUCachedStorage_Put(b *testing.B) {
	underlying := NewMemoryStorage()
	config := DefaultLRUCacheConfig()
	cached := NewLRUCachedStorage(underlying, config)

	data := make([]byte, 8192) // 8KB tile

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("tile/entries/%d", i)
		err := cached.Put(key, data)
		if err != nil {
			b.Fatalf("Put failed: %v", err)
		}
	}
}
