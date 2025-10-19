package storage

import (
	"bytes"
	"testing"
)

func TestCachedStorage_Basic(t *testing.T) {
	underlying := NewMemoryStorage()
	cached := NewCachedStorage(underlying)

	// Test Put and Get
	key := "test-key"
	data := []byte("test-data")

	err := cached.Put(key, data)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify data is in cache
	retrieved, err := cached.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !bytes.Equal(retrieved, data) {
		t.Errorf("Retrieved data mismatch: got %s, want %s", retrieved, data)
	}

	// Verify data is in underlying storage
	underlyingData, err := underlying.Get(key)
	if err != nil {
		t.Fatalf("Underlying Get failed: %v", err)
	}

	if !bytes.Equal(underlyingData, data) {
		t.Errorf("Underlying data mismatch: got %s, want %s", underlyingData, data)
	}
}

func TestCachedStorage_CacheHit(t *testing.T) {
	underlying := NewMemoryStorage()
	cached := NewCachedStorage(underlying)

	key := "test-key"
	data := []byte("test-data")

	// Put data
	err := cached.Put(key, data)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// First get (cache miss, loads from underlying)
	_, err = cached.Get(key)
	if err != nil {
		t.Fatalf("First Get failed: %v", err)
	}

	// Second get (cache hit) - should not touch underlying storage
	// We verify this by deleting from underlying and checking cache still works
	err = underlying.Delete(key)
	if err != nil {
		t.Fatalf("Underlying Delete failed: %v", err)
	}

	// This should still return data from cache
	retrieved, err := cached.Get(key)
	if err != nil {
		t.Fatalf("Cached Get after underlying delete failed: %v", err)
	}

	if !bytes.Equal(retrieved, data) {
		t.Errorf("Cache hit data mismatch: got %s, want %s", retrieved, data)
	}
}

func TestCachedStorage_WriteThroughDurability(t *testing.T) {
	underlying := NewMemoryStorage()
	cached := NewCachedStorage(underlying)

	key := "test-key"
	data := []byte("test-data")

	// Put should write to underlying BEFORE updating cache
	err := cached.Put(key, data)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify underlying storage has the data immediately
	underlyingData, err := underlying.Get(key)
	if err != nil {
		t.Fatalf("Underlying Get failed: %v", err)
	}

	if !bytes.Equal(underlyingData, data) {
		t.Errorf("Underlying data not persisted: got %s, want %s", underlyingData, data)
	}

	// Clear cache and verify we can still read from underlying
	cached.ClearCache()

	retrieved, err := cached.Get(key)
	if err != nil {
		t.Fatalf("Get after cache clear failed: %v", err)
	}

	if !bytes.Equal(retrieved, data) {
		t.Errorf("Data after cache clear mismatch: got %s, want %s", retrieved, data)
	}
}

func TestCachedStorage_Delete(t *testing.T) {
	underlying := NewMemoryStorage()
	cached := NewCachedStorage(underlying)

	key := "test-key"
	data := []byte("test-data")

	// Put data
	err := cached.Put(key, data)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify data exists
	exists, err := cached.Exists(key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Key should exist after Put")
	}

	// Delete
	err = cached.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify data is gone from cache
	retrieved, err := cached.Get(key)
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if retrieved != nil {
		t.Errorf("Data should be nil after delete, got %s", retrieved)
	}

	// Verify data is gone from underlying storage
	underlyingData, err := underlying.Get(key)
	if err != nil {
		t.Fatalf("Underlying Get after delete failed: %v", err)
	}
	if underlyingData != nil {
		t.Errorf("Underlying data should be nil after delete, got %s", underlyingData)
	}
}

func TestCachedStorage_Exists(t *testing.T) {
	underlying := NewMemoryStorage()
	cached := NewCachedStorage(underlying)

	key := "test-key"
	data := []byte("test-data")

	// Key should not exist initially
	exists, err := cached.Exists(key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("Key should not exist initially")
	}

	// Put data
	err = cached.Put(key, data)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Key should exist now
	exists, err = cached.Exists(key)
	if err != nil {
		t.Fatalf("Exists after Put failed: %v", err)
	}
	if !exists {
		t.Error("Key should exist after Put")
	}
}

func TestCachedStorage_List(t *testing.T) {
	underlying := NewMemoryStorage()
	cached := NewCachedStorage(underlying)

	// Put some data with a common prefix
	keys := []string{"prefix/key1", "prefix/key2", "prefix/key3", "other/key4"}
	for _, key := range keys {
		err := cached.Put(key, []byte("data"))
		if err != nil {
			t.Fatalf("Put failed for %s: %v", key, err)
		}
	}

	// List with prefix
	listed, err := cached.List("prefix/")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should have 3 keys with "prefix/" prefix
	if len(listed) != 3 {
		t.Errorf("List returned %d keys, expected 3", len(listed))
	}

	// Verify keys
	expectedKeys := map[string]bool{
		"prefix/key1": true,
		"prefix/key2": true,
		"prefix/key3": true,
	}

	for _, key := range listed {
		if !expectedKeys[key] {
			t.Errorf("Unexpected key in list: %s", key)
		}
	}
}

func TestCachedStorage_CacheStats(t *testing.T) {
	underlying := NewMemoryStorage()
	cached := NewCachedStorage(underlying)

	// Initially empty
	stats := cached.CacheStats()
	if stats.Entries != 0 {
		t.Errorf("Initial entries should be 0, got %d", stats.Entries)
	}
	if stats.SizeBytes != 0 {
		t.Errorf("Initial size should be 0, got %d", stats.SizeBytes)
	}

	// Put some data
	data1 := []byte("test-data-1")
	data2 := []byte("test-data-2-longer")

	err := cached.Put("key1", data1)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	err = cached.Put("key2", data2)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Check stats
	stats = cached.CacheStats()
	if stats.Entries != 2 {
		t.Errorf("Expected 2 entries, got %d", stats.Entries)
	}

	expectedSize := int64(len(data1) + len(data2))
	if stats.SizeBytes != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, stats.SizeBytes)
	}
}

func TestCachedStorage_ConcurrentAccess(t *testing.T) {
	underlying := NewMemoryStorage()
	cached := NewCachedStorage(underlying)

	// Test concurrent reads and writes
	const numGoroutines = 10
	const numOperations = 100

	done := make(chan bool)

	// Writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				key := "key-" + string(rune(id))
				data := []byte("data-" + string(rune(j)))
				err := cached.Put(key, data)
				if err != nil {
					t.Errorf("Put failed: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Readers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				key := "key-" + string(rune(id))
				_, err := cached.Get(key)
				if err != nil {
					t.Errorf("Get failed: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}
}

func TestCachedStorage_ImmutabilityCopy(t *testing.T) {
	underlying := NewMemoryStorage()
	cached := NewCachedStorage(underlying)

	key := "test-key"
	data := []byte("test-data")

	// Put data
	err := cached.Put(key, data)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Get data and modify it
	retrieved := []byte{}
	retrieved, err = cached.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Modify retrieved data
	retrieved[0] = 'X'

	// Get again and verify original data is unchanged
	retrieved2, err := cached.Get(key)
	if err != nil {
		t.Fatalf("Second Get failed: %v", err)
	}

	if !bytes.Equal(retrieved2, data) {
		t.Errorf("Cached data was modified: got %s, want %s", retrieved2, data)
	}
}
