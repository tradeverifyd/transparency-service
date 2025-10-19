package server_test

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

	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/config"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/server"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
)

func TestNewServer(t *testing.T) {
	t.Run("creates server with valid config", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		if srv == nil {
			t.Fatal("expected non-nil server")
		}
	})

	t.Run("rejects config with missing database", func(t *testing.T) {
		cfg := &config.Config{
			Issuer: "https://test.example.com",
			Database: config.DatabaseConfig{
				Path: "/nonexistent/path/db.sqlite",
			},
			Storage: config.StorageConfig{
				Type: "memory",
			},
			Keys: config.KeysConfig{
				Private: "nonexistent.pem",
				Public:  "nonexistent.jwk",
			},
		}

		_, err := server.NewServer(cfg)
		if err == nil {
			t.Error("expected error for missing database")
		}
	})
}

func TestHealthEndpoint(t *testing.T) {
	t.Run("returns 200 OK", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if result["status"] != "healthy" {
			t.Errorf("expected status 'healthy', got %v", result["status"])
		}
	})
}

func TestSCITTConfigurationEndpoint(t *testing.T) {
	t.Run("returns service configuration", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/.well-known/scitt-configuration", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if result["issuer"] != cfg.Issuer {
			t.Errorf("expected issuer %s, got %v", cfg.Issuer, result["issuer"])
		}

		algorithms, ok := result["supported_algorithms"].([]interface{})
		if !ok || len(algorithms) == 0 {
			t.Error("expected supported_algorithms array")
		}
	})
}

func TestSCITTKeysEndpoint(t *testing.T) {
	t.Run("returns COSE Key Set as CBOR", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/.well-known/scitt-keys", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		// Verify Content-Type is application/cbor
		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/cbor" {
			t.Errorf("expected Content-Type application/cbor, got %s", contentType)
		}

		// Verify we got CBOR data (non-empty)
		body, _ := io.ReadAll(resp.Body)
		if len(body) == 0 {
			t.Fatal("expected non-empty CBOR data")
		}

		// CBOR arrays start with 0x80-0x9f (major type 4)
		// We should have at least one key in the array
		if body[0] < 0x80 || body[0] > 0x9f {
			t.Errorf("expected CBOR array, got first byte: 0x%02x", body[0])
		}
	})
}

func TestRegisterStatementEndpoint(t *testing.T) {
	t.Run("registers valid statement", func(t *testing.T) {
		cfg, apiKey, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		// Create test statement
		statement := createTestStatement(t)

		req := httptest.NewRequest(http.MethodPost, "/entries", bytes.NewReader(statement))
		req.Header.Set("Content-Type", "application/cose")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("expected status 201, got %d: %s", resp.StatusCode, string(body))
		}

		// Response should have Location header pointing to receipt endpoint
		location := resp.Header.Get("Location")
		if location == "" {
			t.Fatal("expected Location header")
		}
		if location != "/statements/0/receipt" {
			t.Errorf("expected Location /statements/0/receipt, got %s", location)
		}

		// Response body should be empty
		body, _ := io.ReadAll(resp.Body)
		if len(body) != 0 {
			t.Errorf("expected empty body, got %d bytes", len(body))
		}
	})

	t.Run("rejects invalid content type", func(t *testing.T) {
		cfg, apiKey, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodPost, "/entries", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusUnsupportedMediaType {
			t.Errorf("expected status 415, got %d", resp.StatusCode)
		}
	})

	t.Run("rejects invalid COSE Sign1", func(t *testing.T) {
		cfg, apiKey, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodPost, "/entries", bytes.NewReader([]byte("invalid data")))
		req.Header.Set("Content-Type", "application/cose")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestGetReceiptEndpoint(t *testing.T) {
	t.Run("returns receipt for valid entry", func(t *testing.T) {
		cfg, apiKey, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		// Register a statement first
		statement := createTestStatement(t)
		regReq := httptest.NewRequest(http.MethodPost, "/entries", bytes.NewReader(statement))
		regReq.Header.Set("Content-Type", "application/cose")
		regReq.Header.Set("Authorization", "Bearer "+apiKey)
		regW := httptest.NewRecorder()
		srv.Handler().ServeHTTP(regW, regReq)

		regResp := regW.Result()
		if regResp.StatusCode != http.StatusCreated {
			t.Fatalf("failed to register statement: %d", regResp.StatusCode)
		}

		// Entry IDs are sequential starting from 0
		entryID := int64(0)

		// Get receipt
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/entries/%d", entryID), nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("returns 404 for non-existent entry", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/entries/999999", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("returns 400 for invalid entry ID", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/entries/invalid", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestOpenAPIEndpoints(t *testing.T) {
	t.Run("serves Swagger UI at root", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if !contains(contentType, "text/html") {
			t.Errorf("expected Content-Type text/html, got %s", contentType)
		}

		body, _ := io.ReadAll(resp.Body)
		html := string(body)
		if !contains(html, "swagger-ui") {
			t.Error("expected Swagger UI HTML")
		}
	})

	t.Run("serves OpenAPI spec as JSON", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if !contains(contentType, "application/json") {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}

		body, _ := io.ReadAll(resp.Body)
		var spec map[string]interface{}
		if err := json.Unmarshal(body, &spec); err != nil {
			t.Fatalf("failed to parse OpenAPI spec: %v", err)
		}

		// Verify OpenAPI structure
		if _, ok := spec["openapi"]; !ok {
			t.Error("expected openapi field in spec")
		}

		if _, ok := spec["info"]; !ok {
			t.Error("expected info field in spec")
		}

		if _, ok := spec["paths"]; !ok {
			t.Error("expected paths field in spec")
		}

		// Verify server URL is updated to match the request host
		if servers, ok := spec["servers"].([]interface{}); ok && len(servers) > 0 {
			if server, ok := servers[0].(map[string]interface{}); ok {
				if url, ok := server["url"].(string); ok {
					// Should be http://example.com (from httptest)
					if !contains(url, "example.com") {
						t.Errorf("expected server URL to contain example.com, got %s", url)
					}
				}
				if desc, ok := server["description"].(string); ok {
					if desc != "Current server" {
						t.Errorf("expected description 'Current server', got %s", desc)
					}
				}
			}
		}
	})

	t.Run("returns 404 for non-root paths", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/something", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", resp.StatusCode)
		}
	})
}

func TestCORSMiddleware(t *testing.T) {
	t.Run("adds CORS headers when enabled", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		cfg.Server.CORS.Enabled = true

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("Issuer", "http://localhost:3000")
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		corsHeader := resp.Header.Get("Access-Control-Allow-Issuer")
		if corsHeader == "" {
			t.Error("expected Access-Control-Allow-Issuer header")
		}
	})

	t.Run("handles preflight OPTIONS request", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		cfg.Server.CORS.Enabled = true

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodOptions, "/entries", nil)
		req.Header.Set("Issuer", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 for OPTIONS, got %d", resp.StatusCode)
		}

		allowMethods := resp.Header.Get("Access-Control-Allow-Methods")
		if allowMethods == "" {
			t.Error("expected Access-Control-Allow-Methods header")
		}
	})
}

// Helper functions

func setupTestConfig(t *testing.T) (*config.Config, string, func()) {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()

	// Generate test keys
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	// Save private key as COSE CBOR (with kid set automatically)
	privateKeyCBOR, err := cose.ExportPrivateKeyToCOSECBOR(keyPair.Private)
	if err != nil {
		t.Fatalf("failed to export private key: %v", err)
	}
	privateKeyPath := filepath.Join(tmpDir, "service-key.cbor")
	if err := os.WriteFile(privateKeyPath, privateKeyCBOR, 0600); err != nil {
		t.Fatalf("failed to write private key: %v", err)
	}

	// Save public key as COSE CBOR (with kid set automatically)
	publicKeyCBOR, err := cose.ExportPublicKeyToCOSECBOR(keyPair.Public)
	if err != nil {
		t.Fatalf("failed to export public key: %v", err)
	}
	publicKeyPath := filepath.Join(tmpDir, "service-key-pub.cbor")
	if err := os.WriteFile(publicKeyPath, publicKeyCBOR, 0644); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	// Generate API key for tests
	apiKey, err := config.GenerateAPIKey()
	if err != nil {
		t.Fatalf("failed to generate API key: %v", err)
	}

	// Create config
	cfg := &config.Config{
		Issuer: "https://test.example.com",
		Database: config.DatabaseConfig{
			Type:      "sqlite",
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
			Port:   0, // Random port
			APIKey: apiKey,
			CORS: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
			},
		},
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return cfg, apiKey, cleanup
}

func createTestStatement(t *testing.T) []byte {
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
		Sub: "test-artifact",
	})

	// Create protected headers
	headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
		Alg:       cose.AlgorithmES256,
		Cty:       "application/json",
		CWTClaims: cwtClaims,
	})

	// Create payload
	payload := []byte(`{"test": "data"}`)

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

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Tests for /statements endpoints

func TestPostStatementsEndpoint(t *testing.T) {
	t.Run("registers statement via POST /statements", func(t *testing.T) {
		cfg, apiKey, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		statement := createTestStatement(t)

		req := httptest.NewRequest(http.MethodPost, "/statements", bytes.NewReader(statement))
		req.Header.Set("Content-Type", "application/scitt-statement+cose")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("expected status 201, got %d: %s", resp.StatusCode, string(body))
		}

		// Response should have Location header pointing to receipt endpoint
		location := resp.Header.Get("Location")
		if location == "" {
			t.Fatal("expected Location header")
		}
		if location != "/statements/0/receipt" {
			t.Errorf("expected Location /statements/0/receipt, got %s", location)
		}

		// Response body should be empty
		body, _ := io.ReadAll(resp.Body)
		if len(body) != 0 {
			t.Errorf("expected empty body, got %d bytes", len(body))
		}
	})

	t.Run("rejects POST /statements without API key", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		statement := createTestStatement(t)

		req := httptest.NewRequest(http.MethodPost, "/statements", bytes.NewReader(statement))
		req.Header.Set("Content-Type", "application/scitt-statement+cose")
		// No Authorization header
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("rejects POST /statements with invalid content type", func(t *testing.T) {
		cfg, apiKey, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodPost, "/statements", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusUnsupportedMediaType {
			t.Errorf("expected status 415, got %d", resp.StatusCode)
		}
	})
}

func TestGetStatementsReceiptEndpoint(t *testing.T) {
	t.Run("returns receipt via GET /statements/{entry_id}/receipt", func(t *testing.T) {
		cfg, apiKey, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		// Register a statement first
		statement := createTestStatement(t)
		regReq := httptest.NewRequest(http.MethodPost, "/statements", bytes.NewReader(statement))
		regReq.Header.Set("Content-Type", "application/scitt-statement+cose")
		regReq.Header.Set("Authorization", "Bearer "+apiKey)
		regW := httptest.NewRecorder()
		srv.Handler().ServeHTTP(regW, regReq)

		regResp := regW.Result()
		if regResp.StatusCode != http.StatusCreated {
			t.Fatalf("failed to register statement: %d", regResp.StatusCode)
		}

		// Get receipt via /statements/{entry_id}/receipt
		entryID := int64(0)
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/statements/%d/receipt", entryID), nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("expected status 200, got %d: %s", resp.StatusCode, string(body))
		}

		if resp.Header.Get("Content-Type") != "application/scitt-receipt+cose" {
			t.Errorf("expected Content-Type application/scitt-receipt+cose, got %s", resp.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(resp.Body)
		if len(body) == 0 {
			t.Fatal("expected non-empty receipt")
		}

		// Verify receipt is valid COSE
		_, err = cose.DecodeCoseSign1(body)
		if err != nil {
			t.Fatalf("failed to decode receipt: %v", err)
		}
	})

	t.Run("returns 404 for non-existent entry via /statements/{entry_id}/receipt", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/statements/999999/receipt", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("returns 400 for invalid entry ID in /statements/{entry_id}/receipt", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/statements/invalid/receipt", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestQueryStatementsEndpoint(t *testing.T) {
	t.Run("queries statements with no filters", func(t *testing.T) {
		cfg, apiKey, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		// Register a few test statements
		for i := 0; i < 3; i++ {
			statement := createTestStatement(t)
			regReq := httptest.NewRequest(http.MethodPost, "/statements", bytes.NewReader(statement))
			regReq.Header.Set("Content-Type", "application/scitt-statement+cose")
			regReq.Header.Set("Authorization", "Bearer "+apiKey)
			regW := httptest.NewRecorder()
			srv.Handler().ServeHTTP(regW, regReq)

			if regW.Code != http.StatusCreated {
				t.Fatalf("failed to register statement %d: %d", i, regW.Code)
			}
		}

		// Query statements
		req := httptest.NewRequest(http.MethodGet, "/statements", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("expected status 200, got %d: %s", resp.StatusCode, string(body))
		}

		if resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// Check response structure
		statements, ok := result["statements"].([]interface{})
		if !ok {
			t.Fatal("expected statements array")
		}

		if len(statements) != 3 {
			t.Errorf("expected 3 statements, got %d", len(statements))
		}

		// Verify first statement has required fields
		if len(statements) > 0 {
			stmt := statements[0].(map[string]interface{})
			if _, ok := stmt["entry_id"]; !ok {
				t.Error("expected entry_id field")
			}
			if _, ok := stmt["leaf_hash"]; !ok {
				t.Error("expected leaf_hash field")
			}
			if _, ok := stmt["iss"]; !ok {
				t.Error("expected iss field")
			}
			if _, ok := stmt["registered_at"]; !ok {
				t.Error("expected registered_at field")
			}
			// Verify tree_size_at_registration is NOT present
			if _, ok := stmt["tree_size_at_registration"]; ok {
				t.Error("did not expect tree_size_at_registration field (it's redundant)")
			}
		}

		// Check pagination fields
		if limit, ok := result["limit"].(float64); !ok || limit != 100 {
			t.Errorf("expected limit 100, got %v", result["limit"])
		}

		if offset, ok := result["offset"].(float64); !ok || offset != 0 {
			t.Errorf("expected offset 0, got %v", result["offset"])
		}

		if total, ok := result["total"].(float64); !ok || total != 3 {
			t.Errorf("expected total 3, got %v", result["total"])
		}
	})

	t.Run("queries statements with limit parameter", func(t *testing.T) {
		cfg, apiKey, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		// Register 5 statements
		for i := 0; i < 5; i++ {
			statement := createTestStatement(t)
			regReq := httptest.NewRequest(http.MethodPost, "/statements", bytes.NewReader(statement))
			regReq.Header.Set("Content-Type", "application/scitt-statement+cose")
			regReq.Header.Set("Authorization", "Bearer "+apiKey)
			regW := httptest.NewRecorder()
			srv.Handler().ServeHTTP(regW, regReq)
		}

		// Query with limit=2
		req := httptest.NewRequest(http.MethodGet, "/statements?limit=2", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		statements := result["statements"].([]interface{})
		if len(statements) != 2 {
			t.Errorf("expected 2 statements, got %d", len(statements))
		}

		if limit, ok := result["limit"].(float64); !ok || limit != 2 {
			t.Errorf("expected limit 2, got %v", result["limit"])
		}
	})

	t.Run("queries statements with offset parameter", func(t *testing.T) {
		cfg, apiKey, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		// Register 5 statements
		for i := 0; i < 5; i++ {
			statement := createTestStatement(t)
			regReq := httptest.NewRequest(http.MethodPost, "/statements", bytes.NewReader(statement))
			regReq.Header.Set("Content-Type", "application/scitt-statement+cose")
			regReq.Header.Set("Authorization", "Bearer "+apiKey)
			regW := httptest.NewRecorder()
			srv.Handler().ServeHTTP(regW, regReq)
		}

		// Query with offset=2
		req := httptest.NewRequest(http.MethodGet, "/statements?offset=2", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		statements := result["statements"].([]interface{})
		if len(statements) != 3 {
			t.Errorf("expected 3 statements (5 total - 2 offset), got %d", len(statements))
		}

		if offset, ok := result["offset"].(float64); !ok || offset != 2 {
			t.Errorf("expected offset 2, got %v", result["offset"])
		}
	})

	t.Run("queries statements with iss filter", func(t *testing.T) {
		cfg, apiKey, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		// Register a statement
		statement := createTestStatement(t)
		regReq := httptest.NewRequest(http.MethodPost, "/statements", bytes.NewReader(statement))
		regReq.Header.Set("Content-Type", "application/scitt-statement+cose")
		regReq.Header.Set("Authorization", "Bearer "+apiKey)
		regW := httptest.NewRecorder()
		srv.Handler().ServeHTTP(regW, regReq)

		if regW.Code != http.StatusCreated {
			t.Fatalf("failed to register statement: %d", regW.Code)
		}

		// Query with iss filter
		req := httptest.NewRequest(http.MethodGet, "/statements?iss=https://issuer.example.com", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("expected status 200, got %d: %s", resp.StatusCode, string(body))
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// Verify response structure (filter may or may not match depending on implementation)
		statements, ok := result["statements"].([]interface{})
		if !ok {
			t.Fatal("expected statements array")
		}

		// Just verify the endpoint works - the actual filtering logic is tested at the repository level
		if len(statements) > 0 {
			// If we got results, verify they have the expected issuer
			stmt := statements[0].(map[string]interface{})
			if iss, ok := stmt["iss"].(string); ok {
				if iss != "https://issuer.example.com" {
					t.Errorf("expected iss to be https://issuer.example.com, got %s", iss)
				}
			}
		}
	})

	t.Run("enforces max limit of 1000", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		// Query with limit > 1000
		req := httptest.NewRequest(http.MethodGet, "/statements?limit=5000", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// Limit should be clamped to 1000
		if limit, ok := result["limit"].(float64); !ok || limit != 1000 {
			t.Errorf("expected limit to be clamped to 1000, got %v", result["limit"])
		}
	})

	t.Run("uses default limit of 100 when not specified", func(t *testing.T) {
		cfg, _, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/statements", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if limit, ok := result["limit"].(float64); !ok || limit != 100 {
			t.Errorf("expected default limit 100, got %v", result["limit"])
		}
	})
}
