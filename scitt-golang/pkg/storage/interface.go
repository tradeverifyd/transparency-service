// Package storage provides storage abstractions for the transparency service
package storage

// Storage is an interface for object storage operations
// Implementations include MinIO (S3-compatible) and filesystem storage
type Storage interface {
	// Get retrieves data by key
	// Returns nil if key does not exist
	Get(key string) ([]byte, error)

	// Put stores data at the specified key
	Put(key string, data []byte) error

	// Delete removes data at the specified key
	Delete(key string) error

	// Exists checks if a key exists
	Exists(key string) (bool, error)

	// List returns all keys with the given prefix
	List(prefix string) ([]string, error)
}
