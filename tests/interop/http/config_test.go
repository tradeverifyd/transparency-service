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

// TestTransparencyConfiguration validates GET /.well-known/scitt-configuration
// FR-014: Verify both implementations return equivalent transparency configuration
func TestTransparencyConfiguration(t *testing.T) {
	// Setup test environment
	goDir, tsDir, cleanup := lib.SetupTestEnv(t)
	defer cleanup()

	// Allocate unique ports for both servers
	goPort := lib.GlobalPortAllocator.AllocatePort(t)
	tsPort := lib.GlobalPortAllocator.AllocatePort(t)

	// Start Go server
	goServer := startGoServer(t, goDir, goPort)
	defer goServer.Stop()

	// Start TypeScript server
	tsServer := startTsServer(t, tsDir, tsPort)
	defer tsServer.Stop()

	// Wait for servers to be ready
	waitForServer(t, goPort, 10*time.Second)
	waitForServer(t, tsPort, 10*time.Second)

	// Fetch configuration from Go implementation
	goConfig := fetchTransparencyConfig(t, goPort)

	// Fetch configuration from TypeScript implementation
	tsConfig := fetchTransparencyConfig(t, tsPort)

	// Compare configurations
	result := lib.CompareOutputs(
		&lib.ImplementationResult{
			Implementation: "go",
			Command:        []string{"GET", "/.well-known/scitt-configuration"},
			ExitCode:       0,
			Stdout:         string(goConfig),
			OutputFormat:   "json",
			Success:        true,
		},
		&lib.ImplementationResult{
			Implementation: "typescript",
			Command:        []string{"GET", "/.well-known/scitt-configuration"},
			ExitCode:       0,
			Stdout:         string(tsConfig),
			OutputFormat:   "json",
			Success:        true,
		},
	)

	// Validate specific fields exist and match
	var goData, tsData map[string]interface{}
	if err := json.Unmarshal(goConfig, &goData); err != nil {
		t.Fatalf("Failed to parse Go config: %v", err)
	}
	if err := json.Unmarshal(tsConfig, &tsData); err != nil {
		t.Fatalf("Failed to parse TypeScript config: %v", err)
	}

	// Validate required fields (FR-014)
	requiredFields := []string{"algorithms", "endpoints", "origins"}
	for _, field := range requiredFields {
		if _, exists := goData[field]; !exists {
			t.Errorf("Go config missing required field: %s", field)
		}
		if _, exists := tsData[field]; !exists {
			t.Errorf("TypeScript config missing required field: %s", field)
		}
	}

	// Validate snake_case convention
	goViolations := lib.ValidateSnakeCase(goData)
	if len(goViolations) > 0 {
		t.Errorf("Go config has snake_case violations: %v", goViolations)
	}

	tsViolations := lib.ValidateSnakeCase(tsData)
	if len(tsViolations) > 0 {
		t.Errorf("TypeScript config has snake_case violations: %v", tsViolations)
	}

	// Report comparison results
	if !result.OutputsEquivalent {
		t.Errorf("Transparency configurations are not equivalent:\n%s",
			lib.FormatDifferences(result.Differences))
	}

	// Save test result
	testResult := &lib.TestResult{
		TestID:        "http-config-001",
		TestName:      "Transparency Configuration Compatibility",
		Category:      "http-api",
		ExecutedAt:    time.Now().Format(time.RFC3339),
		GoResult:      result.Differences[0].GoValue,
		TsResult:      result.Differences[0].TsValue,
		Comparison:    result,
		Verdict:       result.Verdict,
		RFCsValidated: []string{"SCRAPI", "Project Requirements"},
	}

	if err := lib.SaveTestResult(testResult, goDir+"/../reports"); err != nil {
		t.Logf("Warning: Failed to save test result: %v", err)
	}

	lib.PrintTestSummary(testResult)
}

// TestTransparencyConfigurationAlgorithms validates algorithm specifications
func TestTransparencyConfigurationAlgorithms(t *testing.T) {
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

	goConfig := fetchTransparencyConfig(t, goPort)
	tsConfig := fetchTransparencyConfig(t, tsPort)

	var goData, tsData map[string]interface{}
	json.Unmarshal(goConfig, &goData)
	json.Unmarshal(tsConfig, &tsData)

	// Validate algorithm specifications match
	goAlgos, goOk := goData["algorithms"].(map[string]interface{})
	tsAlgos, tsOk := tsData["algorithms"].(map[string]interface{})

	if !goOk || !tsOk {
		t.Fatal("Both implementations must have 'algorithms' field")
	}

	// Expected algorithms
	expectedAlgos := []string{"signature", "hash", "merkle"}
	for _, algo := range expectedAlgos {
		if _, exists := goAlgos[algo]; !exists {
			t.Errorf("Go config missing algorithm: %s", algo)
		}
		if _, exists := tsAlgos[algo]; !exists {
			t.Errorf("TypeScript config missing algorithm: %s", algo)
		}
	}

	// Validate ES256 signature algorithm
	if goSig, ok := goAlgos["signature"].(string); ok {
		if goSig != "ES256" && goSig != "ECDSA-P256-SHA256" {
			t.Errorf("Go signature algorithm unexpected: %s", goSig)
		}
	}

	if tsSig, ok := tsAlgos["signature"].(string); ok {
		if tsSig != "ES256" && tsSig != "ECDSA-P256-SHA256" {
			t.Errorf("TypeScript signature algorithm unexpected: %s", tsSig)
		}
	}
}

// TestTransparencyConfigurationEndpoints validates endpoint specifications
func TestTransparencyConfigurationEndpoints(t *testing.T) {
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

	goConfig := fetchTransparencyConfig(t, goPort)
	tsConfig := fetchTransparencyConfig(t, tsPort)

	var goData, tsData map[string]interface{}
	json.Unmarshal(goConfig, &goData)
	json.Unmarshal(tsConfig, &tsData)

	// Validate endpoint specifications
	goEndpoints, goOk := goData["endpoints"].(map[string]interface{})
	tsEndpoints, tsOk := tsData["endpoints"].(map[string]interface{})

	if !goOk || !tsOk {
		t.Fatal("Both implementations must have 'endpoints' field")
	}

	// Expected endpoints
	expectedEndpoints := []string{"register", "entries", "checkpoint", "health"}
	for _, endpoint := range expectedEndpoints {
		if _, exists := goEndpoints[endpoint]; !exists {
			t.Errorf("Go config missing endpoint: %s", endpoint)
		}
		if _, exists := tsEndpoints[endpoint]; !exists {
			t.Errorf("TypeScript config missing endpoint: %s", endpoint)
		}
	}
}

// fetchTransparencyConfig retrieves the transparency configuration from a server
// Tries both standard paths (correct: scitt-configuration, legacy: transparency-configuration)
func fetchTransparencyConfig(t *testing.T, port int) []byte {
	t.Helper()

	// Try both endpoint paths (correct standard is scitt-configuration per SCRAPI)
	urls := []string{
		fmt.Sprintf("http://localhost:%d/.well-known/scitt-configuration", port),
		fmt.Sprintf("http://localhost:%d/.well-known/transparency-configuration", port), // legacy fallback
	}

	for _, url := range urls {
		resp, err := http.Get(url)
		if err != nil {
			continue // Try next URL
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body from port %d: %v", port, err)
			}
			return body
		}
	}

	t.Fatalf("Failed to fetch config from port %d (tried both standard paths)", port)
	return nil
}

// waitForServer waits for a server to become ready
func waitForServer(t *testing.T, port int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("http://localhost:%d/health", port)

	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("Server on port %d did not become ready within %v", port, timeout)
}

// Server represents a running SCITT server
type Server struct {
	Port    int
	Stop    func()
	BaseURL string
}

// startGoServer starts the Go SCITT server
func startGoServer(t *testing.T, workDir string, port int) *Server {
	t.Helper()

	serverProcess, err := lib.StartGoServer(t, workDir, port)
	if err != nil {
		t.Fatalf("Failed to start Go server: %v", err)
	}

	return &Server{
		Port:    port,
		BaseURL: serverProcess.GetBaseURL(),
		Stop:    func() { serverProcess.Stop() },
	}
}

// startTsServer starts the TypeScript SCITT server
func startTsServer(t *testing.T, workDir string, port int) *Server {
	t.Helper()

	serverProcess, err := lib.StartTsServer(t, workDir, port)
	if err != nil {
		t.Fatalf("Failed to start TypeScript server: %v", err)
	}

	return &Server{
		Port:    port,
		BaseURL: serverProcess.GetBaseURL(),
		Stop:    func() { serverProcess.Stop() },
	}
}
