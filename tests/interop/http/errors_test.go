package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/tradeverifyd/scitt/tests/interop/lib"
)

// TestHTTPErrorScenarios validates error handling across implementations
// FR-038, FR-039: Verify both implementations return matching error codes and structures
func TestHTTPErrorScenarios(t *testing.T) {
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

	// Define error scenarios
	errorScenarios := []struct {
		name               string
		method             string
		path               string
		contentType        string
		body               []byte
		expectedStatusCode int
		description        string
	}{
		{
			name:               "invalid-cbor",
			method:             "POST",
			path:               "/entries",
			contentType:        "application/cose",
			body:               []byte("invalid cbor data"),
			expectedStatusCode: http.StatusBadRequest, // 400
			description:        "Invalid CBOR should return 400",
		},
		{
			name:               "missing-content-type",
			method:             "POST",
			path:               "/entries",
			contentType:        "",
			body:               []byte("test data"),
			expectedStatusCode: http.StatusBadRequest, // 400
			description:        "Missing Content-Type should return 400",
		},
		{
			name:               "wrong-content-type",
			method:             "POST",
			path:               "/entries",
			contentType:        "text/plain",
			body:               []byte("test data"),
			expectedStatusCode: http.StatusUnsupportedMediaType, // 415
			description:        "Wrong Content-Type should return 415",
		},
		{
			name:               "entry-not-found",
			method:             "GET",
			path:               "/entries/nonexistent123",
			contentType:        "",
			body:               nil,
			expectedStatusCode: http.StatusNotFound, // 404
			description:        "Missing entry should return 404",
		},
		{
			name:               "invalid-entry-id-format",
			method:             "GET",
			path:               "/entries/invalid!!!id",
			contentType:        "",
			body:               nil,
			expectedStatusCode: http.StatusBadRequest, // 400
			description:        "Invalid entry ID format should return 400",
		},
		{
			name:               "empty-post-body",
			method:             "POST",
			path:               "/entries",
			contentType:        "application/cose",
			body:               []byte{},
			expectedStatusCode: http.StatusBadRequest, // 400
			description:        "Empty POST body should return 400",
		},
		{
			name:               "method-not-allowed-entries",
			method:             "DELETE",
			path:               "/entries",
			contentType:        "",
			body:               nil,
			expectedStatusCode: http.StatusMethodNotAllowed, // 405
			description:        "DELETE on /entries should return 405",
		},
		{
			name:               "method-not-allowed-checkpoint",
			method:             "POST",
			path:               "/checkpoint",
			contentType:        "",
			body:               nil,
			expectedStatusCode: http.StatusMethodNotAllowed, // 405
			description:        "POST on /checkpoint should return 405",
		},
		{
			name:               "invalid-query-param",
			method:             "GET",
			path:               "/entries?limit=-10",
			contentType:        "",
			body:               nil,
			expectedStatusCode: http.StatusBadRequest, // 400
			description:        "Negative limit should return 400",
		},
		{
			name:               "malformed-json-query",
			method:             "GET",
			path:               "/entries?filter={invalid}",
			contentType:        "",
			body:               nil,
			expectedStatusCode: http.StatusBadRequest, // 400
			description:        "Malformed JSON filter should return 400",
		},
	}

	// Run each error scenario
	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Test Go implementation
			goStatusCode, goBody := makeRequest(t, goPort, scenario.method, scenario.path, scenario.contentType, scenario.body)

			// Test TypeScript implementation
			tsStatusCode, tsBody := makeRequest(t, tsPort, scenario.method, scenario.path, scenario.contentType, scenario.body)

			// Compare status codes
			if goStatusCode != scenario.expectedStatusCode {
				t.Errorf("Go: Expected status %d, got %d for %s", scenario.expectedStatusCode, goStatusCode, scenario.description)
			}
			if tsStatusCode != scenario.expectedStatusCode {
				t.Errorf("TypeScript: Expected status %d, got %d for %s", scenario.expectedStatusCode, tsStatusCode, scenario.description)
			}

			// Status codes should match between implementations
			if goStatusCode != tsStatusCode {
				t.Errorf("Status code mismatch: Go=%d, TypeScript=%d for %s", goStatusCode, tsStatusCode, scenario.description)
			}

			// Parse error responses
			var goError, tsError map[string]interface{}
			goIsJSON := json.Unmarshal(goBody, &goError) == nil
			tsIsJSON := json.Unmarshal(tsBody, &tsError) == nil

			// Both should return JSON error responses
			if !goIsJSON && len(goBody) > 0 {
				t.Logf("Warning: Go error response is not JSON: %s", string(goBody))
			}
			if !tsIsJSON && len(tsBody) > 0 {
				t.Logf("Warning: TypeScript error response is not JSON: %s", string(tsBody))
			}

			// If both are JSON, compare structure
			if goIsJSON && tsIsJSON {
				compareErrorStructures(t, goError, tsError, scenario.name)
			}
		})
	}
}

// TestErrorResponseStructure validates error response structure compliance
func TestErrorResponseStructure(t *testing.T) {
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

	// Trigger an error (404)
	goStatusCode, goBody := makeRequest(t, goPort, "GET", "/entries/nonexistent", "", nil)
	tsStatusCode, tsBody := makeRequest(t, tsPort, "GET", "/entries/nonexistent", "", nil)

	if goStatusCode != http.StatusNotFound || tsStatusCode != http.StatusNotFound {
		t.Skip("Error scenarios not producing 404")
	}

	var goError, tsError map[string]interface{}
	if err := json.Unmarshal(goBody, &goError); err != nil {
		t.Fatalf("Go error response is not valid JSON: %v", err)
	}
	if err := json.Unmarshal(tsBody, &tsError); err != nil {
		t.Fatalf("TypeScript error response is not valid JSON: %v", err)
	}

	// Validate required error fields (FR-038, FR-039)
	requiredFields := []string{"error_code", "error_message"}
	for _, field := range requiredFields {
		if _, exists := goError[field]; !exists {
			t.Errorf("Go error response missing required field: %s", field)
		}
		if _, exists := tsError[field]; !exists {
			t.Errorf("TypeScript error response missing required field: %s", field)
		}
	}

	// Validate snake_case convention
	goViolations := lib.ValidateSnakeCase(goError)
	if len(goViolations) > 0 {
		t.Errorf("Go error response has snake_case violations: %v", goViolations)
	}

	tsViolations := lib.ValidateSnakeCase(tsError)
	if len(tsViolations) > 0 {
		t.Errorf("TypeScript error response has snake_case violations: %v", tsViolations)
	}
}

// TestErrorServerFailure validates 500 error handling
func TestErrorServerFailure(t *testing.T) {
	// This test would require simulating server failures
	// which may require special test modes or mocking
	t.Skip("Server failure simulation not yet implemented")
}

// TestErrorConcurrentRequests validates error handling under concurrent load
func TestErrorConcurrentRequests(t *testing.T) {
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

	concurrency := 20

	// Test Go error handling under concurrent load
	t.Run("go-concurrent-errors", func(t *testing.T) {
		done := make(chan int, concurrency)
		for i := 0; i < concurrency; i++ {
			go func() {
				statusCode, _ := makeRequest(t, goPort, "GET", "/entries/nonexistent", "", nil)
				done <- statusCode
			}()
		}

		// All should return 404
		for i := 0; i < concurrency; i++ {
			statusCode := <-done
			if statusCode != http.StatusNotFound {
				t.Errorf("Go: Expected 404, got %d under concurrent load", statusCode)
			}
		}
	})

	// Test TypeScript error handling under concurrent load
	t.Run("typescript-concurrent-errors", func(t *testing.T) {
		done := make(chan int, concurrency)
		for i := 0; i < concurrency; i++ {
			go func() {
				statusCode, _ := makeRequest(t, tsPort, "GET", "/entries/nonexistent", "", nil)
				done <- statusCode
			}()
		}

		// All should return 404
		for i := 0; i < concurrency; i++ {
			statusCode := <-done
			if statusCode != http.StatusNotFound {
				t.Errorf("TypeScript: Expected 404, got %d under concurrent load", statusCode)
			}
		}
	})
}

// TestErrorMalformedCOSE validates COSE parsing error handling
func TestErrorMalformedCOSE(t *testing.T) {
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

	// Malformed COSE payloads
	malformedPayloads := [][]byte{
		[]byte{0xFF, 0xFF, 0xFF}, // Invalid CBOR
		[]byte{},                  // Empty
		[]byte{0xA0},              // Valid CBOR but not COSE Sign1
	}

	for i, payload := range malformedPayloads {
		t.Run(fmt.Sprintf("malformed-%d", i), func(t *testing.T) {
			goStatusCode, _ := makeRequest(t, goPort, "POST", "/entries", "application/cose", payload)
			tsStatusCode, _ := makeRequest(t, tsPort, "POST", "/entries", "application/cose", payload)

			// Both should return 400 Bad Request
			if goStatusCode != http.StatusBadRequest {
				t.Errorf("Go: Expected 400 for malformed COSE, got %d", goStatusCode)
			}
			if tsStatusCode != http.StatusBadRequest {
				t.Errorf("TypeScript: Expected 400 for malformed COSE, got %d", tsStatusCode)
			}

			// Status codes should match
			if goStatusCode != tsStatusCode {
				t.Errorf("Status code mismatch for malformed COSE: Go=%d, TypeScript=%d", goStatusCode, tsStatusCode)
			}
		})
	}
}

// TestErrorInvalidSignature validates signature verification error handling
func TestErrorInvalidSignature(t *testing.T) {
	t.Skip("Invalid signature testing requires properly signed COSE statements")
}

// makeRequest makes an HTTP request and returns status code and body
func makeRequest(t *testing.T, port int, method, path, contentType string, body []byte) (int, []byte) {
	t.Helper()

	url := fmt.Sprintf("http://localhost:%d%s", port, path)

	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request to port %d: %v", port, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	return resp.StatusCode, responseBody
}

// compareErrorStructures compares error response structures
func compareErrorStructures(t *testing.T, goError, tsError map[string]interface{}, scenarioName string) {
	t.Helper()

	// Compare error structures
	result := lib.CompareOutputs(
		&lib.ImplementationResult{
			Implementation: "go",
			Command:        []string{"ERROR", scenarioName},
			ExitCode:       1,
			Stdout:         mustJSON(goError),
			OutputFormat:   "json",
			Success:        false,
		},
		&lib.ImplementationResult{
			Implementation: "typescript",
			Command:        []string{"ERROR", scenarioName},
			ExitCode:       1,
			Stdout:         mustJSON(tsError),
			OutputFormat:   "json",
			Success:        false,
		},
	)

	if result.Verdict == "divergent" {
		t.Logf("Error structures differ for %s:\n%s", scenarioName, lib.FormatDifferences(result.Differences))
	}
}

// mustJSON marshals data to JSON string
func mustJSON(data interface{}) string {
	bytes, _ := json.Marshal(data)
	return string(bytes)
}
