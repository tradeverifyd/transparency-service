package cli_test

import (
	"strings"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/cli"
)

// TestRootCommand tests the root command initialization
func TestRootCommand(t *testing.T) {
	t.Run("creates root command", func(t *testing.T) {
		cmd := cli.NewRootCommand("1.0.0", "abc123", "2025-01-01")

		if cmd == nil {
			t.Fatal("expected non-nil root command")
		}

		if cmd.Use != "scitt" {
			t.Errorf("expected Use 'scitt', got '%s'", cmd.Use)
		}
	})

	t.Run("has version", func(t *testing.T) {
		cmd := cli.NewRootCommand("1.0.0", "abc123", "2025-01-01")

		if cmd.Version == "" {
			t.Error("expected version to be set")
		}

		if !strings.Contains(cmd.Version, "1.0.0") {
			t.Errorf("expected version to contain '1.0.0', got '%s'", cmd.Version)
		}
	})

	t.Run("has verbose flag", func(t *testing.T) {
		cmd := cli.NewRootCommand("1.0.0", "abc123", "2025-01-01")

		flag := cmd.PersistentFlags().Lookup("verbose")
		if flag == nil {
			t.Error("expected verbose flag to exist")
		}
	})

	t.Run("has config flag", func(t *testing.T) {
		cmd := cli.NewRootCommand("1.0.0", "abc123", "2025-01-01")

		flag := cmd.PersistentFlags().Lookup("config")
		if flag == nil {
			t.Error("expected config flag to exist")
		}
	})

	t.Run("has service subcommand", func(t *testing.T) {
		cmd := cli.NewRootCommand("1.0.0", "abc123", "2025-01-01")

		serviceCmd, _, err := cmd.Find([]string{"service"})
		if err != nil {
			t.Fatalf("failed to find service command: %v", err)
		}

		if serviceCmd.Use != "service" {
			t.Errorf("expected service command, got '%s'", serviceCmd.Use)
		}
	})

	t.Run("has issuer subcommand", func(t *testing.T) {
		cmd := cli.NewRootCommand("1.0.0", "abc123", "2025-01-01")

		issuerCmd, _, err := cmd.Find([]string{"issuer"})
		if err != nil {
			t.Fatalf("failed to find issuer command: %v", err)
		}

		if issuerCmd.Use != "issuer" {
			t.Errorf("expected issuer command, got '%s'", issuerCmd.Use)
		}
	})

	t.Run("has statement subcommand", func(t *testing.T) {
		cmd := cli.NewRootCommand("1.0.0", "abc123", "2025-01-01")

		statementCmd, _, err := cmd.Find([]string{"statement"})
		if err != nil {
			t.Fatalf("failed to find statement command: %v", err)
		}

		if statementCmd.Use != "statement" {
			t.Errorf("expected statement command, got '%s'", statementCmd.Use)
		}
	})

	t.Run("has receipt subcommand", func(t *testing.T) {
		cmd := cli.NewRootCommand("1.0.0", "abc123", "2025-01-01")

		receiptCmd, _, err := cmd.Find([]string{"receipt"})
		if err != nil {
			t.Fatalf("failed to find receipt command: %v", err)
		}

		if receiptCmd.Use != "receipt" {
			t.Errorf("expected receipt command, got '%s'", receiptCmd.Use)
		}
	})
}

// TestStatementSubcommands tests statement subcommands
func TestStatementSubcommands(t *testing.T) {
	t.Run("has sign subcommand", func(t *testing.T) {
		cmd := cli.NewRootCommand("1.0.0", "abc123", "2025-01-01")

		signCmd, _, err := cmd.Find([]string{"statement", "sign"})
		if err != nil {
			t.Fatalf("failed to find statement sign command: %v", err)
		}

		if signCmd.Use != "sign" {
			t.Errorf("expected sign command, got '%s'", signCmd.Use)
		}
	})

	t.Run("has verify subcommand", func(t *testing.T) {
		cmd := cli.NewRootCommand("1.0.0", "abc123", "2025-01-01")

		verifyCmd, _, err := cmd.Find([]string{"statement", "verify"})
		if err != nil {
			t.Fatalf("failed to find statement verify command: %v", err)
		}

		if verifyCmd.Use != "verify" {
			t.Errorf("expected verify command, got '%s'", verifyCmd.Use)
		}
	})

	t.Run("has hash subcommand", func(t *testing.T) {
		cmd := cli.NewRootCommand("1.0.0", "abc123", "2025-01-01")

		hashCmd, _, err := cmd.Find([]string{"statement", "hash"})
		if err != nil {
			t.Fatalf("failed to find statement hash command: %v", err)
		}

		if hashCmd.Use != "hash" {
			t.Errorf("expected hash command, got '%s'", hashCmd.Use)
		}
	})
}

// TestReceiptSubcommands tests receipt subcommands
func TestReceiptSubcommands(t *testing.T) {
	t.Run("has verify subcommand", func(t *testing.T) {
		cmd := cli.NewRootCommand("1.0.0", "abc123", "2025-01-01")

		verifyCmd, _, err := cmd.Find([]string{"receipt", "verify"})
		if err != nil {
			t.Fatalf("failed to find receipt verify command: %v", err)
		}

		if verifyCmd.Use != "verify" {
			t.Errorf("expected verify command, got '%s'", verifyCmd.Use)
		}
	})

	t.Run("has info subcommand", func(t *testing.T) {
		cmd := cli.NewRootCommand("1.0.0", "abc123", "2025-01-01")

		infoCmd, _, err := cmd.Find([]string{"receipt", "info"})
		if err != nil {
			t.Fatalf("failed to find receipt info command: %v", err)
		}

		if infoCmd.Use != "info" {
			t.Errorf("expected info command, got '%s'", infoCmd.Use)
		}
	})
}
