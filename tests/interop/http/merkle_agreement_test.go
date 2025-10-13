package http

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/tradeverifyd/scitt/tests/interop/lib"
)

// TestMerkleTreeAgreement validates that both implementations compute the same
// merkle tree root for identical registered statements
func TestMerkleTreeAgreement(t *testing.T) {
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

	// Register the same statements to both implementations
	statements := []string{"small", "medium", "large"}

	t.Logf("Registering %d identical statements to both implementations", len(statements))

	for i, stmtName := range statements {
		statement := loadTestStatementForMerkle(t, stmtName)

		// Register to Go
		goResp := registerStatementForMerkle(t, goPort, statement)
		goEntryID := extractEntryID(t, goResp, "go")
		t.Logf("Go: Registered %s statement with entry_id=%d", stmtName, goEntryID)

		// Register to TypeScript
		tsResp := registerStatementForMerkle(t, tsPort, statement)
		tsEntryID := extractEntryID(t, tsResp, "typescript")
		t.Logf("TypeScript: Registered %s statement with entry_id=%d", stmtName, tsEntryID)

		// Allow for different starting indices but same progression
		// Just log the entry IDs for debugging
		t.Logf("Statement %d: Go entry_id=%d, TypeScript entry_id=%d", i, goEntryID, tsEntryID)
	}

	// Fetch checkpoints after all registrations
	goCheckpoint := fetchCheckpoint(t, goPort)
	tsCheckpoint := fetchCheckpoint(t, tsPort)

	goCP := parseCheckpoint(t, goCheckpoint)
	tsCP := parseCheckpoint(t, tsCheckpoint)

	// Extract tree metadata
	goTreeSize := int(goCP["tree_size"].(float64))
	tsTreeSize := int(tsCP["tree_size"].(float64))
	goRootHash := goCP["root_hash"].(string)
	tsRootHash := tsCP["root_hash"].(string)

	t.Logf("Go tree: size=%d, root_hash=%s", goTreeSize, goRootHash)
	t.Logf("TypeScript tree: size=%d, root_hash=%s", tsTreeSize, tsRootHash)

	// Validate tree sizes match
	if goTreeSize != tsTreeSize {
		t.Errorf("Tree size mismatch: Go=%d, TypeScript=%d", goTreeSize, tsTreeSize)
		t.Errorf("Both implementations registered the same statements but have different tree sizes")
	}

	// Validate root hashes match
	if goRootHash != tsRootHash {
		t.Errorf("Root hash mismatch:")
		t.Errorf("  Go:         %s", goRootHash)
		t.Errorf("  TypeScript: %s", tsRootHash)
		t.Errorf("Both implementations must compute identical merkle tree roots for the same statements")

		// Decode hashes to check if one is hex and one is base64
		goBytes, goErr := hex.DecodeString(goRootHash)
		tsBytes, tsErr := hex.DecodeString(tsRootHash)

		if goErr != nil {
			t.Logf("Go root_hash is not hex-encoded (might be base64)")
		}
		if tsErr != nil {
			t.Logf("TypeScript root_hash is not hex-encoded (might be base64)")
		}

		if goErr == nil && tsErr == nil {
			if bytes.Equal(goBytes, tsBytes) {
				t.Errorf("Root hashes are equal when decoded, but string representations differ")
			}
		}
	} else {
		t.Logf("✓ Merkle tree roots match: %s", goRootHash)
	}

	// Validate both implementations use the same encoding (hex vs base64)
	_, goIsHex := hex.DecodeString(goRootHash)
	_, tsIsHex := hex.DecodeString(tsRootHash)

	if (goIsHex == nil) != (tsIsHex == nil) {
		t.Errorf("Root hash encoding mismatch: Go uses %s, TypeScript uses %s",
			encodingType(goIsHex == nil),
			encodingType(tsIsHex == nil))
	}
}

// TestMerkleTreeRootEncoding validates that both implementations use hex encoding
// per RFC 6962 for merkle tree roots
func TestMerkleTreeRootEncoding(t *testing.T) {
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

	// Register a statement to create non-empty tree
	statement := loadTestStatementForMerkle(t, "small")
	registerStatementForMerkle(t, goPort, statement)
	registerStatementForMerkle(t, tsPort, statement)

	// Fetch checkpoints
	goCheckpoint := fetchCheckpoint(t, goPort)
	tsCheckpoint := fetchCheckpoint(t, tsPort)

	goCP := parseCheckpoint(t, goCheckpoint)
	tsCP := parseCheckpoint(t, tsCheckpoint)

	goRootHash := goCP["root_hash"].(string)
	tsRootHash := tsCP["root_hash"].(string)

	// RFC 6962 Section 2.1 specifies merkle tree hashes should be represented
	// as hex-encoded strings in the signed tree head
	goBytes, goErr := hex.DecodeString(goRootHash)
	tsBytes, tsErr := hex.DecodeString(tsRootHash)

	if goErr != nil {
		t.Errorf("Go root_hash is not hex-encoded: %v (value: %s)", goErr, goRootHash)
		t.Errorf("RFC 6962 requires hex encoding for merkle tree hashes")
	} else {
		t.Logf("✓ Go root_hash is hex-encoded (%d bytes)", len(goBytes))
	}

	if tsErr != nil {
		t.Errorf("TypeScript root_hash is not hex-encoded: %v (value: %s)", tsErr, tsRootHash)
		t.Errorf("RFC 6962 requires hex encoding for merkle tree hashes")
	} else {
		t.Logf("✓ TypeScript root_hash is hex-encoded (%d bytes)", len(tsBytes))
	}

	// Validate hash length (should be 32 bytes for SHA-256)
	if goErr == nil && len(goBytes) != 32 {
		t.Errorf("Go root_hash has incorrect length: %d bytes (expected 32 for SHA-256)", len(goBytes))
	}
	if tsErr == nil && len(tsBytes) != 32 {
		t.Errorf("TypeScript root_hash has incorrect length: %d bytes (expected 32 for SHA-256)", len(tsBytes))
	}
}

// TestMerkleTreeTileAgreement validates that both implementations generate
// identical merkle tree tiles for the same registered statements
func TestMerkleTreeTileAgreement(t *testing.T) {
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

	// Register multiple statements to create tiles
	statements := []string{"small", "medium", "large"}
	for _, stmtName := range statements {
		statement := loadTestStatementForMerkle(t, stmtName)
		registerStatementForMerkle(t, goPort, statement)
		registerStatementForMerkle(t, tsPort, statement)
	}

	// Fetch tile/0 from both implementations
	// Tiles follow the C2SP tlog-tiles naming convention
	goTile := fetchTile(t, goPort, "tile/0/000")
	tsTile := fetchTile(t, tsPort, "tile/0/000")

	if goTile == nil || tsTile == nil {
		t.Skip("Tile endpoints not yet implemented in one or both implementations")
		return
	}

	// Compare tiles byte-for-byte
	if !bytes.Equal(goTile, tsTile) {
		t.Errorf("Tile data mismatch:")
		t.Errorf("  Go tile length:         %d bytes", len(goTile))
		t.Errorf("  TypeScript tile length: %d bytes", len(tsTile))
		t.Errorf("Tiles must be identical for the same set of registered statements")

		// Show first few bytes for debugging
		maxLen := 32
		if len(goTile) < maxLen {
			maxLen = len(goTile)
		}
		if len(tsTile) < maxLen {
			maxLen = len(tsTile)
		}

		if maxLen > 0 {
			t.Logf("Go tile (first %d bytes):         %x", maxLen, goTile[:maxLen])
			t.Logf("TypeScript tile (first %d bytes): %x", maxLen, tsTile[:maxLen])
		}
	} else {
		t.Logf("✓ Tiles match: %d bytes", len(goTile))
	}
}

// Helper functions

func extractEntryID(t *testing.T, respBody []byte, impl string) int {
	t.Helper()

	var data map[string]interface{}
	if err := json.Unmarshal(respBody, &data); err != nil {
		t.Fatalf("Failed to parse %s response: %v", impl, err)
	}

	if id, ok := data["entry_id"].(float64); ok {
		return int(id)
	}

	t.Fatalf("%s response missing entry_id field", impl)
	return -1
}

func loadTestStatementForMerkle(t *testing.T, name string) []byte {
	t.Helper()

	// Load COSE statement from fixtures
	fixturePath := fmt.Sprintf("../fixtures/statements/%s.cose", name)
	statement, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("Failed to load test statement '%s': %v", name, err)
	}

	return statement
}

func registerStatementForMerkle(t *testing.T, port int, statement []byte) []byte {
	t.Helper()

	url := fmt.Sprintf("http://localhost:%d/entries", port)
	resp, err := http.Post(url, "application/cose", bytes.NewReader(statement))
	if err != nil {
		t.Fatalf("Failed to register statement on port %d: %v", port, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response from port %d: %v", port, err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 201 or 200, got %d from port %d: %s", resp.StatusCode, port, string(body))
	}

	return body
}

func fetchTile(t *testing.T, port int, tilePath string) []byte {
	t.Helper()

	url := fmt.Sprintf("http://localhost:%d/%s", port, tilePath)
	resp, err := http.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return body
}

func encodingType(isHex bool) string {
	if isHex {
		return "hex"
	}
	return "base64 or other"
}
