package interop

import (
	"os"
	"testing"
)

// TestMain is the entry point for the test suite
// It performs any necessary setup before tests run and cleanup after
func TestMain(m *testing.M) {
	// Setup phase
	// Note: Individual tests handle their own isolated environments via t.TempDir()
	// This is just for global test suite setup if needed

	// Run tests
	exitCode := m.Run()

	// Cleanup phase
	// Note: Individual tests handle their own cleanup via defer and t.Cleanup()

	os.Exit(exitCode)
}
