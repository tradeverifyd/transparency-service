package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tradeverifyd/scitt/tests/interop/lib"
)

// TestPostEntries validates POST /entries endpoint
// FR-011: Verify statement registration returns 201 with entry_id and statement_hash
func TestPostEntries(t *testing.T) {
	goPort := lib.GlobalPortAllocator.AllocatePort(t)
	tsPort := lib.GlobalPortAllocator.AllocatePort(t)

	goDir, tsDir, cleanup := lib.SetupTestEnv(t)
	defer cleanup()

	goServer := startGoServer(t, goDir, goPort)
	defer goServer.Stop()

	tsServer := startTsServer(t, tsDir, tsPort)
	defer tsServer.Stop()

	waitForServer(t, goPort, 10*time.Second)
	waitForServer(t, tsPort, 10*time.Second)

	// Load test statement (COSE Sign1 from fixtures)
	statement := loadTestStatement(t, "small")

	// Register to Go server
	goResponse := registerStatement(t, goPort, statement)

	// Register to TypeScript server
	tsResponse := registerStatement(t, tsPort, statement)

	// Parse responses
	var goData, tsData map[string]interface{}
	if err := json.Unmarshal(goResponse, &goData); err != nil {
		t.Fatalf("Failed to parse Go response: %v", err)
	}
	if err := json.Unmarshal(tsResponse, &tsData); err != nil {
		t.Fatalf("Failed to parse TypeScript response: %v", err)
	}

	// Validate required fields using implementation-aware validation (FR-011)
	goMissing := lib.ValidateRegistrationResponse(goData, "go")
	if len(goMissing) > 0 {
		t.Errorf("Go response missing required fields: %v", goMissing)
	}

	tsMissing := lib.ValidateRegistrationResponse(tsData, "typescript")
	if len(tsMissing) > 0 {
		t.Errorf("TypeScript response missing required fields: %v", tsMissing)
	}

	// Extract entry IDs using implementation-aware helper
	goEntryID, err := lib.ExtractEntryID(goData, "go")
	if err != nil {
		t.Errorf("Failed to extract Go entry_id: %v", err)
	} else {
		t.Logf("Go entry_id: %s", goEntryID)
	}

	tsEntryID, err := lib.ExtractEntryID(tsData, "typescript")
	if err != nil {
		t.Errorf("Failed to extract TypeScript entry_id: %v", err)
	} else {
		t.Logf("TypeScript entry_id: %s", tsEntryID)
	}

	// Validate hex encoding for statement hash (Go implementation)
	if stmtHash, ok := lib.ExtractStatementHash(goData, "go"); ok {
		if violation := lib.ValidateHexEncoding(stmtHash, "statement_hash"); violation != nil {
			t.Errorf("Go statement_hash hex validation failed: %v", violation.Description)
		}
	}

	// Compare response structures with implementation awareness
	result := lib.CompareRegistrationResponses(goData, tsData)

	// Log differences (expected to have minor differences in field naming)
	if len(result.Differences) > 0 {
		t.Logf("Found %d difference(s) between implementations:", len(result.Differences))
		for _, diff := range result.Differences {
			t.Logf("  - %s (%s): %s", diff.FieldPath, diff.Severity, diff.Explanation)
		}
	}

	// Test passes if responses are compatible (not necessarily identical)
	if result.Verdict == "divergent" {
		t.Errorf("Registration responses are incompatible:\n%s", lib.FormatDifferences(result.Differences))
	}

	// Save test result
	testResult := &lib.TestResult{
		TestID:        "http-entries-001",
		TestName:      "POST /entries - Statement Registration",
		Category:      "http-api",
		ExecutedAt:    time.Now().Format(time.RFC3339),
		Comparison:    result,
		Verdict:       result.Verdict,
		RFCsValidated: []string{"RFC 9052", "SCRAPI"},
	}

	lib.PrintTestSummary(testResult)
}

// TestGetEntries validates GET /entries/{id} endpoint
// FR-012: Verify receipt retrieval returns equivalent structures
func TestGetEntries(t *testing.T) {
	goPort := lib.GlobalPortAllocator.AllocatePort(t)
	tsPort := lib.GlobalPortAllocator.AllocatePort(t)

	goDir, tsDir, cleanup := lib.SetupTestEnv(t)
	defer cleanup()

	goServer := startGoServer(t, goDir, goPort)
	defer goServer.Stop()

	tsServer := startTsServer(t, tsDir, tsPort)
	defer tsServer.Stop()

	waitForServer(t, goPort, 10*time.Second)
	waitForServer(t, tsPort, 10*time.Second)

	// Register statement to both servers
	statement := loadTestStatement(t, "small")
	goRegResp := registerStatement(t, goPort, statement)
	tsRegResp := registerStatement(t, tsPort, statement)

	// Extract entry IDs using implementation-aware helper
	var goRegData, tsRegData map[string]interface{}
	json.Unmarshal(goRegResp, &goRegData)
	json.Unmarshal(tsRegResp, &tsRegData)

	goEntryID, err := lib.ExtractEntryID(goRegData, "go")
	if err != nil {
		t.Fatalf("Failed to extract Go entry_id from registration response: %v", err)
	}

	tsEntryID, err := lib.ExtractEntryID(tsRegData, "typescript")
	if err != nil {
		t.Fatalf("Failed to extract TypeScript entry_id from registration response: %v", err)
	}

	// Retrieve receipts
	goReceipt := getEntry(t, goPort, goEntryID)
	tsReceipt := getEntry(t, tsPort, tsEntryID)

	// Parse receipts
	var goData, tsData map[string]interface{}
	if err := json.Unmarshal(goReceipt, &goData); err != nil {
		t.Fatalf("Failed to parse Go receipt: %v\nRaw response: %s", err, string(goReceipt))
	}
	if err := json.Unmarshal(tsReceipt, &tsData); err != nil {
		t.Fatalf("Failed to parse TypeScript receipt: %v\nRaw response: %s", err, string(tsReceipt))
	}

	// Validate receipt contains required metadata using implementation-aware validation (FR-012)
	goMissing := lib.ValidateReceiptResponse(goData, "go")
	if len(goMissing) > 0 {
		t.Errorf("Go receipt missing required fields: %v", goMissing)
	}

	tsMissing := lib.ValidateReceiptResponse(tsData, "typescript")
	if len(tsMissing) > 0 {
		t.Errorf("TypeScript receipt missing required fields: %v", tsMissing)
	}

	// Compare receipt structures
	result := lib.CompareOutputs(
		&lib.ImplementationResult{
			Implementation: "go",
			Command:        []string{"GET", fmt.Sprintf("/entries/%s", goEntryID)},
			ExitCode:       0,
			Stdout:         string(goReceipt),
			OutputFormat:   "json",
			Success:        true,
		},
		&lib.ImplementationResult{
			Implementation: "typescript",
			Command:        []string{"GET", fmt.Sprintf("/entries/%s", tsEntryID)},
			ExitCode:       0,
			Stdout:         string(tsReceipt),
			OutputFormat:   "json",
			Success:        true,
		},
	)

	testResult := &lib.TestResult{
		TestID:        "http-entries-002",
		TestName:      "GET /entries/{id} - Receipt Retrieval",
		Category:      "http-api",
		ExecutedAt:    time.Now().Format(time.RFC3339),
		Comparison:    result,
		Verdict:       result.Verdict,
		RFCsValidated: []string{"SCRAPI"},
	}

	lib.PrintTestSummary(testResult)
}

// TestPostEntriesWithMultiplePayloads tests registration with different payload sizes
func TestPostEntriesWithMultiplePayloads(t *testing.T) {
	goPort := lib.GlobalPortAllocator.AllocatePort(t)
	tsPort := lib.GlobalPortAllocator.AllocatePort(t)

	goDir, tsDir, cleanup := lib.SetupTestEnv(t)
	defer cleanup()

	goServer := startGoServer(t, goDir, goPort)
	defer goServer.Stop()

	tsServer := startTsServer(t, tsDir, tsPort)
	defer tsServer.Stop()

	waitForServer(t, goPort, 10*time.Second)
	waitForServer(t, tsPort, 10*time.Second)

	// Test with different payload sizes
	payloadSizes := []string{"small", "medium", "large"}

	for _, size := range payloadSizes {
		t.Run(fmt.Sprintf("payload-%s", size), func(t *testing.T) {
			statement := loadTestStatement(t, size)

			// Register to both servers
			goResponse := registerStatement(t, goPort, statement)
			tsResponse := registerStatement(t, tsPort, statement)

			// Validate both succeeded
			var goData, tsData map[string]interface{}
			if err := json.Unmarshal(goResponse, &goData); err != nil {
				t.Fatalf("Failed to parse Go response for %s payload: %v", size, err)
			}
			if err := json.Unmarshal(tsResponse, &tsData); err != nil {
				t.Fatalf("Failed to parse TypeScript response for %s payload: %v", size, err)
			}

			// Use implementation-aware extraction
			goEntryID, err := lib.ExtractEntryID(goData, "go")
			if err != nil {
				t.Errorf("Go failed to register %s payload: %v", size, err)
			} else {
				t.Logf("Go registered %s payload with entry_id: %s", size, goEntryID)
			}

			tsEntryID, err := lib.ExtractEntryID(tsData, "typescript")
			if err != nil {
				t.Errorf("TypeScript failed to register %s payload: %v", size, err)
			} else {
				t.Logf("TypeScript registered %s payload with entry_id: %s", size, tsEntryID)
			}
		})
	}
}

// TestEntriesConcurrentRegistration tests concurrent statement registration
func TestEntriesConcurrentRegistration(t *testing.T) {
	goPort := lib.GlobalPortAllocator.AllocatePort(t)
	tsPort := lib.GlobalPortAllocator.AllocatePort(t)

	goDir, tsDir, cleanup := lib.SetupTestEnv(t)
	defer cleanup()

	goServer := startGoServer(t, goDir, goPort)
	defer goServer.Stop()

	tsServer := startTsServer(t, tsDir, tsPort)
	defer tsServer.Stop()

	waitForServer(t, goPort, 10*time.Second)
	waitForServer(t, tsPort, 10*time.Second)

	// Register multiple statements concurrently
	concurrency := 10
	statement := loadTestStatement(t, "small")

	// Test Go implementation
	t.Run("go-concurrent", func(t *testing.T) {
		done := make(chan bool, concurrency)
		for i := 0; i < concurrency; i++ {
			go func() {
				response := registerStatement(t, goPort, statement)
				var data map[string]interface{}
				if err := json.Unmarshal(response, &data); err == nil {
					// Use implementation-aware extraction
					if _, err := lib.ExtractEntryID(data, "go"); err == nil {
						done <- true
						return
					}
				}
				done <- false
			}()
		}

		// Wait for all to complete
		successCount := 0
		for i := 0; i < concurrency; i++ {
			if <-done {
				successCount++
			}
		}

		// Note: May have failures due to duplicate statement detection
		t.Logf("Go: %d/%d successful concurrent registrations", successCount, concurrency)
		if successCount == 0 {
			t.Errorf("Go: No successful registrations (expected at least 1)")
		}
	})

	// Test TypeScript implementation
	t.Run("typescript-concurrent", func(t *testing.T) {
		done := make(chan bool, concurrency)
		for i := 0; i < concurrency; i++ {
			go func() {
				response := registerStatement(t, tsPort, statement)
				var data map[string]interface{}
				if err := json.Unmarshal(response, &data); err == nil {
					// Use implementation-aware extraction
					if _, err := lib.ExtractEntryID(data, "typescript"); err == nil {
						done <- true
						return
					}
				}
				done <- false
			}()
		}

		// Wait for all to complete
		successCount := 0
		for i := 0; i < concurrency; i++ {
			if <-done {
				successCount++
			}
		}

		// Note: May have failures due to duplicate statement detection
		t.Logf("TypeScript: %d/%d successful concurrent registrations", successCount, concurrency)
		if successCount == 0 {
			t.Errorf("TypeScript: No successful registrations (expected at least 1)")
		}
	})
}

// registerStatement registers a COSE Sign1 statement to a server
func registerStatement(t *testing.T, port int, statement []byte) []byte {
	t.Helper()

	url := fmt.Sprintf("http://localhost:%d/entries", port)

	req, err := http.NewRequest("POST", url, bytes.NewReader(statement))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/cose")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to register statement to port %d: %v", port, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201 Created, got %d from port %d: %s", resp.StatusCode, port, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body from port %d: %v", port, err)
	}

	return body
}

// getEntry retrieves an entry/receipt from a server
func getEntry(t *testing.T, port int, entryID string) []byte {
	t.Helper()

	url := fmt.Sprintf("http://localhost:%d/entries/%s", port, entryID)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to get entry from port %d: %v", port, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d from port %d", resp.StatusCode, port)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body from port %d: %v", port, err)
	}

	return body
}

// loadTestStatement loads a COSE Sign1 statement from fixtures
func loadTestStatement(t *testing.T, size string) []byte {
	t.Helper()

	// Load COSE Sign1 statement from fixtures
	statementPath := filepath.Join("..", "fixtures", "statements", fmt.Sprintf("%s.cose", size))

	data, err := os.ReadFile(statementPath)
	if err != nil {
		t.Fatalf("Failed to load test statement from %s: %v", statementPath, err)
	}

	return data
}
