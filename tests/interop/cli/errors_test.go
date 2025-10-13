package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tradeverifyd/scitt/tests/interop/lib"
)

// TestCLIErrorScenarios validates error handling across CLI implementations
// FR-040, FR-041: Verify both CLIs handle errors consistently
func TestCLIErrorScenarios(t *testing.T) {
	goDir, tsDir, cleanup := lib.SetupTestEnv(t)
	defer cleanup()

	// Define error scenarios
	errorScenarios := []struct {
		name            string
		command         []string
		expectedFailure bool
		description     string
	}{
		{
			name:            "missing-required-argument",
			command:         []string{"statement", "sign"},
			expectedFailure: true,
			description:     "Missing required --payload argument",
		},
		{
			name:            "invalid-file-path",
			command:         []string{"statement", "sign", "--payload", "/nonexistent/file.json", "--key", "key.json"},
			expectedFailure: true,
			description:     "Non-existent payload file",
		},
		{
			name:            "invalid-key-path",
			command:         []string{"statement", "sign", "--payload", "payload.json", "--key", "/nonexistent/key.json"},
			expectedFailure: true,
			description:     "Non-existent key file",
		},
		{
			name:            "unknown-command",
			command:         []string{"invalid-command"},
			expectedFailure: true,
			description:     "Unknown command should fail",
		},
		{
			name:            "invalid-flag",
			command:         []string{"statement", "sign", "--invalid-flag"},
			expectedFailure: true,
			description:     "Invalid flag should fail",
		},
		{
			name:            "invalid-port",
			command:         []string{"serve", "--port", "invalid"},
			expectedFailure: true,
			description:     "Invalid port value",
		},
		{
			name:            "negative-port",
			command:         []string{"serve", "--port", "-1"},
			expectedFailure: true,
			description:     "Negative port value",
		},
		{
			name:            "port-too-large",
			command:         []string{"serve", "--port", "99999"},
			expectedFailure: true,
			description:     "Port value too large",
		},
	}

	// Run each error scenario
	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Test Go CLI
			goResult := lib.RunGoCLI(scenario.command, goDir, nil, 10)

			// Test TypeScript CLI
			tsResult := lib.RunTsCLI(scenario.command, tsDir, nil, 10)

			// Both should fail if expectedFailure is true
			if scenario.expectedFailure {
				if goResult.ExitCode == 0 {
					t.Errorf("Go: Expected failure for %s, but succeeded", scenario.description)
				}
				if tsResult.ExitCode == 0 {
					t.Errorf("TypeScript: Expected failure for %s, but succeeded", scenario.description)
				}

				// Exit codes should be non-zero and similar
				if goResult.ExitCode != 0 && tsResult.ExitCode != 0 {
					// Both failed as expected
					t.Logf("Both failed as expected: Go=%d, TypeScript=%d", goResult.ExitCode, tsResult.ExitCode)
				}

				// Error messages should exist
				if goResult.Stderr == "" && goResult.Stdout == "" {
					t.Errorf("Go: No error message provided for %s", scenario.description)
				}
				if tsResult.Stderr == "" && tsResult.Stdout == "" {
					t.Errorf("TypeScript: No error message provided for %s", scenario.description)
				}
			}

			// Compare error outputs
			compareErrorOutputs(t, goResult, tsResult, scenario.name)
		})
	}
}

// TestCLIErrorMessages validates error message quality
func TestCLIErrorMessages(t *testing.T) {
	goDir, tsDir, cleanup := lib.SetupTestEnv(t)
	defer cleanup()

	// Test missing required argument
	t.Run("missing-payload-error", func(t *testing.T) {
		goResult := lib.RunGoCLI([]string{"statement", "sign", "--key", "key.json"}, goDir, nil, 10)
		tsResult := lib.RunTsCLI([]string{"statement", "sign", "--key", "key.json"}, tsDir, nil, 10)

		// Both should fail
		if goResult.ExitCode == 0 || tsResult.ExitCode == 0 {
			t.Error("Should fail when --payload is missing")
		}

		// Error messages should mention "payload"
		goError := goResult.Stderr + goResult.Stdout
		tsError := tsResult.Stderr + tsResult.Stdout

		if !containsIgnoreCase(goError, "payload") {
			t.Errorf("Go error should mention 'payload': %s", goError)
		}
		if !containsIgnoreCase(tsError, "payload") {
			t.Errorf("TypeScript error should mention 'payload': %s", tsError)
		}
	})

	// Test invalid file path
	t.Run("file-not-found-error", func(t *testing.T) {
		goResult := lib.RunGoCLI([]string{"statement", "sign",
			"--payload", "/nonexistent/file.json",
			"--key", "key.json"}, goDir, nil, 10)
		tsResult := lib.RunTsCLI([]string{"statement", "sign",
			"--payload", "/nonexistent/file.json",
			"--key", "key.json"}, tsDir, nil, 10)

		// Both should fail
		if goResult.ExitCode == 0 || tsResult.ExitCode == 0 {
			t.Error("Should fail when file doesn't exist")
		}

		// Error messages should mention file issue
		goError := goResult.Stderr + goResult.Stdout
		tsError := tsResult.Stderr + tsResult.Stdout

		fileErrorKeywords := []string{"not found", "does not exist", "no such file", "ENOENT"}
		if !containsAnyIgnoreCase(goError, fileErrorKeywords) {
			t.Logf("Go error may not clearly indicate file not found: %s", goError)
		}
		if !containsAnyIgnoreCase(tsError, fileErrorKeywords) {
			t.Logf("TypeScript error may not clearly indicate file not found: %s", tsError)
		}
	})
}

// TestCLIErrorExitCodes validates exit code consistency
func TestCLIErrorExitCodes(t *testing.T) {
	goDir, tsDir, cleanup := lib.SetupTestEnv(t)
	defer cleanup()

	// Test various error conditions and their exit codes
	errorTests := []struct {
		name     string
		command  []string
		checkFn  func(exitCode int) bool
		describe string
	}{
		{
			name:     "missing-args",
			command:  []string{"statement", "sign"},
			checkFn:  func(code int) bool { return code != 0 },
			describe: "Should have non-zero exit code",
		},
		{
			name:     "invalid-command",
			command:  []string{"nonexistent-command"},
			checkFn:  func(code int) bool { return code != 0 },
			describe: "Should have non-zero exit code",
		},
		{
			name:     "help-flag",
			command:  []string{"--help"},
			checkFn:  func(code int) bool { return code == 0 || code == 1 },
			describe: "Help should exit 0 or 1",
		},
	}

	for _, test := range errorTests {
		t.Run(test.name, func(t *testing.T) {
			goResult := lib.RunGoCLI(test.command, goDir, nil, 10)
			tsResult := lib.RunTsCLI(test.command, tsDir, nil, 10)

			if !test.checkFn(goResult.ExitCode) {
				t.Errorf("Go exit code %d doesn't match expectation: %s", goResult.ExitCode, test.describe)
			}
			if !test.checkFn(tsResult.ExitCode) {
				t.Errorf("TypeScript exit code %d doesn't match expectation: %s", tsResult.ExitCode, test.describe)
			}

			// Log exit code comparison
			if goResult.ExitCode != tsResult.ExitCode {
				t.Logf("Exit codes differ: Go=%d, TypeScript=%d (both valid if within expected range)",
					goResult.ExitCode, tsResult.ExitCode)
			}
		})
	}
}

// TestCLIMalformedInputHandling validates handling of malformed inputs
func TestCLIMalformedInputHandling(t *testing.T) {
	goDir, tsDir, cleanup := lib.SetupTestEnv(t)
	defer cleanup()

	// Create malformed files
	malformedJSON := filepath.Join(goDir, "malformed.json")
	if err := writeFile(malformedJSON, []byte("{invalid json")); err != nil {
		t.Fatal(err)
	}

	malformedKey := filepath.Join(goDir, "malformed.key")
	if err := writeFile(malformedKey, []byte("not a valid key")); err != nil {
		t.Fatal(err)
	}

	// Test malformed JSON payload
	t.Run("malformed-json", func(t *testing.T) {
		goResult := lib.RunGoCLI([]string{"statement", "sign",
			"--payload", malformedJSON,
			"--key", "key.json"}, goDir, nil, 10)
		tsResult := lib.RunTsCLI([]string{"statement", "sign",
			"--payload", malformedJSON,
			"--key", "key.json"}, tsDir, nil, 10)

		// Should fail or handle gracefully
		if goResult.ExitCode == 0 && tsResult.ExitCode == 0 {
			t.Log("Both implementations accept malformed JSON (may be intentional)")
		}
	})

	// Test malformed key file
	t.Run("malformed-key", func(t *testing.T) {
		validPayload := filepath.Join("..", "fixtures", "payloads", "small.json")

		goResult := lib.RunGoCLI([]string{"statement", "sign",
			"--payload", validPayload,
			"--key", malformedKey}, goDir, nil, 10)
		tsResult := lib.RunTsCLI([]string{"statement", "sign",
			"--payload", validPayload,
			"--key", malformedKey}, tsDir, nil, 10)

		// Both should fail with malformed key
		if goResult.ExitCode == 0 {
			t.Error("Go should reject malformed key")
		}
		if tsResult.ExitCode == 0 {
			t.Error("TypeScript should reject malformed key")
		}
	})
}

// TestCLIHelpConsistency validates help output consistency
func TestCLIHelpConsistency(t *testing.T) {
	goDir, tsDir, cleanup := lib.SetupTestEnv(t)
	defer cleanup()

	commands := [][]string{
		{"--help"},
		{"statement", "--help"},
		{"statement", "sign", "--help"},
		{"serve", "--help"},
	}

	for _, cmd := range commands {
		cmdName := strings.Join(cmd, " ")
		t.Run(cmdName, func(t *testing.T) {
			goResult := lib.RunGoCLI(cmd, goDir, nil, 10)
			tsResult := lib.RunTsCLI(cmd, tsDir, nil, 10)

			// Help should not fail catastrophically
			if goResult.ExitCode > 2 {
				t.Errorf("Go help exit code too high: %d", goResult.ExitCode)
			}
			if tsResult.ExitCode > 2 {
				t.Errorf("TypeScript help exit code too high: %d", tsResult.ExitCode)
			}

			// Should produce output
			if goResult.Stdout == "" && goResult.Stderr == "" {
				t.Error("Go help produced no output")
			}
			if tsResult.Stdout == "" && tsResult.Stderr == "" {
				t.Error("TypeScript help produced no output")
			}
		})
	}
}

// compareErrorOutputs compares error outputs from both implementations
func compareErrorOutputs(t *testing.T, goResult, tsResult *lib.CLIResult, scenarioName string) {
	t.Helper()

	// If both succeeded, nothing to compare
	if goResult.ExitCode == 0 && tsResult.ExitCode == 0 {
		return
	}

	// Compare exit codes
	if goResult.ExitCode != tsResult.ExitCode {
		t.Logf("Exit codes differ for %s: Go=%d, TypeScript=%d",
			scenarioName, goResult.ExitCode, tsResult.ExitCode)
	}

	// Log error outputs
	if goResult.ExitCode != 0 {
		t.Logf("Go error: %s", goResult.Stderr+goResult.Stdout)
	}
	if tsResult.ExitCode != 0 {
		t.Logf("TypeScript error: %s", tsResult.Stderr+tsResult.Stdout)
	}
}

// containsIgnoreCase checks if text contains substring (case-insensitive)
func containsIgnoreCase(text, substr string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substr))
}

// containsAnyIgnoreCase checks if text contains any of the substrings
func containsAnyIgnoreCase(text string, substrings []string) bool {
	for _, substr := range substrings {
		if containsIgnoreCase(text, substr) {
			return true
		}
	}
	return false
}

// writeFile writes content to a file
func writeFile(path string, content []byte) error {
	return os.WriteFile(path, content, 0644)
}
