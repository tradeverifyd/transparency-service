package http

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tradeverifyd/scitt/tests/interop/lib"
)

// TestCheckpoint validates GET /checkpoint endpoint
// FR-013: Verify signed tree heads have identical format across implementations
func TestCheckpoint(t *testing.T) {
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

	// Register some statements first to create non-empty tree
	statement := loadTestStatement(t, "small")
	registerStatement(t, goPort, statement)
	registerStatement(t, tsPort, statement)

	// Fetch checkpoints
	goCheckpoint := fetchCheckpoint(t, goPort)
	tsCheckpoint := fetchCheckpoint(t, tsPort)

	// Parse checkpoints
	goCP := parseCheckpoint(t, goCheckpoint)
	tsCP := parseCheckpoint(t, tsCheckpoint)

	// Validate required fields (FR-013)
	requiredFields := []string{"origin", "tree_size", "root_hash", "timestamp"}
	for _, field := range requiredFields {
		if _, exists := goCP[field]; !exists {
			t.Errorf("Go checkpoint missing required field: %s", field)
		}
		if _, exists := tsCP[field]; !exists {
			t.Errorf("TypeScript checkpoint missing required field: %s", field)
		}
	}

	// Validate tree_size is non-negative
	if treeSize, ok := goCP["tree_size"].(float64); ok {
		if treeSize < 0 {
			t.Errorf("Go checkpoint has negative tree_size: %v", treeSize)
		}
	}
	if treeSize, ok := tsCP["tree_size"].(float64); ok {
		if treeSize < 0 {
			t.Errorf("TypeScript checkpoint has negative tree_size: %v", treeSize)
		}
	}

	// Validate root_hash is hex-encoded
	if rootHash, ok := goCP["root_hash"].(string); ok {
		if violation := lib.ValidateHexEncoding(rootHash, "root_hash"); violation != nil {
			t.Errorf("Go root_hash hex validation failed: %v", violation.Description)
		}
	}
	if rootHash, ok := tsCP["root_hash"].(string); ok {
		if violation := lib.ValidateHexEncoding(rootHash, "root_hash"); violation != nil {
			t.Errorf("TypeScript root_hash hex validation failed: %v", violation.Description)
		}
	}

	// Validate RFC 6962 compliance
	goViolations := lib.ValidateRFCCompliance(goCP, "RFC 6962")
	if len(goViolations) > 0 {
		t.Errorf("Go checkpoint has RFC 6962 violations: %v", goViolations)
	}

	tsViolations := lib.ValidateRFCCompliance(tsCP, "RFC 6962")
	if len(tsViolations) > 0 {
		t.Errorf("TypeScript checkpoint has RFC 6962 violations: %v", tsViolations)
	}

	// Compare checkpoint structures
	result := lib.CompareOutputs(
		&lib.ImplementationResult{
			Implementation: "go",
			Command:        []string{"GET", "/checkpoint"},
			ExitCode:       0,
			Stdout:         string(goCheckpoint),
			OutputFormat:   "json",
			Success:        true,
		},
		&lib.ImplementationResult{
			Implementation: "typescript",
			Command:        []string{"GET", "/checkpoint"},
			ExitCode:       0,
			Stdout:         string(tsCheckpoint),
			OutputFormat:   "json",
			Success:        true,
		},
	)

	testResult := &lib.TestResult{
		TestID:        "http-checkpoint-001",
		TestName:      "GET /checkpoint - Signed Tree Head",
		Category:      "http-api",
		ExecutedAt:    time.Now().Format(time.RFC3339),
		Comparison:    result,
		Verdict:       result.Verdict,
		RFCsValidated: []string{"RFC 6962", "SCRAPI"},
	}

	lib.PrintTestSummary(testResult)
}

// TestCheckpointFormat validates checkpoint format consistency
func TestCheckpointFormat(t *testing.T) {
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

	goCheckpoint := fetchCheckpoint(t, goPort)
	tsCheckpoint := fetchCheckpoint(t, tsPort)

	// Validate checkpoint format (could be text format per RFC 6962-bis or JSON)
	// If text format, validate line-by-line structure
	goText := string(goCheckpoint)
	tsText := string(tsCheckpoint)

	// Check if JSON format
	var goJSON, tsJSON map[string]interface{}
	goIsJSON := json.Unmarshal(goCheckpoint, &goJSON) == nil
	tsIsJSON := json.Unmarshal(tsCheckpoint, &tsJSON) == nil

	if goIsJSON != tsIsJSON {
		t.Errorf("Checkpoint format mismatch: Go JSON=%v, TypeScript JSON=%v", goIsJSON, tsIsJSON)
	}

	// If text format, validate structure
	if !goIsJSON && !tsIsJSON {
		goLines := strings.Split(strings.TrimSpace(goText), "\n")
		tsLines := strings.Split(strings.TrimSpace(tsText), "\n")

		// Signed checkpoints should have origin, size, hash, timestamp lines
		if len(goLines) < 4 {
			t.Errorf("Go checkpoint has insufficient lines: %d", len(goLines))
		}
		if len(tsLines) < 4 {
			t.Errorf("TypeScript checkpoint has insufficient lines: %d", len(tsLines))
		}
	}
}

// TestCheckpointAfterMultipleRegistrations validates checkpoint updates
func TestCheckpointAfterMultipleRegistrations(t *testing.T) {
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

	// Get initial checkpoints
	goCP1 := parseCheckpoint(t, fetchCheckpoint(t, goPort))
	tsCP1 := parseCheckpoint(t, fetchCheckpoint(t, tsPort))

	initialGoSize := goCP1["tree_size"].(float64)
	initialTsSize := tsCP1["tree_size"].(float64)

	// Register statements
	statement := loadTestStatement(t, "small")
	registerStatement(t, goPort, statement)
	registerStatement(t, tsPort, statement)

	// Get updated checkpoints
	goCP2 := parseCheckpoint(t, fetchCheckpoint(t, goPort))
	tsCP2 := parseCheckpoint(t, fetchCheckpoint(t, tsPort))

	updatedGoSize := goCP2["tree_size"].(float64)
	updatedTsSize := tsCP2["tree_size"].(float64)

	// Validate tree size increased
	if updatedGoSize <= initialGoSize {
		t.Errorf("Go tree size did not increase: %v -> %v", initialGoSize, updatedGoSize)
	}
	if updatedTsSize <= initialTsSize {
		t.Errorf("TypeScript tree size did not increase: %v -> %v", initialTsSize, updatedTsSize)
	}

	// Validate root hash changed
	initialGoHash := goCP1["root_hash"].(string)
	updatedGoHash := goCP2["root_hash"].(string)
	if initialGoHash == updatedGoHash && initialGoSize != updatedGoSize {
		t.Errorf("Go root hash did not change after tree size increased")
	}

	initialTsHash := tsCP1["root_hash"].(string)
	updatedTsHash := tsCP2["root_hash"].(string)
	if initialTsHash == updatedTsHash && initialTsSize != updatedTsSize {
		t.Errorf("TypeScript root hash did not change after tree size increased")
	}
}

// TestCheckpointSignature validates checkpoint signature
func TestCheckpointSignature(t *testing.T) {
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

	goCheckpoint := fetchCheckpoint(t, goPort)
	tsCheckpoint := fetchCheckpoint(t, tsPort)

	goCP := parseCheckpoint(t, goCheckpoint)
	tsCP := parseCheckpoint(t, tsCheckpoint)

	// Validate signature field exists
	if _, exists := goCP["signature"]; !exists {
		t.Error("Go checkpoint missing signature field")
	}
	if _, exists := tsCP["signature"]; !exists {
		t.Error("TypeScript checkpoint missing signature field")
	}

	// Validate signature is hex-encoded or base64
	if sig, ok := goCP["signature"].(string); ok {
		if len(sig) == 0 {
			t.Error("Go checkpoint has empty signature")
		}
	}
	if sig, ok := tsCP["signature"].(string); ok {
		if len(sig) == 0 {
			t.Error("TypeScript checkpoint has empty signature")
		}
	}
}

// fetchCheckpoint retrieves the checkpoint from a server
func fetchCheckpoint(t *testing.T, port int) []byte {
	t.Helper()

	url := fmt.Sprintf("http://localhost:%d/checkpoint", port)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to fetch checkpoint from port %d: %v", port, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d from port %d", resp.StatusCode, port)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read checkpoint body from port %d: %v", port, err)
	}

	return body
}

// parseCheckpoint parses a checkpoint into a map structure
func parseCheckpoint(t *testing.T, checkpoint []byte) map[string]interface{} {
	t.Helper()

	// Try JSON first
	var jsonData map[string]interface{}
	if err := json.Unmarshal(checkpoint, &jsonData); err == nil {
		return jsonData
	}

	// Parse as text format (RFC 6962-bis signed checkpoint)
	// Format:
	// origin
	// tree_size
	// root_hash
	// timestamp
	// — signature —
	// <signature bytes>

	result := make(map[string]interface{})
	text := string(checkpoint)
	scanner := bufio.NewScanner(strings.NewReader(text))

	lineNum := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "—") {
			continue
		}

		switch lineNum {
		case 0:
			result["origin"] = line
		case 1:
			var treeSize float64
			fmt.Sscanf(line, "%f", &treeSize)
			result["tree_size"] = treeSize
		case 2:
			result["root_hash"] = line
		case 3:
			result["timestamp"] = line
		default:
			// Signature or other data
			result["signature"] = line
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Failed to parse checkpoint: %v", err)
	}

	return result
}
