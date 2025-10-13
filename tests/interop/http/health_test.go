package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/tradeverifyd/scitt/tests/interop/lib"
)

// TestHealthCheck validates GET /health endpoint
// FR-015: Verify health endpoint returns 200 OK with equivalent structures
func TestHealthCheck(t *testing.T) {
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

	// Fetch health status
	goHealth := fetchHealth(t, goPort)
	tsHealth := fetchHealth(t, tsPort)

	// Parse responses
	var goData, tsData map[string]interface{}
	if err := json.Unmarshal(goHealth, &goData); err != nil {
		t.Fatalf("Failed to parse Go health response: %v", err)
	}
	if err := json.Unmarshal(tsHealth, &tsData); err != nil {
		t.Fatalf("Failed to parse TypeScript health response: %v", err)
	}

	// Validate status field exists
	if _, exists := goData["status"]; !exists {
		t.Error("Go health response missing 'status' field")
	}
	if _, exists := tsData["status"]; !exists {
		t.Error("TypeScript health response missing 'status' field")
	}

	// Validate status is "healthy" or "ok"
	if status, ok := goData["status"].(string); ok {
		if status != "healthy" && status != "ok" && status != "up" {
			t.Errorf("Go health status unexpected: %s", status)
		}
	}
	if status, ok := tsData["status"].(string); ok {
		if status != "healthy" && status != "ok" && status != "up" {
			t.Errorf("TypeScript health status unexpected: %s", status)
		}
	}

	// Compare health response structures
	result := lib.CompareOutputs(
		&lib.ImplementationResult{
			Implementation: "go",
			Command:        []string{"GET", "/health"},
			ExitCode:       0,
			Stdout:         string(goHealth),
			OutputFormat:   "json",
			Success:        true,
		},
		&lib.ImplementationResult{
			Implementation: "typescript",
			Command:        []string{"GET", "/health"},
			ExitCode:       0,
			Stdout:         string(tsHealth),
			OutputFormat:   "json",
			Success:        true,
		},
	)

	testResult := &lib.TestResult{
		TestID:     "http-health-001",
		TestName:   "GET /health - Health Check",
		Category:   "http-api",
		ExecutedAt: time.Now().Format(time.RFC3339),
		Comparison: result,
		Verdict:    result.Verdict,
		RFCsValidated: []string{"SCRAPI"},
	}

	lib.PrintTestSummary(testResult)
}

// TestHealthCheckResponseTime validates health check response time
func TestHealthCheckResponseTime(t *testing.T) {
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

	// Measure Go health check response time
	goStart := time.Now()
	fetchHealth(t, goPort)
	goDuration := time.Since(goStart)

	// Measure TypeScript health check response time
	tsStart := time.Now()
	fetchHealth(t, tsPort)
	tsDuration := time.Since(tsStart)

	// Health checks should be fast (<100ms typically)
	maxDuration := 1 * time.Second
	if goDuration > maxDuration {
		t.Errorf("Go health check too slow: %v (max: %v)", goDuration, maxDuration)
	}
	if tsDuration > maxDuration {
		t.Errorf("TypeScript health check too slow: %v (max: %v)", tsDuration, maxDuration)
	}

	t.Logf("Health check performance: Go=%v, TypeScript=%v", goDuration, tsDuration)
}

// TestHealthCheckReliability validates health check reliability under load
func TestHealthCheckReliability(t *testing.T) {
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

	// Test Go reliability
	t.Run("go-reliability", func(t *testing.T) {
		successCount := 0
		iterations := 50

		for i := 0; i < iterations; i++ {
			url := fmt.Sprintf("http://localhost:%d/health", goPort)
			resp, err := http.Get(url)
			if err == nil && resp.StatusCode == http.StatusOK {
				successCount++
				resp.Body.Close()
			}
		}

		successRate := float64(successCount) / float64(iterations) * 100
		if successRate < 99.0 {
			t.Errorf("Go health check success rate too low: %.2f%% (expected ≥99%%)", successRate)
		}
	})

	// Test TypeScript reliability
	t.Run("typescript-reliability", func(t *testing.T) {
		successCount := 0
		iterations := 50

		for i := 0; i < iterations; i++ {
			url := fmt.Sprintf("http://localhost:%d/health", tsPort)
			resp, err := http.Get(url)
			if err == nil && resp.StatusCode == http.StatusOK {
				successCount++
				resp.Body.Close()
			}
		}

		successRate := float64(successCount) / float64(iterations) * 100
		if successRate < 99.0 {
			t.Errorf("TypeScript health check success rate too low: %.2f%% (expected ≥99%%)", successRate)
		}
	})
}

// TestHealthCheckConcurrent validates health check under concurrent load
func TestHealthCheckConcurrent(t *testing.T) {
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

	// Test Go concurrent health checks
	t.Run("go-concurrent", func(t *testing.T) {
		done := make(chan bool, concurrency)
		for i := 0; i < concurrency; i++ {
			go func() {
				url := fmt.Sprintf("http://localhost:%d/health", goPort)
				resp, err := http.Get(url)
				if err == nil && resp.StatusCode == http.StatusOK {
					resp.Body.Close()
					done <- true
					return
				}
				done <- false
			}()
		}

		successCount := 0
		for i := 0; i < concurrency; i++ {
			if <-done {
				successCount++
			}
		}

		if successCount != concurrency {
			t.Errorf("Go: Expected %d successful health checks, got %d", concurrency, successCount)
		}
	})

	// Test TypeScript concurrent health checks
	t.Run("typescript-concurrent", func(t *testing.T) {
		done := make(chan bool, concurrency)
		for i := 0; i < concurrency; i++ {
			go func() {
				url := fmt.Sprintf("http://localhost:%d/health", tsPort)
				resp, err := http.Get(url)
				if err == nil && resp.StatusCode == http.StatusOK {
					resp.Body.Close()
					done <- true
					return
				}
				done <- false
			}()
		}

		successCount := 0
		for i := 0; i < concurrency; i++ {
			if <-done {
				successCount++
			}
		}

		if successCount != concurrency {
			t.Errorf("TypeScript: Expected %d successful health checks, got %d", concurrency, successCount)
		}
	})
}

// fetchHealth retrieves the health status from a server
func fetchHealth(t *testing.T, port int) []byte {
	t.Helper()

	url := fmt.Sprintf("http://localhost:%d/health", port)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to fetch health from port %d: %v", port, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d from port %d", resp.StatusCode, port)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read health body from port %d: %v", port, err)
	}

	return body
}
