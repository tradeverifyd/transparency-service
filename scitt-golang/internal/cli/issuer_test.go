package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/cli"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
)

func TestIssuerCommand(t *testing.T) {
	rootCmd := cli.NewRootCommand("test", "abc123", "2024-01-01")

	t.Run("has issuer subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == "issuer" {
				found = true
				break
			}
		}
		if !found {
			t.Error("issuer subcommand not found")
		}
	})
}

func TestIssuerKeyCommand(t *testing.T) {
	rootCmd := cli.NewRootCommand("test", "abc123", "2024-01-01")
	issuerCmd, _, err := rootCmd.Find([]string{"issuer"})
	if err != nil {
		t.Fatalf("failed to find issuer command: %v", err)
	}

	t.Run("has key subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range issuerCmd.Commands() {
			if cmd.Name() == "key" {
				found = true
				break
			}
		}
		if !found {
			t.Error("key subcommand not found")
		}
	})
}

func TestIssuerKeyGenerateCommand(t *testing.T) {
	rootCmd := cli.NewRootCommand("test", "abc123", "2024-01-01")
	keyCmd, _, err := rootCmd.Find([]string{"issuer", "key"})
	if err != nil {
		t.Fatalf("failed to find issuer key command: %v", err)
	}

	t.Run("has generate subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range keyCmd.Commands() {
			if cmd.Name() == "generate" {
				found = true
				break
			}
		}
		if !found {
			t.Error("generate subcommand not found")
		}
	})
}

func TestIssuerKeyGenerate(t *testing.T) {
	t.Run("generates default key files", func(t *testing.T) {
		// Create temporary directory
		tmpDir := t.TempDir()
		oldDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		defer os.Chdir(oldDir)

		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to change to temp directory: %v", err)
		}

		// Execute command
		rootCmd := cli.NewRootCommand("test", "abc123", "2024-01-01")
		rootCmd.SetArgs([]string{"issuer", "key", "generate"})

		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("failed to execute command: %v", err)
		}

		// Check that files were created
		if _, err := os.Stat("private_key.cbor"); os.IsNotExist(err) {
			t.Error("private_key.cbor was not created")
		}
		if _, err := os.Stat("public_key.cbor"); os.IsNotExist(err) {
			t.Error("public_key.cbor was not created")
		}

		// Verify files can be read back
		privateData, err := os.ReadFile("private_key.cbor")
		if err != nil {
			t.Fatalf("failed to read private key: %v", err)
		}
		publicData, err := os.ReadFile("public_key.cbor")
		if err != nil {
			t.Fatalf("failed to read public key: %v", err)
		}

		// Verify keys can be imported
		privateKey, err := cose.ImportPrivateKeyFromCOSECBOR(privateData)
		if err != nil {
			t.Fatalf("failed to import private key: %v", err)
		}
		publicKey, err := cose.ImportPublicKeyFromCOSECBOR(publicData)
		if err != nil {
			t.Fatalf("failed to import public key: %v", err)
		}

		// Verify keys match
		if privateKey.X.Cmp(publicKey.X) != 0 {
			t.Error("private and public key X coordinates do not match")
		}
		if privateKey.Y.Cmp(publicKey.Y) != 0 {
			t.Error("private and public key Y coordinates do not match")
		}
	})

	t.Run("generates custom path key files", func(t *testing.T) {
		// Create temporary directory
		tmpDir := t.TempDir()

		privateKeyPath := filepath.Join(tmpDir, "my-private.cbor")
		publicKeyPath := filepath.Join(tmpDir, "my-public.cbor")

		// Execute command
		rootCmd := cli.NewRootCommand("test", "abc123", "2024-01-01")
		rootCmd.SetArgs([]string{
			"issuer", "key", "generate",
			"--private-key", privateKeyPath,
			"--public-key", publicKeyPath,
		})

		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("failed to execute command: %v", err)
		}

		// Check that files were created
		if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
			t.Error("custom private key file was not created")
		}
		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			t.Error("custom public key file was not created")
		}

		// Verify files can be read back
		privateData, err := os.ReadFile(privateKeyPath)
		if err != nil {
			t.Fatalf("failed to read private key: %v", err)
		}
		publicData, err := os.ReadFile(publicKeyPath)
		if err != nil {
			t.Fatalf("failed to read public key: %v", err)
		}

		// Verify keys can be imported
		_, err = cose.ImportPrivateKeyFromCOSECBOR(privateData)
		if err != nil {
			t.Fatalf("failed to import private key: %v", err)
		}
		_, err = cose.ImportPublicKeyFromCOSECBOR(publicData)
		if err != nil {
			t.Fatalf("failed to import public key: %v", err)
		}
	})

	t.Run("generates different keys each time", func(t *testing.T) {
		// Create temporary directory
		tmpDir := t.TempDir()

		// Generate first key pair
		privateKeyPath1 := filepath.Join(tmpDir, "private1.cbor")
		publicKeyPath1 := filepath.Join(tmpDir, "public1.cbor")

		rootCmd1 := cli.NewRootCommand("test", "abc123", "2024-01-01")
		rootCmd1.SetArgs([]string{
			"issuer", "key", "generate",
			"--private-key", privateKeyPath1,
			"--public-key", publicKeyPath1,
		})

		if err := rootCmd1.Execute(); err != nil {
			t.Fatalf("failed to execute command 1: %v", err)
		}

		// Generate second key pair
		privateKeyPath2 := filepath.Join(tmpDir, "private2.cbor")
		publicKeyPath2 := filepath.Join(tmpDir, "public2.cbor")

		rootCmd2 := cli.NewRootCommand("test", "abc123", "2024-01-01")
		rootCmd2.SetArgs([]string{
			"issuer", "key", "generate",
			"--private-key", privateKeyPath2,
			"--public-key", publicKeyPath2,
		})

		if err := rootCmd2.Execute(); err != nil {
			t.Fatalf("failed to execute command 2: %v", err)
		}

		// Read both private keys
		privateData1, err := os.ReadFile(privateKeyPath1)
		if err != nil {
			t.Fatalf("failed to read private key 1: %v", err)
		}
		privateData2, err := os.ReadFile(privateKeyPath2)
		if err != nil {
			t.Fatalf("failed to read private key 2: %v", err)
		}

		// Import both private keys
		privateKey1, err := cose.ImportPrivateKeyFromCOSECBOR(privateData1)
		if err != nil {
			t.Fatalf("failed to import private key 1: %v", err)
		}
		privateKey2, err := cose.ImportPrivateKeyFromCOSECBOR(privateData2)
		if err != nil {
			t.Fatalf("failed to import private key 2: %v", err)
		}

		// Verify keys are different
		if privateKey1.D.Cmp(privateKey2.D) == 0 {
			t.Error("generated identical private keys")
		}
	})
}
