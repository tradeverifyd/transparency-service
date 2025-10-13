package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/cli"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/config"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
)

func TestServiceCommand(t *testing.T) {
	rootCmd := cli.NewRootCommand("test", "abc123", "2024-01-01")

	t.Run("has service subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == "service" {
				found = true
				break
			}
		}
		if !found {
			t.Error("service subcommand not found")
		}
	})
}

func TestServiceDefinitionCommand(t *testing.T) {
	rootCmd := cli.NewRootCommand("test", "abc123", "2024-01-01")
	serviceCmd, _, err := rootCmd.Find([]string{"service"})
	if err != nil {
		t.Fatalf("failed to find service command: %v", err)
	}

	t.Run("has create subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range serviceCmd.Commands() {
			if cmd.Name() == "create" {
				found = true
				break
			}
		}
		if !found {
			t.Error("create subcommand not found")
		}
	})

	t.Run("has start subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range serviceCmd.Commands() {
			if cmd.Name() == "start" {
				found = true
				break
			}
		}
		if !found {
			t.Error("start subcommand not found")
		}
	})
}

func TestServiceDefinitionCreateCommand(t *testing.T) {
	rootCmd := cli.NewRootCommand("test", "abc123", "2024-01-01")
	createCmd, _, err := rootCmd.Find([]string{"service", "create"})
	if err != nil {
		t.Fatalf("failed to find service create command: %v", err)
	}

	t.Run("create command exists", func(t *testing.T) {
		if createCmd.Name() != "create" {
			t.Error("create command not found")
		}
	})
}

func TestServiceDefinitionCreate(t *testing.T) {
	t.Run("creates service definition with valid inputs", func(t *testing.T) {
		// Create temporary directory
		tmpDir := t.TempDir()

		// Generate keys for testing
		keyPair, err := cose.GenerateES256KeyPair()
		if err != nil {
			t.Fatalf("failed to generate key pair: %v", err)
		}

		privateKeyCBOR, err := cose.ExportPrivateKeyToCOSECBOR(keyPair.Private)
		if err != nil {
			t.Fatalf("failed to export private key: %v", err)
		}

		publicKeyCBOR, err := cose.ExportPublicKeyToCOSECBOR(keyPair.Public)
		if err != nil {
			t.Fatalf("failed to export public key: %v", err)
		}

		privateKeyPath := filepath.Join(tmpDir, "priv.cbor")
		publicKeyPath := filepath.Join(tmpDir, "pub.cbor")

		if err := os.WriteFile(privateKeyPath, privateKeyCBOR, 0600); err != nil {
			t.Fatalf("failed to write private key: %v", err)
		}

		if err := os.WriteFile(publicKeyPath, publicKeyCBOR, 0644); err != nil {
			t.Fatalf("failed to write public key: %v", err)
		}

		// Setup paths
		tileStorage := filepath.Join(tmpDir, "tiles")
		metadataStorage := filepath.Join(tmpDir, "scitt.db")
		definition := filepath.Join(tmpDir, "scitt.yaml")

		// Execute command
		rootCmd := cli.NewRootCommand("test", "abc123", "2024-01-01")
		rootCmd.SetArgs([]string{
			"service", "create",
			"--receipt-issuer", "https://transparency.example",
			"--receipt-signing-key", privateKeyPath,
			"--receipt-verification-key", publicKeyPath,
			"--tile-storage", tileStorage,
			"--metadata-storage", metadataStorage,
			"--definition", definition,
		})

		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("failed to execute command: %v", err)
		}

		// Verify definition file was created
		if _, err := os.Stat(definition); os.IsNotExist(err) {
			t.Error("definition file was not created")
		}

		// Verify tile storage directory was created
		if _, err := os.Stat(tileStorage); os.IsNotExist(err) {
			t.Error("tile storage directory was not created")
		}

		// Verify database file was created
		if _, err := os.Stat(metadataStorage); os.IsNotExist(err) {
			t.Error("metadata storage file was not created")
		}

		// Load and verify configuration
		cfg, err := config.LoadConfig(definition)
		if err != nil {
			t.Fatalf("failed to load configuration: %v", err)
		}

		if cfg.Issuer != "https://transparency.example" {
			t.Errorf("expected issuer https://transparency.example, got %s", cfg.Issuer)
		}

		if cfg.Database.Path != metadataStorage {
			t.Errorf("expected database path %s, got %s", metadataStorage, cfg.Database.Path)
		}

		if cfg.Storage.Path != tileStorage {
			t.Errorf("expected storage path %s, got %s", tileStorage, cfg.Storage.Path)
		}

		if cfg.Keys.Private != privateKeyPath {
			t.Errorf("expected private key path %s, got %s", privateKeyPath, cfg.Keys.Private)
		}

		if cfg.Keys.Public != publicKeyPath {
			t.Errorf("expected public key path %s, got %s", publicKeyPath, cfg.Keys.Public)
		}

		// Verify default server config
		if cfg.Server.Host != "127.0.0.1" {
			t.Errorf("expected default host 127.0.0.1, got %s", cfg.Server.Host)
		}

		if cfg.Server.Port != 56177 {
			t.Errorf("expected default port 56177, got %d", cfg.Server.Port)
		}
	})

	t.Run("fails with invalid receipt issuer URL", func(t *testing.T) {
		tmpDir := t.TempDir()

		rootCmd := cli.NewRootCommand("test", "abc123", "2024-01-01")
		rootCmd.SetArgs([]string{
			"service", "create",
			"--receipt-issuer", "not-a-url",
			"--receipt-signing-key", filepath.Join(tmpDir, "priv.cbor"),
			"--receipt-verification-key", filepath.Join(tmpDir, "pub.cbor"),
			"--tile-storage", filepath.Join(tmpDir, "tiles"),
			"--metadata-storage", filepath.Join(tmpDir, "scitt.db"),
			"--definition", filepath.Join(tmpDir, "scitt.yaml"),
		})

		err := rootCmd.Execute()
		if err == nil {
			t.Error("expected error for invalid URL, got nil")
		}
	})

	t.Run("fails with non-existent signing key", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create public key but not private key
		keyPair, _ := cose.GenerateES256KeyPair()
		publicKeyCBOR, _ := cose.ExportPublicKeyToCOSECBOR(keyPair.Public)
		publicKeyPath := filepath.Join(tmpDir, "pub.cbor")
		os.WriteFile(publicKeyPath, publicKeyCBOR, 0644)

		rootCmd := cli.NewRootCommand("test", "abc123", "2024-01-01")
		rootCmd.SetArgs([]string{
			"service", "create",
			"--receipt-issuer", "https://transparency.example",
			"--receipt-signing-key", filepath.Join(tmpDir, "nonexistent.cbor"),
			"--receipt-verification-key", publicKeyPath,
			"--tile-storage", filepath.Join(tmpDir, "tiles"),
			"--metadata-storage", filepath.Join(tmpDir, "scitt.db"),
			"--definition", filepath.Join(tmpDir, "scitt.yaml"),
		})

		err := rootCmd.Execute()
		if err == nil {
			t.Error("expected error for non-existent signing key, got nil")
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Generate keys
		keyPair, _ := cose.GenerateES256KeyPair()
		privateKeyCBOR, _ := cose.ExportPrivateKeyToCOSECBOR(keyPair.Private)
		publicKeyCBOR, _ := cose.ExportPublicKeyToCOSECBOR(keyPair.Public)

		privateKeyPath := filepath.Join(tmpDir, "priv.cbor")
		publicKeyPath := filepath.Join(tmpDir, "pub.cbor")

		os.WriteFile(privateKeyPath, privateKeyCBOR, 0600)
		os.WriteFile(publicKeyPath, publicKeyCBOR, 0644)

		// Use nested paths
		tileStorage := filepath.Join(tmpDir, "data", "tiles")
		metadataStorage := filepath.Join(tmpDir, "data", "db", "scitt.db")
		definition := filepath.Join(tmpDir, "config", "scitt.yaml")

		rootCmd := cli.NewRootCommand("test", "abc123", "2024-01-01")
		rootCmd.SetArgs([]string{
			"service", "create",
			"--receipt-issuer", "https://transparency.example",
			"--receipt-signing-key", privateKeyPath,
			"--receipt-verification-key", publicKeyPath,
			"--tile-storage", tileStorage,
			"--metadata-storage", metadataStorage,
			"--definition", definition,
		})

		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("failed to execute command: %v", err)
		}

		// Verify nested directories were created
		if _, err := os.Stat(filepath.Join(tmpDir, "data", "tiles")); os.IsNotExist(err) {
			t.Error("nested tile storage directory was not created")
		}

		if _, err := os.Stat(filepath.Join(tmpDir, "data", "db")); os.IsNotExist(err) {
			t.Error("nested database directory was not created")
		}

		if _, err := os.Stat(filepath.Join(tmpDir, "config")); os.IsNotExist(err) {
			t.Error("nested config directory was not created")
		}
	})
}
