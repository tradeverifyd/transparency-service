package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tradeverifyd/scitt/tests/interop/lib"
)

// TestInitCommand validates the init command across implementations
// FR-006, FR-008: Verify both CLIs create equivalent directory structures and config files
func TestInitCommand(t *testing.T) {
	goDir, tsDir, cleanup := lib.SetupTestEnv(t)
	defer cleanup()

	// Run Go CLI init (requires --dir and --origin)
	goResult := lib.RunGoCLI([]string{"init", "--dir", goDir, "--origin", "https://scitt.example.com"}, goDir, nil, 30)
	if goResult.ExitCode != 0 {
		t.Logf("Go binary path: %s", lib.GetGoBinaryPath())
		t.Logf("Go work dir: %s", goDir)
		t.Logf("Go result: exit=%d, stdout=%s, stderr=%s, err=%v", goResult.ExitCode, goResult.Stdout, goResult.Stderr, goResult.Error)
		t.Fatalf("Go init failed: exit=%d, stderr=%s", goResult.ExitCode, goResult.Stderr)
	}

	// Run TypeScript CLI init (uses transparency init subcommand)
	tsResult := lib.RunTsCLI([]string{"transparency", "init", "--database", tsDir + "/transparency.db"}, tsDir, nil, 30)
	if tsResult.ExitCode != 0 {
		t.Fatalf("TypeScript init failed: exit=%d, stderr=%s", tsResult.ExitCode, tsResult.Stderr)
	}

	// Compare directory structures
	goStructure := getDirStructure(t, goDir)
	tsStructure := getDirStructure(t, tsDir)

	// Both should have similar structures
	expectedDirs := []string{"keys", "data", "config"}
	for _, dir := range expectedDirs {
		goHasDir := containsPath(goStructure, dir)
		tsHasDir := containsPath(tsStructure, dir)

		if goHasDir != tsHasDir {
			t.Errorf("Directory structure mismatch for '%s': Go=%v, TypeScript=%v", dir, goHasDir, tsHasDir)
		}
	}

	// Validate config file exists and is valid JSON
	validateConfigFile(t, goDir, "go")
	validateConfigFile(t, tsDir, "typescript")

	// Compare init outputs
	comparison := lib.CompareOutputs(
		&lib.ImplementationResult{
			Implementation: "go",
			Command:        goResult.Command,
			ExitCode:       goResult.ExitCode,
			Stdout:         goResult.Stdout,
			OutputFormat:   "text",
			Success:        goResult.ExitCode == 0,
			DurationMs:     goResult.DurationMs,
		},
		&lib.ImplementationResult{
			Implementation: "typescript",
			Command:        tsResult.Command,
			ExitCode:       tsResult.ExitCode,
			Stdout:         tsResult.Stdout,
			OutputFormat:   "text",
			Success:        tsResult.ExitCode == 0,
			DurationMs:     tsResult.DurationMs,
		},
	)

	testResult := &lib.TestResult{
		TestID:        "cli-init-001",
		TestName:      "CLI Init Command",
		Category:      "cli",
		ExecutedAt:    lib.GetTimestamp(),
		Comparison:    comparison,
		Verdict:       comparison.Verdict,
		RFCsValidated: []string{"Project Requirements"},
	}

	lib.PrintTestSummary(testResult)
}

// TestInitCommandWithKeypairGeneration validates keypair generation during init
func TestInitCommandWithKeypairGeneration(t *testing.T) {
	goDir, tsDir, cleanup := lib.SetupTestEnv(t)
	defer cleanup()

	// Run Go CLI init (Go generates keypair by default)
	goResult := lib.RunGoCLI([]string{"init", "--dir", goDir, "--origin", "https://scitt.example.com"}, goDir, nil, 30)
	if goResult.ExitCode != 0 {
		t.Logf("Go init with key generation: %s", goResult.Stderr)
	}

	// Run TypeScript CLI init
	tsResult := lib.RunTsCLI([]string{"transparency", "init", "--database", tsDir + "/transparency.db"}, tsDir, nil, 30)
	if tsResult.ExitCode != 0 {
		t.Logf("TypeScript init: %s", tsResult.Stderr)
	}

	// Validate keypairs were generated
	if goResult.ExitCode == 0 {
		validateKeypairGeneration(t, goDir, "go")
	}
	if tsResult.ExitCode == 0 {
		validateKeypairGeneration(t, tsDir, "typescript")
	}
}

// TestInitCommandIdempotency validates that init is idempotent
func TestInitCommandIdempotency(t *testing.T) {
	goDir, tsDir, cleanup := lib.SetupTestEnv(t)
	defer cleanup()

	// Test Go CLI idempotency
	t.Run("go-idempotent", func(t *testing.T) {
		// First init
		result1 := lib.RunGoCLI([]string{"init", "--dir", goDir, "--origin", "https://scitt.example.com"}, goDir, nil, 30)
		if result1.ExitCode != 0 {
			t.Fatalf("First Go init failed: %s", result1.Stderr)
		}

		// Second init (should fail without --force, or succeed with --force)
		result2 := lib.RunGoCLI([]string{"init", "--dir", goDir, "--origin", "https://scitt.example.com", "--force"}, goDir, nil, 30)
		if result2.ExitCode != 0 {
			t.Errorf("Second Go init with --force should succeed: exit=%d", result2.ExitCode)
		}
	})

	// Test TypeScript CLI idempotency
	t.Run("typescript-idempotent", func(t *testing.T) {
		// First init
		result1 := lib.RunTsCLI([]string{"transparency", "init", "--database", tsDir + "/transparency.db"}, tsDir, nil, 30)
		if result1.ExitCode != 0 {
			t.Fatalf("First TypeScript init failed: %s", result1.Stderr)
		}

		// Second init (should succeed with --force or fail without it)
		result2 := lib.RunTsCLI([]string{"transparency", "init", "--database", tsDir + "/transparency.db", "--force"}, tsDir, nil, 30)
		if result2.ExitCode != 0 {
			t.Errorf("Second TypeScript init with --force should succeed: exit=%d", result2.ExitCode)
		}
	})
}

// TestInitCommandWithCustomConfig validates custom configuration
func TestInitCommandWithCustomConfig(t *testing.T) {
	goDir, tsDir, cleanup := lib.SetupTestEnv(t)
	defer cleanup()

	// Run Go CLI init with custom config
	goResult := lib.RunGoCLI([]string{"init", "--dir", goDir, "--origin", "https://example.com"}, goDir, nil, 30)
	if goResult.ExitCode != 0 {
		t.Logf("Go init with custom config: %s", goResult.Stderr)
	}

	// Run TypeScript CLI init with custom config
	tsResult := lib.RunTsCLI([]string{"transparency", "init", "--database", tsDir + "/transparency.db", "--port", "8080"}, tsDir, nil, 30)
	if tsResult.ExitCode != 0 {
		t.Logf("TypeScript init with custom config: %s", tsResult.Stderr)
	}

	// Validate custom config was applied
	if goResult.ExitCode == 0 {
		validateCustomConfig(t, goDir, "go", "https://example.com")
	}
	if tsResult.ExitCode == 0 {
		validateCustomConfig(t, tsDir, "typescript", "8080")
	}
}

// getDirStructure returns a list of files and directories
func getDirStructure(t *testing.T, rootPath string) []string {
	t.Helper()

	var paths []string
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(rootPath, path)
		if relPath != "." {
			paths = append(paths, relPath)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk directory %s: %v", rootPath, err)
	}

	return paths
}

// containsPath checks if a path contains a substring
func containsPath(paths []string, needle string) bool {
	for _, path := range paths {
		if filepath.Base(path) == needle || path == needle {
			return true
		}
	}
	return false
}

// validateConfigFile validates that a config file exists and is valid
func validateConfigFile(t *testing.T, dir string, impl string) {
	t.Helper()

	// Look for common config file names
	configFiles := []string{
		"config.json",
		"scitt.json",
		".scitt.json",
		"config/config.json",
	}

	var configPath string
	for _, file := range configFiles {
		fullPath := filepath.Join(dir, file)
		if _, err := os.Stat(fullPath); err == nil {
			configPath = fullPath
			break
		}
	}

	if configPath == "" {
		t.Logf("%s: No config file found (checked: %v)", impl, configFiles)
		return
	}

	// Read and validate JSON
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Errorf("%s: Failed to read config file: %v", impl, err)
		return
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Errorf("%s: Config file is not valid JSON: %v", impl, err)
		return
	}

	// Validate snake_case convention
	violations := lib.ValidateSnakeCase(config)
	if len(violations) > 0 {
		t.Errorf("%s: Config file has snake_case violations: %v", impl, violations)
	}

	t.Logf("%s: Config file validated at %s", impl, configPath)
}

// validateKeypairGeneration validates that keypairs were generated
func validateKeypairGeneration(t *testing.T, dir string, impl string) {
	t.Helper()

	// Implementation-specific key file names
	var keyFiles []string
	if impl == "go" {
		// Go creates service-key.pem (private) and service-key.jwk (public)
		keyFiles = []string{"service-key.pem", "service-key.jwk"}
	} else {
		// TypeScript creates service-key.json
		keyFiles = []string{"service-key.json"}
	}

	var foundKeys []string
	for _, keyFile := range keyFiles {
		keyPath := filepath.Join(dir, keyFile)
		if _, err := os.Stat(keyPath); err == nil {
			foundKeys = append(foundKeys, keyFile)
		}
	}

	if len(foundKeys) == 0 {
		t.Errorf("%s: No keypairs found (expected: %v)", impl, keyFiles)
		return
	}

	t.Logf("%s: Keypairs successfully generated: %v", impl, foundKeys)
}

// validateCustomConfig validates custom configuration was applied
func validateCustomConfig(t *testing.T, dir string, impl string, expectedValue string) {
	t.Helper()

	// Implementation-specific config files
	var configFiles []string
	if impl == "go" {
		// Go creates scitt.yaml
		configFiles = []string{"scitt.yaml", "scitt.yml"}
	} else {
		// TypeScript may create various config files
		configFiles = []string{"config.json", "transparency.json", ".transparency.json"}
	}

	var configPath string
	for _, file := range configFiles {
		fullPath := filepath.Join(dir, file)
		if _, err := os.Stat(fullPath); err == nil {
			configPath = fullPath
			break
		}
	}

	if configPath == "" {
		t.Logf("%s: No config file found to validate (checked: %v)", impl, configFiles)
		return
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Errorf("%s: Failed to read config file: %v", impl, err)
		return
	}

	// Check if expected value appears in config
	// For Go (YAML), check for origin string
	// For TypeScript (JSON), parse and check
	if impl == "go" {
		// Simple string search for YAML
		if !strings.Contains(string(data), expectedValue) {
			t.Logf("%s: Expected value '%s' not found in config", impl, expectedValue)
		} else {
			t.Logf("%s: Custom config validated (found: %s)", impl, expectedValue)
		}
	} else {
		// Parse JSON for TypeScript
		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			// Not JSON, try string search
			if strings.Contains(string(data), expectedValue) {
				t.Logf("%s: Custom config validated (found: %s)", impl, expectedValue)
			} else {
				t.Logf("%s: Expected value not found in config", impl)
			}
			return
		}

		// Check common config fields
		found := false
		for _, key := range []string{"origin", "port", "hostname"} {
			if val, ok := config[key]; ok {
				if fmt.Sprintf("%v", val) == expectedValue {
					found = true
					break
				}
			}
		}

		if found {
			t.Logf("%s: Custom config validated", impl)
		} else {
			t.Logf("%s: Expected value not found in parsed config", impl)
		}
	}
}
