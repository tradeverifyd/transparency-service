package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

var (
	portAllocator *PortAllocator
	portOnce      sync.Once
)

// SetupTestEnv creates an isolated test environment with temporary directories
// and allocated ports for both Go and TypeScript implementations.
// It returns the working directories for each implementation and a cleanup function.
func SetupTestEnv(t *testing.T) (goDir, tsDir string, cleanup func()) {
	t.Helper()

	// Initialize port allocator once
	portOnce.Do(func() {
		portAllocator = NewPortAllocator(20000, 30000)
	})

	// Create temporary directories using t.TempDir() for automatic cleanup
	baseDir := t.TempDir()

	goDir = filepath.Join(baseDir, "go-impl")
	tsDir = filepath.Join(baseDir, "ts-impl")

	// Create implementation-specific directories
	if err := os.MkdirAll(goDir, 0755); err != nil {
		t.Fatalf("Failed to create Go work directory: %v", err)
	}
	if err := os.MkdirAll(tsDir, 0755); err != nil {
		t.Fatalf("Failed to create TypeScript work directory: %v", err)
	}

	// Cleanup function (mostly handled by t.TempDir, but can add extra cleanup here)
	cleanup = func() {
		// t.TempDir() handles directory cleanup automatically
		// Add any additional cleanup here if needed (e.g., killing processes)
	}

	// Register cleanup to run when test completes
	t.Cleanup(cleanup)

	return goDir, tsDir, cleanup
}

// GetTestContext creates a TestExecutionContext for a test
func GetTestContext(t *testing.T, testName string) *TestExecutionContext {
	t.Helper()

	goDir, tsDir, _ := SetupTestEnv(t)

	// Allocate unique ports
	goPort := portAllocator.AllocatePort(t)
	tsPort := portAllocator.AllocatePort(t)

	return &TestExecutionContext{
		TestID:          fmt.Sprintf("test-%s-%d", testName, os.Getpid()),
		TestName:        testName,
		GoWorkDir:       goDir,
		TsWorkDir:       tsDir,
		GoServerPort:    goPort,
		TsServerPort:    tsPort,
		TimeoutSeconds:  120, // Default 2 minute timeout
		ParallelEnabled: true,
		CleanupOnPass:   true,
	}
}

// GetGoBinaryPath returns the path to the Go CLI binary
// Checks SCITT_GO_CLI environment variable first, then looks in standard locations
func GetGoBinaryPath() string {
	if envPath := os.Getenv("SCITT_GO_CLI"); envPath != "" {
		return envPath
	}

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "scitt" // Fallback
	}

	// Try standard locations relative to repository root
	// Assumes we're somewhere in tests/interop or its subdirectories
	candidates := []string{
		filepath.Join(cwd, "../../scitt-golang/scitt"),
		filepath.Join(cwd, "../../../scitt-golang/scitt"),                // from tests/interop/cli or tests/interop/http
		filepath.Join(cwd, "../../../../scitt-golang/scitt"),              // deeper nesting
		filepath.Join(filepath.Dir(cwd), "../scitt-golang/scitt"),         // alternative
		"../../scitt-golang/cmd/scitt/scitt",
		"/usr/local/bin/scitt",
	}

	for _, path := range candidates {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	return "scitt" // Fallback to PATH lookup
}

// GetTsCLICommand returns the command to invoke the TypeScript CLI
// Checks SCITT_TS_CLI environment variable first, then uses default
func GetTsCLICommand() string {
	if envCmd := os.Getenv("SCITT_TS_CLI"); envCmd != "" {
		return envCmd
	}

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "bun run ../../scitt-typescript/src/cli/index.ts" // Fallback
	}

	// Try standard locations relative to repository root
	candidates := []string{
		filepath.Join(cwd, "../../scitt-typescript/src/cli/index.ts"),
		filepath.Join(cwd, "../../../scitt-typescript/src/cli/index.ts"),
		filepath.Join(cwd, "../../../../scitt-typescript/src/cli/index.ts"),
	}

	for _, path := range candidates {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return fmt.Sprintf("bun run %s", absPath)
			}
		}
	}

	// Default fallback
	return "bun run ../../scitt-typescript/src/cli/index.ts"
}
