package server_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/config"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/server"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
)

func TestCheckpointEndpoint(t *testing.T) {
	cfg, apiKey, cleanup := setupTestConfigWithLocalStorage(t)
	defer cleanup()

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

	// First register a statement so there's an entry to checkpoint
	statement := createTestStatement(t)
	regReq := httptest.NewRequest(http.MethodPost, "/entries", bytes.NewReader(statement))
	regReq.Header.Set("Content-Type", "application/scitt-statement+cose")
	regReq.Header.Set("Authorization", "Bearer "+apiKey)
	regW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(regW, regReq)

	if regW.Code != http.StatusCreated {
		t.Fatalf("Failed to register statement: %d", regW.Code)
	}

	// Test checkpoint endpoint (should return receipt for last entry)
	req := httptest.NewRequest(http.MethodGet, "/checkpoint", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/scitt-receipt+cose" {
		t.Errorf("Expected Content-Type 'application/scitt-receipt+cose', got '%s'", w.Header().Get("Content-Type"))
	}

	body := w.Body.Bytes()
	if len(body) == 0 {
		t.Error("Expected non-empty checkpoint receipt")
	}

	// Verify it's a valid COSE Sign1
	_, err = cose.DecodeCoseSign1(body)
	if err != nil {
		t.Fatalf("Failed to decode checkpoint receipt: %v", err)
	}

	t.Logf("Checkpoint receipt size: %d bytes", len(body))
}

func TestTileEndpoint(t *testing.T) {
	cfg, _, cleanup := setupTestConfigWithLocalStorage(t)
	defer cleanup()

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

	// Create a test entry tile with some data
	storagePath := cfg.Storage.Path
	tilePath := filepath.Join(storagePath, "tile", "entries", "000")
	if err := os.MkdirAll(filepath.Dir(tilePath), 0755); err != nil {
		t.Fatalf("Failed to create tile directory: %v", err)
	}

	// Write test data (32 bytes = 1 hash)
	testHash := make([]byte, 32)
	for i := range testHash {
		testHash[i] = byte(i)
	}
	if err := os.WriteFile(tilePath, testHash, 0644); err != nil {
		t.Fatalf("Failed to write test tile: %v", err)
	}

	// Test entry tile endpoint
	req := httptest.NewRequest(http.MethodGet, "/tile/entries/000", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/octet-stream" {
		t.Errorf("Expected Content-Type 'application/octet-stream', got '%s'", w.Header().Get("Content-Type"))
	}

	if w.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Errorf("Expected Cache-Control 'public, max-age=31536000, immutable', got '%s'", w.Header().Get("Cache-Control"))
	}

	body := w.Body.Bytes()
	if len(body) != 32 {
		t.Errorf("Expected 32 bytes, got %d", len(body))
	}
}

func TestTileEndpointNotFound(t *testing.T) {
	cfg, _, cleanup := setupTestConfigWithLocalStorage(t)
	defer cleanup()

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

	// Test non-existent tile
	req := httptest.NewRequest(http.MethodGet, "/tile/entries/999", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Helper function to setup test config with local storage
func setupTestConfigWithLocalStorage(t *testing.T) (*config.Config, string, func()) {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()

	// Generate test keys
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	// Save private key as COSE CBOR
	privateKeyCBOR, err := cose.ExportPrivateKeyToCOSECBOR(keyPair.Private)
	if err != nil {
		t.Fatalf("failed to export private key: %v", err)
	}
	privateKeyPath := filepath.Join(tmpDir, "service-key.cbor")
	if err := os.WriteFile(privateKeyPath, privateKeyCBOR, 0600); err != nil {
		t.Fatalf("failed to write private key: %v", err)
	}

	// Save public key as COSE CBOR
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

	// Create config with local storage (not memory)
	cfg := &config.Config{
		Issuer: "https://test.example.com",
		Database: config.DatabaseConfig{
			Type:      "sqlite",
			Path:      filepath.Join(tmpDir, "test.db"),
			EnableWAL: true,
		},
		Storage: config.StorageConfig{
			Type: "local",
			Path: filepath.Join(tmpDir, "storage"),
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
