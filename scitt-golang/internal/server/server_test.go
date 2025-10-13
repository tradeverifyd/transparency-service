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
		cfg, cleanup := setupTestConfig(t)
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
			Origin: "https://test.example.com",
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
		cfg, cleanup := setupTestConfig(t)
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
		cfg, cleanup := setupTestConfig(t)
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

		if result["origin"] != cfg.Origin {
			t.Errorf("expected origin %s, got %v", cfg.Origin, result["origin"])
		}

		algorithms, ok := result["supported_algorithms"].([]interface{})
		if !ok || len(algorithms) == 0 {
			t.Error("expected supported_algorithms array")
		}
	})
}

func TestSCITTKeysEndpoint(t *testing.T) {
	t.Run("returns COSE Key Set as CBOR", func(t *testing.T) {
		cfg, cleanup := setupTestConfig(t)
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

func TestCheckpointEndpoint(t *testing.T) {
	t.Run("returns checkpoint", func(t *testing.T) {
		cfg, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/checkpoint", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		checkpoint := string(body)

		// Should start with origin
		if len(checkpoint) == 0 {
			t.Error("expected non-empty checkpoint")
		}

		// Should contain origin URL
		if !contains(checkpoint, cfg.Origin) {
			t.Errorf("checkpoint should contain origin %s", cfg.Origin)
		}
	})

	t.Run("returns text/plain content type", func(t *testing.T) {
		cfg, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/checkpoint", nil)
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		contentType := resp.Header.Get("Content-Type")
		if contentType != "text/plain" {
			t.Errorf("expected Content-Type text/plain, got %s", contentType)
		}
	})
}

func TestRegisterStatementEndpoint(t *testing.T) {
	t.Run("registers valid statement", func(t *testing.T) {
		cfg, cleanup := setupTestConfig(t)
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
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("expected status 201, got %d: %s", resp.StatusCode, string(body))
		}

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if _, ok := result["entry_id"]; !ok {
			t.Error("expected entry_id in response")
		}

		if _, ok := result["statement_hash"]; !ok {
			t.Error("expected statement_hash in response")
		}
	})

	t.Run("rejects invalid content type", func(t *testing.T) {
		cfg, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodPost, "/entries", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("rejects invalid COSE Sign1", func(t *testing.T) {
		cfg, cleanup := setupTestConfig(t)
		defer cleanup()

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodPost, "/entries", bytes.NewReader([]byte("invalid data")))
		req.Header.Set("Content-Type", "application/cose")
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
		cfg, cleanup := setupTestConfig(t)
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
		regW := httptest.NewRecorder()
		srv.Handler().ServeHTTP(regW, regReq)

		regResp := regW.Result()
		if regResp.StatusCode != http.StatusCreated {
			t.Fatalf("failed to register statement: %d", regResp.StatusCode)
		}

		regBody, _ := io.ReadAll(regResp.Body)
		var regResult map[string]interface{}
		json.Unmarshal(regBody, &regResult)
		entryID := int64(regResult["entry_id"].(float64))

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
		cfg, cleanup := setupTestConfig(t)
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
		cfg, cleanup := setupTestConfig(t)
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
		cfg, cleanup := setupTestConfig(t)
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
		cfg, cleanup := setupTestConfig(t)
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

		// Verify server URL is updated to match config origin
		if servers, ok := spec["servers"].([]interface{}); ok && len(servers) > 0 {
			if server, ok := servers[0].(map[string]interface{}); ok {
				if url, ok := server["url"].(string); ok {
					if url != cfg.Origin {
						t.Errorf("expected server URL %s, got %s", cfg.Origin, url)
					}
				}
			}
		}
	})

	t.Run("returns 404 for non-root paths", func(t *testing.T) {
		cfg, cleanup := setupTestConfig(t)
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
		cfg, cleanup := setupTestConfig(t)
		defer cleanup()

		cfg.Server.CORS.Enabled = true

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()

		srv.Handler().ServeHTTP(w, req)

		resp := w.Result()
		corsHeader := resp.Header.Get("Access-Control-Allow-Origin")
		if corsHeader == "" {
			t.Error("expected Access-Control-Allow-Origin header")
		}
	})

	t.Run("handles preflight OPTIONS request", func(t *testing.T) {
		cfg, cleanup := setupTestConfig(t)
		defer cleanup()

		cfg.Server.CORS.Enabled = true

		srv, err := server.NewServer(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}
		defer srv.Close()

		req := httptest.NewRequest(http.MethodOptions, "/entries", nil)
		req.Header.Set("Origin", "http://localhost:3000")
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

func setupTestConfig(t *testing.T) (*config.Config, func()) {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()

	// Generate test keys
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	// Save private key
	privatePEM, err := cose.ExportPrivateKeyToPEM(keyPair.Private)
	if err != nil {
		t.Fatalf("failed to export private key: %v", err)
	}
	privateKeyPath := filepath.Join(tmpDir, "service-key.pem")
	if err := os.WriteFile(privateKeyPath, []byte(privatePEM), 0600); err != nil {
		t.Fatalf("failed to write private key: %v", err)
	}

	// Save public key
	publicJWK, err := cose.ExportPublicKeyToJWK(keyPair.Public)
	if err != nil {
		t.Fatalf("failed to export public key: %v", err)
	}
	publicJWKBytes, err := cose.MarshalJWK(publicJWK)
	if err != nil {
		t.Fatalf("failed to marshal JWK: %v", err)
	}
	publicKeyPath := filepath.Join(tmpDir, "service-key.jwk")
	if err := os.WriteFile(publicKeyPath, publicJWKBytes, 0644); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	// Create config
	cfg := &config.Config{
		Origin: "https://test.example.com",
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
			Port: 0, // Random port
			CORS: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
			},
		},
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return cfg, cleanup
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
