package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
)

// TestNewLocalStorage tests local storage creation
func TestNewLocalStorage(t *testing.T) {
	t.Run("creates storage with new directory", func(t *testing.T) {
		tempDir := t.TempDir()
		storePath := filepath.Join(tempDir, "test-storage")

		store, err := storage.NewLocalStorage(storePath)
		if err != nil {
			t.Fatalf("failed to create local storage: %v", err)
		}

		if store == nil {
			t.Fatal("expected non-nil storage")
		}

		// Verify directory was created
		if _, err := os.Stat(storePath); os.IsNotExist(err) {
			t.Error("storage directory was not created")
		}
	})

	t.Run("creates storage with existing directory", func(t *testing.T) {
		tempDir := t.TempDir()

		store, err := storage.NewLocalStorage(tempDir)
		if err != nil {
			t.Fatalf("failed to create local storage: %v", err)
		}

		if store == nil {
			t.Fatal("expected non-nil storage")
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		tempDir := t.TempDir()
		storePath := filepath.Join(tempDir, "nested", "path", "storage")

		_, err := storage.NewLocalStorage(storePath)
		if err != nil {
			t.Fatalf("failed to create local storage with nested path: %v", err)
		}

		// Verify nested directories were created
		if _, err := os.Stat(storePath); os.IsNotExist(err) {
			t.Error("nested storage directory was not created")
		}
	})
}

// TestLocalStoragePutGet tests put and get operations
func TestLocalStoragePutGet(t *testing.T) {
	t.Run("can store and retrieve data", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		data := []byte("test data")
		err := store.Put("test-key", data)
		if err != nil {
			t.Fatalf("failed to put data: %v", err)
		}

		retrieved, err := store.Get("test-key")
		if err != nil {
			t.Fatalf("failed to get data: %v", err)
		}

		if string(retrieved) != string(data) {
			t.Errorf("retrieved data does not match: expected %s, got %s", data, retrieved)
		}
	})

	t.Run("returns nil for non-existent key", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		retrieved, err := store.Get("non-existent")
		if err != nil {
			t.Fatalf("get should not error for non-existent key: %v", err)
		}

		if retrieved != nil {
			t.Error("expected nil for non-existent key")
		}
	})

	t.Run("can overwrite existing data", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		_ = store.Put("key", []byte("original"))
		_ = store.Put("key", []byte("updated"))

		retrieved, _ := store.Get("key")
		if string(retrieved) != "updated" {
			t.Errorf("expected updated data, got %s", retrieved)
		}
	})

	t.Run("handles nested keys with slashes", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		data := []byte("nested data")
		err := store.Put("dir1/dir2/file.txt", data)
		if err != nil {
			t.Fatalf("failed to put nested key: %v", err)
		}

		retrieved, err := store.Get("dir1/dir2/file.txt")
		if err != nil {
			t.Fatalf("failed to get nested key: %v", err)
		}

		if string(retrieved) != string(data) {
			t.Error("retrieved nested data does not match")
		}
	})

	t.Run("handles binary data", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		data := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		_ = store.Put("binary", data)

		retrieved, _ := store.Get("binary")
		for i, b := range data {
			if retrieved[i] != b {
				t.Errorf("byte %d mismatch: expected %x, got %x", i, b, retrieved[i])
			}
		}
	})

	t.Run("handles empty data", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		data := []byte{}
		_ = store.Put("empty", data)

		retrieved, _ := store.Get("empty")
		if len(retrieved) != 0 {
			t.Errorf("expected empty data, got %d bytes", len(retrieved))
		}
	})
}

// TestLocalStorageDelete tests delete operations
func TestLocalStorageDelete(t *testing.T) {
	t.Run("can delete existing key", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		_ = store.Put("to-delete", []byte("data"))

		err := store.Delete("to-delete")
		if err != nil {
			t.Fatalf("failed to delete: %v", err)
		}

		retrieved, _ := store.Get("to-delete")
		if retrieved != nil {
			t.Error("key should not exist after deletion")
		}
	})

	t.Run("delete non-existent key does not error", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		err := store.Delete("non-existent")
		if err != nil {
			t.Errorf("delete of non-existent key should not error: %v", err)
		}
	})
}

// TestLocalStorageExists tests exists operations
func TestLocalStorageExists(t *testing.T) {
	t.Run("returns true for existing key", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		_ = store.Put("exists", []byte("data"))

		exists, err := store.Exists("exists")
		if err != nil {
			t.Fatalf("exists check failed: %v", err)
		}

		if !exists {
			t.Error("key should exist")
		}
	})

	t.Run("returns false for non-existent key", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		exists, err := store.Exists("non-existent")
		if err != nil {
			t.Fatalf("exists check failed: %v", err)
		}

		if exists {
			t.Error("key should not exist")
		}
	})
}

// TestLocalStorageList tests list operations
func TestLocalStorageList(t *testing.T) {
	t.Run("lists all keys with empty prefix", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		_ = store.Put("key1", []byte("data1"))
		_ = store.Put("key2", []byte("data2"))
		_ = store.Put("key3", []byte("data3"))

		keys, err := store.List("")
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}

		if len(keys) != 3 {
			t.Errorf("expected 3 keys, got %d", len(keys))
		}
	})

	t.Run("filters keys by prefix", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		_ = store.Put("tile/0/001", []byte("data"))
		_ = store.Put("tile/0/002", []byte("data"))
		_ = store.Put("tile/1/001", []byte("data"))
		_ = store.Put("checkpoint/001", []byte("data"))

		keys, err := store.List("tile/0/")
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}

		if len(keys) != 2 {
			t.Errorf("expected 2 keys with prefix 'tile/0/', got %d", len(keys))
		}

		for _, key := range keys {
			if key != "tile/0/001" && key != "tile/0/002" {
				t.Errorf("unexpected key: %s", key)
			}
		}
	})

	t.Run("returns empty list for non-matching prefix", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		_ = store.Put("key1", []byte("data"))

		keys, err := store.List("non-matching")
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}

		if len(keys) != 0 {
			t.Errorf("expected empty list, got %d keys", len(keys))
		}
	})

	t.Run("handles nested directory structure", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		_ = store.Put("a/b/c/file1", []byte("data"))
		_ = store.Put("a/b/c/file2", []byte("data"))
		_ = store.Put("a/b/file3", []byte("data"))

		keys, err := store.List("a/b/c/")
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}

		if len(keys) != 2 {
			t.Errorf("expected 2 keys in a/b/c/, got %d", len(keys))
		}
	})
}

// TestLocalStorageClear tests clear operation
func TestLocalStorageClear(t *testing.T) {
	t.Run("removes all data", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		_ = store.Put("key1", []byte("data1"))
		_ = store.Put("key2", []byte("data2"))
		_ = store.Put("nested/key3", []byte("data3"))

		err := store.Clear()
		if err != nil {
			t.Fatalf("clear failed: %v", err)
		}

		keys, _ := store.List("")
		if len(keys) != 0 {
			t.Errorf("expected empty storage after clear, got %d keys", len(keys))
		}
	})

	t.Run("clear on empty storage does not error", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		err := store.Clear()
		if err != nil {
			t.Errorf("clear on empty storage should not error: %v", err)
		}
	})
}

// TestLocalStorageSize tests size operation
func TestLocalStorageSize(t *testing.T) {
	t.Run("returns correct size", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		_ = store.Put("key1", []byte("data"))
		_ = store.Put("key2", []byte("data"))
		_ = store.Put("key3", []byte("data"))

		size, err := store.Size()
		if err != nil {
			t.Fatalf("size check failed: %v", err)
		}

		if size != 3 {
			t.Errorf("expected size 3, got %d", size)
		}
	})

	t.Run("returns 0 for empty storage", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		size, _ := store.Size()
		if size != 0 {
			t.Errorf("expected size 0, got %d", size)
		}
	})
}

// TestLocalStorageCopy tests copy operations
func TestLocalStorageCopy(t *testing.T) {
	t.Run("can copy between storage instances", func(t *testing.T) {
		source, _ := storage.NewLocalStorage(filepath.Join(t.TempDir(), "source"))
		dest, _ := storage.NewLocalStorage(filepath.Join(t.TempDir(), "dest"))

		data := []byte("copy me")
		_ = source.Put("test-key", data)

		err := source.CopyTo(dest, "test-key")
		if err != nil {
			t.Fatalf("copy failed: %v", err)
		}

		retrieved, _ := dest.Get("test-key")
		if string(retrieved) != string(data) {
			t.Error("copied data does not match")
		}
	})

	t.Run("copy from memory to local", func(t *testing.T) {
		source := storage.NewMemoryStorage()
		dest, _ := storage.NewLocalStorage(t.TempDir())

		data := []byte("memory to local")
		_ = source.Put("key", data)

		err := dest.CopyFrom(source, "key")
		if err != nil {
			t.Fatalf("copy failed: %v", err)
		}

		retrieved, _ := dest.Get("key")
		if string(retrieved) != string(data) {
			t.Error("copied data does not match")
		}
	})
}

// TestLocalStorageOpenReader tests reader interface
func TestLocalStorageOpenReader(t *testing.T) {
	t.Run("can read data via reader", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		data := []byte("stream this data")
		_ = store.Put("stream-key", data)

		reader, err := store.OpenReader("stream-key")
		if err != nil {
			t.Fatalf("failed to open reader: %v", err)
		}
		defer reader.Close()

		buf := make([]byte, len(data))
		n, err := reader.Read(buf)
		if err != nil {
			t.Fatalf("failed to read: %v", err)
		}

		if n != len(data) {
			t.Errorf("expected to read %d bytes, got %d", len(data), n)
		}

		if string(buf) != string(data) {
			t.Error("read data does not match")
		}
	})

	t.Run("returns error for non-existent key", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		_, err := store.OpenReader("non-existent")
		if err == nil {
			t.Error("expected error for non-existent key")
		}
	})
}

// TestLocalStorageConcurrency tests concurrent operations
func TestLocalStorageConcurrency(t *testing.T) {
	t.Run("handles concurrent writes", func(t *testing.T) {
		store, _ := storage.NewLocalStorage(t.TempDir())

		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(n int) {
				_ = store.Put(string(rune('a'+n)), []byte{byte(n)})
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		size, _ := store.Size()
		if size != 10 {
			t.Errorf("expected 10 keys after concurrent writes, got %d", size)
		}
	})
}
