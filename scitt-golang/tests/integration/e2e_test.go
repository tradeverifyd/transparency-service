package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/config"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/server"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/database"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/merkle"
)

// TestEndToEndFlow tests the complete transparency service workflow
func TestEndToEndFlow(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()
	cfg, err := setupTestService(t, tmpDir)
	if err != nil {
		t.Fatalf("failed to setup test service: %v", err)
	}

	// Create server
	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Close()

	// Test 1: Health check
	t.Run("health check", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var result map[string]interface{}
		json.NewDecoder(w.Body).Decode(&result)
		if result["status"] != "healthy" {
			t.Errorf("expected healthy status, got %v", result["status"])
		}
	})

	// Test 2: Get initial checkpoint (empty tree)
	var initialCheckpoint string
	t.Run("get initial checkpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/checkpoint", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		initialCheckpoint = w.Body.String()
		if !strings.Contains(initialCheckpoint, cfg.Origin) {
			t.Errorf("checkpoint should contain origin")
		}

		// Parse checkpoint to verify format
		lines := strings.Split(strings.TrimSpace(initialCheckpoint), "\n")
		if len(lines) < 4 {
			t.Errorf("checkpoint should have at least 4 lines, got %d", len(lines))
		}

		// Line 1: origin
		if lines[0] != cfg.Origin {
			t.Errorf("expected origin %s, got %s", cfg.Origin, lines[0])
		}

		// Line 2: tree size (should be 0)
		if lines[1] != "0" {
			t.Errorf("expected tree size 0, got %s", lines[1])
		}
	})

	// Test 3: Register first statement
	var firstEntryID int64
	var firstStatementHash string
	t.Run("register first statement", func(t *testing.T) {
		statement := createTestStatement(t, "artifact-1")

		req := httptest.NewRequest(http.MethodPost, "/entries", bytes.NewReader(statement))
		req.Header.Set("Content-Type", "application/cose")
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			body, _ := io.ReadAll(w.Body)
			t.Fatalf("expected status 201, got %d: %s", w.Code, string(body))
		}

		var result map[string]interface{}
		json.NewDecoder(w.Body).Decode(&result)

		firstEntryID = int64(result["entry_id"].(float64))
		firstStatementHash = result["statement_hash"].(string)

		if firstStatementHash == "" {
			t.Error("expected non-empty statement hash")
		}
	})

	// Test 4: Get checkpoint after first statement
	t.Run("get checkpoint after first statement", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/checkpoint", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		checkpoint := w.Body.String()
		lines := strings.Split(strings.TrimSpace(checkpoint), "\n")

		// Tree size should now be 1
		if lines[1] != "1" {
			t.Errorf("expected tree size 1, got %s", lines[1])
		}

		// Checkpoint should be different from initial
		if checkpoint == initialCheckpoint {
			t.Error("checkpoint should change after adding statement")
		}
	})

	// Test 5: Get receipt for first statement
	t.Run("get receipt for first statement", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/entries/%d", firstEntryID), nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		receipt := w.Body.String()
		if !strings.Contains(receipt, fmt.Sprintf("%d", firstEntryID)) {
			t.Errorf("receipt should contain entry ID")
		}
	})

	// Test 6: Register multiple statements
	entryIDs := []int64{firstEntryID}
	t.Run("register multiple statements", func(t *testing.T) {
		for i := 2; i <= 10; i++ {
			statement := createTestStatement(t, fmt.Sprintf("artifact-%d", i))

			req := httptest.NewRequest(http.MethodPost, "/entries", bytes.NewReader(statement))
			req.Header.Set("Content-Type", "application/cose")
			w := httptest.NewRecorder()
			srv.Handler().ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Errorf("statement %d: expected status 201, got %d", i, w.Code)
				continue
			}

			var result map[string]interface{}
			json.NewDecoder(w.Body).Decode(&result)
			entryID := int64(result["entry_id"].(float64))
			entryIDs = append(entryIDs, entryID)
		}
	})

	// Test 7: Verify final tree size
	t.Run("verify final tree size", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/checkpoint", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		checkpoint := w.Body.String()
		lines := strings.Split(strings.TrimSpace(checkpoint), "\n")

		// Tree size should be 10
		if lines[1] != "10" {
			t.Errorf("expected tree size 10, got %s", lines[1])
		}
	})

	// Test 8: Verify all receipts are retrievable
	t.Run("verify all receipts retrievable", func(t *testing.T) {
		for _, entryID := range entryIDs {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/entries/%d", entryID), nil)
			w := httptest.NewRecorder()
			srv.Handler().ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("entry %d: expected status 200, got %d", entryID, w.Code)
			}
		}
	})

	// Test 9: Verify database state
	t.Run("verify database state", func(t *testing.T) {
		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      cfg.Database.Path,
			EnableWAL: cfg.Database.EnableWAL,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		// Check tree size
		treeSize, err := database.GetCurrentTreeSize(db)
		if err != nil {
			t.Fatalf("failed to get tree size: %v", err)
		}

		if treeSize != 10 {
			t.Errorf("expected tree size 10, got %d", treeSize)
		}

		// Check statement count
		statements, err := database.FindStatementsBy(db, database.StatementQueryFilters{})
		if err != nil {
			t.Fatalf("failed to query statements: %v", err)
		}

		if len(statements) != 10 {
			t.Errorf("expected 10 statements, got %d", len(statements))
		}
	})

	// Test 10: Test transparency configuration
	t.Run("get transparency configuration", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/.well-known/transparency-configuration", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var result map[string]interface{}
		json.NewDecoder(w.Body).Decode(&result)

		if result["origin"] != cfg.Origin {
			t.Errorf("expected origin %s, got %v", cfg.Origin, result["origin"])
		}

		algorithms := result["supported_algorithms"].([]interface{})
		if len(algorithms) == 0 {
			t.Error("expected at least one supported algorithm")
		}
	})
}

// TestCheckpointVerification tests checkpoint signature verification
func TestCheckpointVerification(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()
	cfg, err := setupTestService(t, tmpDir)
	if err != nil {
		t.Fatalf("failed to setup test service: %v", err)
	}

	// Create server
	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Close()

	// Register a statement to get non-empty tree
	statement := createTestStatement(t, "test-artifact")
	req := httptest.NewRequest(http.MethodPost, "/entries", bytes.NewReader(statement))
	req.Header.Set("Content-Type", "application/cose")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("failed to register statement: %d", w.Code)
	}

	// Get checkpoint
	req = httptest.NewRequest(http.MethodGet, "/checkpoint", nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("failed to get checkpoint: %d", w.Code)
	}

	checkpointStr := w.Body.String()

	// Decode checkpoint
	checkpoint, err := merkle.DecodeCheckpoint(checkpointStr)
	if err != nil {
		t.Fatalf("failed to decode checkpoint: %v", err)
	}

	// Load public key
	jwkData, err := os.ReadFile(cfg.Keys.Public)
	if err != nil {
		t.Fatalf("failed to read public key: %v", err)
	}

	jwk, err := cose.UnmarshalJWK(jwkData)
	if err != nil {
		t.Fatalf("failed to unmarshal JWK: %v", err)
	}

	publicKey, err := cose.ImportPublicKeyFromJWK(jwk)
	if err != nil {
		t.Fatalf("failed to import public key: %v", err)
	}

	// Verify checkpoint signature
	valid, err := merkle.VerifyCheckpoint(checkpoint, publicKey)
	if err != nil {
		t.Fatalf("failed to verify checkpoint: %v", err)
	}

	if !valid {
		t.Error("checkpoint signature should be valid")
	}

	// Verify checkpoint properties
	if checkpoint.TreeSize != 1 {
		t.Errorf("expected tree size 1, got %d", checkpoint.TreeSize)
	}

	if checkpoint.Origin != cfg.Origin {
		t.Errorf("expected origin %s, got %s", cfg.Origin, checkpoint.Origin)
	}

	// Verify timestamp is recent (within last minute)
	now := time.Now().Unix() * 1000 // milliseconds
	if checkpoint.Timestamp < now-60000 || checkpoint.Timestamp > now+1000 {
		t.Errorf("checkpoint timestamp %d is not recent (now: %d)", checkpoint.Timestamp, now)
	}
}

// Helper functions

func setupTestService(t *testing.T, tmpDir string) (*config.Config, error) {
	t.Helper()

	// Generate test keys
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Save private key
	privatePEM, err := cose.ExportPrivateKeyToPEM(keyPair.Private)
	if err != nil {
		return nil, fmt.Errorf("failed to export private key: %w", err)
	}
	privateKeyPath := filepath.Join(tmpDir, "service-key.pem")
	if err := os.WriteFile(privateKeyPath, []byte(privatePEM), 0600); err != nil {
		return nil, fmt.Errorf("failed to write private key: %w", err)
	}

	// Save public key
	publicJWK, err := cose.ExportPublicKeyToJWK(keyPair.Public)
	if err != nil {
		return nil, fmt.Errorf("failed to export public key: %w", err)
	}
	publicJWKBytes, err := cose.MarshalJWK(publicJWK)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JWK: %w", err)
	}
	publicKeyPath := filepath.Join(tmpDir, "service-key.jwk")
	if err := os.WriteFile(publicKeyPath, publicJWKBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write public key: %w", err)
	}

	// Create config
	cfg := &config.Config{
		Origin: "https://integration-test.example.com",
		Database: config.DatabaseConfig{
			Path:      filepath.Join(tmpDir, "test.db"),
			EnableWAL: true,
		},
		Storage: config.StorageConfig{
			Type: "memory",
		},
		Keys: config.KeysConfig{
			Private: privateKeyPath,
			Public:  publicKeyPath,
		},
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 0,
			CORS: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
			},
		},
	}

	return cfg, nil
}

func createTestStatement(t *testing.T, subject string) []byte {
	t.Helper()

	// Generate test key pair
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	// Create signer
	signer, err := cose.NewES256Signer(keyPair.Private)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	// Create CWT claims
	cwtClaims := cose.CreateCWTClaims(cose.CWTClaimsOptions{
		Iss: "https://issuer.example.com",
		Sub: subject,
	})

	// Create protected headers
	headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
		Alg:       cose.AlgorithmES256,
		Cty:       "application/json",
		CWTClaims: cwtClaims,
	})

	// Create payload with timestamp for uniqueness
	payload := []byte(fmt.Sprintf(`{"artifact": "%s", "timestamp": %d}`, subject, time.Now().UnixNano()))

	// Sign
	coseSign1Struct, err := cose.CreateCoseSign1(headers, payload, signer, cose.CoseSign1Options{})
	if err != nil {
		t.Fatalf("failed to create COSE Sign1: %v", err)
	}

	// Encode
	statement, err := cose.EncodeCoseSign1(coseSign1Struct)
	if err != nil {
		t.Fatalf("failed to encode COSE Sign1: %v", err)
	}

	return statement
}
