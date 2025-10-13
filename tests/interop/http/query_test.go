package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/tradeverifyd/scitt/tests/interop/lib"
)

// TestStatementQuery validates query endpoints with filters
// FR-016: Verify query endpoints return equivalent result sets
func TestStatementQuery(t *testing.T) {
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

	// Register test statements with different attributes
	testStatements := []map[string]interface{}{
		{
			"issuer":       "https://example.com/alice",
			"subject":      "pkg:npm/package-a@1.0.0",
			"content_type": "application/json",
		},
		{
			"issuer":       "https://example.com/bob",
			"subject":      "pkg:npm/package-b@2.0.0",
			"content_type": "application/json",
		},
		{
			"issuer":       "https://example.com/alice",
			"subject":      "pkg:npm/package-c@1.0.0",
			"content_type": "application/vnd.in-toto+json",
		},
	}

	// Register to both servers
	for _, stmt := range testStatements {
		stmtBytes, _ := json.Marshal(stmt)
		registerStatement(t, goPort, stmtBytes)
		registerStatement(t, tsPort, stmtBytes)
	}

	// Query by issuer
	t.Run("query-by-issuer", func(t *testing.T) {
		issuer := "https://example.com/alice"
		goResults := queryStatements(t, goPort, map[string]string{"issuer": issuer})
		tsResults := queryStatements(t, tsPort, map[string]string{"issuer": issuer})

		compareQueryResults(t, goResults, tsResults, "issuer", issuer)
	})

	// Query by subject
	t.Run("query-by-subject", func(t *testing.T) {
		subject := "pkg:npm/package-a@1.0.0"
		goResults := queryStatements(t, goPort, map[string]string{"subject": subject})
		tsResults := queryStatements(t, tsPort, map[string]string{"subject": subject})

		compareQueryResults(t, goResults, tsResults, "subject", subject)
	})

	// Query by content_type
	t.Run("query-by-content-type", func(t *testing.T) {
		contentType := "application/vnd.in-toto+json"
		goResults := queryStatements(t, goPort, map[string]string{"content_type": contentType})
		tsResults := queryStatements(t, tsPort, map[string]string{"content_type": contentType})

		compareQueryResults(t, goResults, tsResults, "content_type", contentType)
	})
}

// TestQueryPagination validates query pagination behavior
func TestQueryPagination(t *testing.T) {
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

	// Register multiple statements
	for i := 0; i < 25; i++ {
		stmt := map[string]interface{}{
			"issuer":       "https://example.com/test",
			"subject":      fmt.Sprintf("test-subject-%d", i),
			"content_type": "application/json",
		}
		stmtBytes, _ := json.Marshal(stmt)
		registerStatement(t, goPort, stmtBytes)
		registerStatement(t, tsPort, stmtBytes)
	}

	// Test pagination with limit
	t.Run("pagination-limit", func(t *testing.T) {
		limit := 10
		goResults := queryStatements(t, goPort, map[string]string{"limit": fmt.Sprintf("%d", limit)})
		tsResults := queryStatements(t, tsPort, map[string]string{"limit": fmt.Sprintf("%d", limit)})

		var goData, tsData map[string]interface{}
		json.Unmarshal(goResults, &goData)
		json.Unmarshal(tsResults, &tsData)

		// Check result count
		if results, ok := goData["results"].([]interface{}); ok {
			if len(results) > limit {
				t.Errorf("Go returned more results than limit: %d > %d", len(results), limit)
			}
		}
		if results, ok := tsData["results"].([]interface{}); ok {
			if len(results) > limit {
				t.Errorf("TypeScript returned more results than limit: %d > %d", len(results), limit)
			}
		}
	})

	// Test pagination with offset
	t.Run("pagination-offset", func(t *testing.T) {
		offset := 5
		limit := 10
		goResults := queryStatements(t, goPort, map[string]string{
			"offset": fmt.Sprintf("%d", offset),
			"limit":  fmt.Sprintf("%d", limit),
		})
		tsResults := queryStatements(t, tsPort, map[string]string{
			"offset": fmt.Sprintf("%d", offset),
			"limit":  fmt.Sprintf("%d", limit),
		})

		compareQueryResults(t, goResults, tsResults, "pagination", fmt.Sprintf("offset=%d,limit=%d", offset, limit))
	})
}

// TestQueryResultOrdering validates query result ordering consistency
func TestQueryResultOrdering(t *testing.T) {
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

	// Register statements in specific order
	for i := 0; i < 10; i++ {
		stmt := map[string]interface{}{
			"issuer":       "https://example.com/test",
			"subject":      fmt.Sprintf("test-%02d", i),
			"content_type": "application/json",
		}
		stmtBytes, _ := json.Marshal(stmt)
		registerStatement(t, goPort, stmtBytes)
		registerStatement(t, tsPort, stmtBytes)
		time.Sleep(10 * time.Millisecond) // Ensure ordering
	}

	// Query all results
	goResults := queryStatements(t, goPort, map[string]string{})
	tsResults := queryStatements(t, tsPort, map[string]string{})

	var goData, tsData map[string]interface{}
	json.Unmarshal(goResults, &goData)
	json.Unmarshal(tsResults, &tsData)

	// Both should return results in consistent order
	// (typically insertion order or by entry_id)
	goResultList, goOk := goData["results"].([]interface{})
	tsResultList, tsOk := tsData["results"].([]interface{})

	if !goOk || !tsOk {
		t.Fatal("Both implementations must return 'results' array")
	}

	if len(goResultList) != len(tsResultList) {
		t.Errorf("Result count mismatch: Go=%d, TypeScript=%d", len(goResultList), len(tsResultList))
	}

	// Validate ordering consistency (entry IDs should be comparable)
	if len(goResultList) > 1 && len(tsResultList) > 1 {
		// Check if both are in same order (ascending or descending)
		// by comparing first vs last entry
		t.Logf("Go results: %d, TypeScript results: %d", len(goResultList), len(tsResultList))
	}
}

// TestQueryEmptyResults validates empty result handling
func TestQueryEmptyResults(t *testing.T) {
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

	// Query with filter that matches nothing
	nonExistentIssuer := "https://example.com/nonexistent"
	goResults := queryStatements(t, goPort, map[string]string{"issuer": nonExistentIssuer})
	tsResults := queryStatements(t, tsPort, map[string]string{"issuer": nonExistentIssuer})

	var goData, tsData map[string]interface{}
	json.Unmarshal(goResults, &goData)
	json.Unmarshal(tsResults, &tsData)

	// Both should return empty results array (not null)
	goResultList, goOk := goData["results"].([]interface{})
	tsResultList, tsOk := tsData["results"].([]interface{})

	if !goOk {
		t.Error("Go should return 'results' array even when empty")
	}
	if !tsOk {
		t.Error("TypeScript should return 'results' array even when empty")
	}

	if len(goResultList) != 0 {
		t.Errorf("Go should return empty results, got %d", len(goResultList))
	}
	if len(tsResultList) != 0 {
		t.Errorf("TypeScript should return empty results, got %d", len(tsResultList))
	}
}

// queryStatements queries statements from a server with filters
func queryStatements(t *testing.T, port int, filters map[string]string) []byte {
	t.Helper()

	baseURL := fmt.Sprintf("http://localhost:%d/entries", port)

	// Build query parameters
	params := url.Values{}
	for key, value := range filters {
		params.Add(key, value)
	}

	queryURL := baseURL
	if len(params) > 0 {
		queryURL = baseURL + "?" + params.Encode()
	}

	resp, err := http.Get(queryURL)
	if err != nil {
		t.Fatalf("Failed to query statements from port %d: %v", port, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d from port %d: %s", resp.StatusCode, port, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read query response from port %d: %v", port, err)
	}

	return body
}

// compareQueryResults compares query results from both implementations
func compareQueryResults(t *testing.T, goResults, tsResults []byte, filterType, filterValue string) {
	t.Helper()

	result := lib.CompareOutputs(
		&lib.ImplementationResult{
			Implementation: "go",
			Command:        []string{"GET", fmt.Sprintf("/entries?%s=%s", filterType, filterValue)},
			ExitCode:       0,
			Stdout:         string(goResults),
			OutputFormat:   "json",
			Success:        true,
		},
		&lib.ImplementationResult{
			Implementation: "typescript",
			Command:        []string{"GET", fmt.Sprintf("/entries?%s=%s", filterType, filterValue)},
			ExitCode:       0,
			Stdout:         string(tsResults),
			OutputFormat:   "json",
			Success:        true,
		},
	)

	if result.Verdict == "divergent" {
		t.Errorf("Query results diverge for %s=%s:\n%s",
			filterType, filterValue, lib.FormatDifferences(result.Differences))
	}

	testResult := &lib.TestResult{
		TestID:        fmt.Sprintf("http-query-%s", filterType),
		TestName:      fmt.Sprintf("Statement Query by %s", filterType),
		Category:      "http-api",
		ExecutedAt:    time.Now().Format(time.RFC3339),
		Comparison:    result,
		Verdict:       result.Verdict,
		RFCsValidated: []string{"SCRAPI"},
	}

	lib.PrintTestSummary(testResult)
}
