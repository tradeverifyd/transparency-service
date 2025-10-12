package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage implements Storage interface using local filesystem
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a new local filesystem storage
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
	}, nil
}

// getPath returns the full filesystem path for a key
func (s *LocalStorage) getPath(key string) string {
	return filepath.Join(s.basePath, filepath.FromSlash(key))
}

// Put stores data at the specified key
func (s *LocalStorage) Put(key string, data []byte) error {
	filePath := s.getPath(key)
	dir := filepath.Dir(filePath)

	// Ensure parent directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for key %s: %w", key, err)
	}

	// Write data atomically using temp file + rename
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file for key %s: %w", key, err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		// Clean up temp file on error
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file for key %s: %w", key, err)
	}

	return nil
}

// Get retrieves data at the specified key
func (s *LocalStorage) Get(key string) ([]byte, error) {
	filePath := s.getPath(key)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Return nil for not found, not an error
		}
		return nil, fmt.Errorf("failed to read file for key %s: %w", key, err)
	}

	return data, nil
}

// Delete removes data at the specified key
func (s *LocalStorage) Delete(key string) error {
	filePath := s.getPath(key)

	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file for key %s: %w", key, err)
	}

	return nil
}

// Exists checks if a key exists
func (s *LocalStorage) Exists(key string) (bool, error) {
	filePath := s.getPath(key)

	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat file for key %s: %w", key, err)
	}

	return true, nil
}

// List returns all keys with the given prefix
func (s *LocalStorage) List(prefix string) ([]string, error) {
	var keys []string

	// Walk the base directory
	err := filepath.Walk(s.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't access
			if os.IsPermission(err) {
				return filepath.SkipDir
			}
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Convert absolute path to relative key
		relPath, err := filepath.Rel(s.basePath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Normalize to use forward slashes (cross-platform)
		key := filepath.ToSlash(relPath)

		// Check if key starts with prefix
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return keys, nil
}

// Size returns the number of items in storage (for testing)
func (s *LocalStorage) Size() (int, error) {
	keys, err := s.List("")
	if err != nil {
		return 0, err
	}
	return len(keys), nil
}

// Clear removes all data (for testing)
func (s *LocalStorage) Clear() error {
	// Remove all contents of base directory
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		path := filepath.Join(s.basePath, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}

	return nil
}

// CopyFrom copies data from another storage to this one
func (s *LocalStorage) CopyFrom(source Storage, key string) error {
	data, err := source.Get(key)
	if err != nil {
		return fmt.Errorf("failed to get from source: %w", err)
	}

	if data == nil {
		return fmt.Errorf("key not found in source: %s", key)
	}

	if err := s.Put(key, data); err != nil {
		return fmt.Errorf("failed to put to destination: %w", err)
	}

	return nil
}

// CopyTo copies data from this storage to another one
func (s *LocalStorage) CopyTo(dest Storage, key string) error {
	data, err := s.Get(key)
	if err != nil {
		return fmt.Errorf("failed to get from source: %w", err)
	}

	if data == nil {
		return fmt.Errorf("key not found: %s", key)
	}

	if err := dest.Put(key, data); err != nil {
		return fmt.Errorf("failed to put to destination: %w", err)
	}

	return nil
}

// OpenReader returns a reader for streaming large objects (optional optimization)
func (s *LocalStorage) OpenReader(key string) (io.ReadCloser, error) {
	filePath := s.getPath(key)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to open file for key %s: %w", key, err)
	}

	return file, nil
}

// String returns a debug string representation
func (s *LocalStorage) String() string {
	size, _ := s.Size()
	return fmt.Sprintf("LocalStorage{basePath: %s, items: %d}", s.basePath, size)
}
