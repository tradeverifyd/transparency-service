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
	"testing"
	"time"

	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/config"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/server"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/database"
)

// TestEndToEndFlow tests the complete transparency service workflow
func TestEndToEndFlow(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()
	cfg, apiKey, err := setupTestService(t, tmpDir)
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

	// Test 2: Register first statement
	var firstEntryID int64
	t.Run("register first statement", func(t *testing.T) {
		statement := createTestStatement(t, "artifact-1")

		req := httptest.NewRequest(http.MethodPost, "/entries", bytes.NewReader(statement))
		req.Header.Set("Content-Type", "application/cose")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			body, _ := io.ReadAll(w.Body)
			t.Fatalf("expected status 201, got %d: %s", w.Code, string(body))
		}

		// Response is a COSE Sign1 receipt (application/cose)
		if w.Header().Get("Content-Type") != "application/cose" {
			t.Errorf("expected Content-Type application/cose, got %s", w.Header().Get("Content-Type"))
		}

		receipt := w.Body.Bytes()
		if len(receipt) == 0 {
			t.Fatal("expected non-empty receipt")
		}

		// Decode receipt to verify it's valid COSE
		_, err := cose.DecodeCoseSign1(receipt)
		if err != nil {
			t.Fatalf("failed to decode receipt: %v", err)
		}

		// Entry IDs are sequential starting from 0
		firstEntryID = 0
	})

	// Test 3: Get receipt for first statement
	t.Run("get receipt for first statement", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/entries/%d", firstEntryID), nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		// Receipt is COSE binary (application/cose)
		receipt := w.Body.Bytes()
		if len(receipt) == 0 {
			t.Fatal("expected non-empty receipt")
		}

		// Verify receipt is valid COSE
		_, err := cose.DecodeCoseSign1(receipt)
		if err != nil {
			t.Fatalf("failed to decode receipt: %v", err)
		}
	})

	// Test 4: Register multiple statements
	entryIDs := []int64{firstEntryID}
	t.Run("register multiple statements", func(t *testing.T) {
		for i := 2; i <= 10; i++ {
			statement := createTestStatement(t, fmt.Sprintf("artifact-%d", i))

			req := httptest.NewRequest(http.MethodPost, "/entries", bytes.NewReader(statement))
			req.Header.Set("Content-Type", "application/cose")
			req.Header.Set("Authorization", "Bearer "+apiKey)
			w := httptest.NewRecorder()
			srv.Handler().ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				body, _ := io.ReadAll(w.Body)
				t.Errorf("statement %d: expected status 201, got %d: %s", i, w.Code, string(body))
				continue
			}

			// Verify receipt is valid COSE
			receipt := w.Body.Bytes()
			if _, err := cose.DecodeCoseSign1(receipt); err != nil {
				t.Errorf("statement %d: invalid receipt: %v", i, err)
			}

			// Entry IDs are sequential (0, 1, 2, ...)
			entryID := int64(i - 1)
			entryIDs = append(entryIDs, entryID)
		}
	})

	// Test 5: Verify all receipts are retrievable
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

	// Test 6: Verify database state
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

	// Test 7: Test SCITT configuration
	t.Run("get SCITT configuration", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/.well-known/scitt-configuration", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var result map[string]interface{}
		json.NewDecoder(w.Body).Decode(&result)

		if result["issuer"] != cfg.Issuer {
			t.Errorf("expected issuer %s, got %v", cfg.Issuer, result["issuer"])
		}

		algorithms := result["supported_algorithms"].([]interface{})
		if len(algorithms) == 0 {
			t.Error("expected at least one supported algorithm")
		}
	})
}

// Helper functions

func setupTestService(t *testing.T, tmpDir string) (*config.Config, string, error) {
	t.Helper()

	// Generate test keys
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Save private key as COSE CBOR (with kid set automatically)
	privateKeyCBOR, err := cose.ExportPrivateKeyToCOSECBOR(keyPair.Private)
	if err != nil {
		return nil, "", fmt.Errorf("failed to export private key: %w", err)
	}
	privateKeyPath := filepath.Join(tmpDir, "service-key.cbor")
	if err := os.WriteFile(privateKeyPath, privateKeyCBOR, 0600); err != nil {
		return nil, "", fmt.Errorf("failed to write private key: %w", err)
	}

	// Save public key as COSE CBOR (with kid set automatically)
	publicKeyCBOR, err := cose.ExportPublicKeyToCOSECBOR(keyPair.Public)
	if err != nil {
		return nil, "", fmt.Errorf("failed to export public key: %w", err)
	}
	publicKeyPath := filepath.Join(tmpDir, "service-key-pub.cbor")
	if err := os.WriteFile(publicKeyPath, publicKeyCBOR, 0644); err != nil {
		return nil, "", fmt.Errorf("failed to write public key: %w", err)
	}

	// Generate API key for tests
	apiKey, err := config.GenerateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Create config
	cfg := &config.Config{
		Issuer: "https://integration-test.example.com",
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
			Host:   "127.0.0.1",
			Port:   0,
			APIKey: apiKey,
			CORS: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
			},
		},
	}

	return cfg, apiKey, nil
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
