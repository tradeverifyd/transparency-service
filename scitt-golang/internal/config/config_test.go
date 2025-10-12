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

		if cfg.Origin == "" {
			t.Error("expected non-empty origin")
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
	t.Run("rejects empty origin", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Origin = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("should reject empty origin")
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
			Origin: "https://example.com",
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
		original.Origin = "https://test.example.com"

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
		if loaded.Origin != original.Origin {
			t.Errorf("origin mismatch: expected %s, got %s", original.Origin, loaded.Origin)
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
			Origin: "https://example.com",
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
			Origin: "https://example.com",
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
