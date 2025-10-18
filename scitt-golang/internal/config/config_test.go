package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/config"
)

// TestDefaultConfig tests default configuration
func TestDefaultConfig(t *testing.T) {
	t.Run("creates default config", func(t *testing.T) {
		cfg := config.DefaultConfig()

		if cfg == nil {
			t.Fatal("expected non-nil config")
		}

		if cfg.Issuer == "" {
			t.Error("expected non-empty issuer")
		}

		if cfg.Database.Path == "" {
			t.Error("expected non-empty database path")
		}

		if cfg.Storage.Type == "" {
			t.Error("expected non-empty storage type")
		}
	})

	t.Run("default config is valid", func(t *testing.T) {
		cfg := config.DefaultConfig()

		err := cfg.Validate()
		if err != nil {
			t.Errorf("default config should be valid: %v", err)
		}
	})
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	t.Run("rejects empty issuer", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Issuer = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("should reject empty issuer")
		}
	})

	t.Run("rejects empty database path", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Database.Path = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("should reject empty database path")
		}
	})

	t.Run("rejects empty storage type", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Storage.Type = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("should reject empty storage type")
		}
	})

	t.Run("rejects local storage without path", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Storage.Type = "local"
		cfg.Storage.Path = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("should reject local storage without path")
		}
	})

	t.Run("rejects S3 storage without config", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Storage.Type = "s3"
		cfg.Storage.S3 = nil

		err := cfg.Validate()
		if err == nil {
			t.Error("should reject S3 storage without config")
		}
	})

	t.Run("rejects invalid port", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Server.Port = 0

		err := cfg.Validate()
		if err == nil {
			t.Error("should reject port 0")
		}

		cfg.Server.Port = 99999
		err = cfg.Validate()
		if err == nil {
			t.Error("should reject port > 65535")
		}
	})

	t.Run("accepts valid config", func(t *testing.T) {
		cfg := &config.Config{
			Issuer: "https://example.com",
			Database: config.DatabaseConfig{
				Path:      "test.db",
				EnableWAL: true,
			},
			Storage: config.StorageConfig{
				Type: "local",
				Path: "./storage",
			},
			Keys: config.KeysConfig{
				Private: "key.pem",
				Public:  "key.jwk",
			},
			Server: config.ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("valid config should pass validation: %v", err)
		}
	})
}

// TestConfigSaveLoad tests saving and loading configuration
func TestConfigSaveLoad(t *testing.T) {
	t.Run("can save and load config", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		original := config.DefaultConfig()
		original.Issuer = "https://test.example.com"

		// Save config
		err := config.SaveConfig(original, configPath)
		if err != nil {
			t.Fatalf("failed to save config: %v", err)
		}

		// Load config
		loaded, err := config.LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		// Verify
		if loaded.Issuer != original.Issuer {
			t.Errorf("issuer mismatch: expected %s, got %s", original.Issuer, loaded.Issuer)
		}

		if loaded.Database.Path != original.Database.Path {
			t.Errorf("database path mismatch")
		}

		if loaded.Storage.Type != original.Storage.Type {
			t.Errorf("storage type mismatch")
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := config.LoadConfig("/nonexistent/config.yaml")
		if err == nil {
			t.Error("should return error for non-existent file")
		}
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "bad.yaml")

		// Write invalid YAML
		_ = os.WriteFile(configPath, []byte("invalid: yaml: content: [[["), 0644)

		_, err := config.LoadConfig(configPath)
		if err == nil {
			t.Error("should return error for invalid YAML")
		}
	})
}

// TestStorageConfig tests storage configuration
func TestStorageConfig(t *testing.T) {
	t.Run("supports local storage", func(t *testing.T) {
		cfg := &config.Config{
			Issuer: "https://example.com",
			Database: config.DatabaseConfig{
				Path: "test.db",
			},
			Storage: config.StorageConfig{
				Type: "local",
				Path: "./storage",
			},
			Keys: config.KeysConfig{
				Private: "key.pem",
				Public:  "key.jwk",
			},
			Server: config.ServerConfig{
				Port: 8080,
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("local storage config should be valid: %v", err)
		}
	})

	t.Run("supports memory storage", func(t *testing.T) {
		cfg := &config.Config{
			Issuer: "https://example.com",
			Database: config.DatabaseConfig{
				Path: "test.db",
			},
			Storage: config.StorageConfig{
				Type: "memory",
			},
			Keys: config.KeysConfig{
				Private: "key.pem",
				Public:  "key.jwk",
			},
			Server: config.ServerConfig{
				Port: 8080,
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("memory storage config should be valid: %v", err)
		}
	})
}

// TestCORSConfig tests CORS configuration
func TestCORSConfig(t *testing.T) {
	t.Run("supports CORS configuration", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Server.CORS.Enabled = true
		cfg.Server.CORS.AllowedOrigins = []string{
			"https://example.com",
			"https://another.com",
		}

		if !cfg.Server.CORS.Enabled {
			t.Error("CORS should be enabled")
		}

		if len(cfg.Server.CORS.AllowedOrigins) != 2 {
			t.Errorf("expected 2 allowed origins, got %d", len(cfg.Server.CORS.AllowedOrigins))
		}
	})
}

// TestEnvironmentVariableExpansion tests environment variable expansion in config
func TestEnvironmentVariableExpansion(t *testing.T) {
	t.Run("expands MongoDB URI from environment variable", func(t *testing.T) {
		// Set environment variables
		os.Setenv("TEST_MONGODB_URI", "mongodb://testuser:testpass@localhost:27017")
		os.Setenv("TEST_MONGODB_DB", "test_database")
		os.Setenv("TEST_API_KEY", "test-api-key-12345")
		defer os.Unsetenv("TEST_MONGODB_URI")
		defer os.Unsetenv("TEST_MONGODB_DB")
		defer os.Unsetenv("TEST_API_KEY")

		// Create config with env var references
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "test-config.yaml")
		configYAML := `issuer: http://127.0.0.1:56177
database:
  type: mongodb
  mongodb:
    uri: ${TEST_MONGODB_URI}
    database: ${TEST_MONGODB_DB}
storage:
  type: local
  path: ./demo/tiles
keys:
  private: ./demo/priv.cbor
  public: ./demo/pub.cbor
server:
  host: 127.0.0.1
  port: 56177
  api_key: ${TEST_API_KEY}
  cors:
    enabled: true
    allowed_origins:
      - "*"
`
		err := os.WriteFile(configPath, []byte(configYAML), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Load config
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		// Verify environment variables were expanded
		if cfg.Database.MongoDB.URI != "mongodb://testuser:testpass@localhost:27017" {
			t.Errorf("MongoDB URI not expanded correctly, got: %s", cfg.Database.MongoDB.URI)
		}

		if cfg.Database.MongoDB.Database != "test_database" {
			t.Errorf("MongoDB database not expanded correctly, got: %s", cfg.Database.MongoDB.Database)
		}

		if cfg.Server.APIKey != "test-api-key-12345" {
			t.Errorf("API key not expanded correctly, got: %s", cfg.Server.APIKey)
		}
	})

	t.Run("handles unexpanded env vars gracefully", func(t *testing.T) {
		// Ensure the env var doesn't exist
		os.Unsetenv("NONEXISTENT_MONGODB_URI")

		// Create config with env var reference that doesn't exist
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "test-config.yaml")
		configYAML := `issuer: http://127.0.0.1:56177
database:
  type: mongodb
  mongodb:
    uri: ${NONEXISTENT_MONGODB_URI}
    database: test_db
storage:
  type: local
  path: ./demo/tiles
keys:
  private: ./demo/priv.cbor
  public: ./demo/pub.cbor
server:
  host: 127.0.0.1
  port: 56177
  api_key: some-key
  cors:
    enabled: true
    allowed_origins:
      - "*"
`
		err := os.WriteFile(configPath, []byte(configYAML), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Load config - it will keep the unexpanded value
		cfg, err := config.LoadConfig(configPath)

		// Config should load but might fail validation or have unexpanded value
		// The important thing is that secrets aren't logged/exposed
		if err == nil && cfg != nil {
			// If it loaded, the URI should still have the ${...} pattern
			if cfg.Database.MongoDB.URI != "${NONEXISTENT_MONGODB_URI}" {
				t.Logf("Environment variable was handled: %s", cfg.Database.MongoDB.URI)
			}
		}
	})
}
