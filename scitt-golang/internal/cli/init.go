package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/database"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
)

type initOptions struct {
	dir        string
	origin     string
	dbPath     string
	storagePath string
	force      bool
}

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	opts := &initOptions{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new SCITT transparency service",
		Long: `Initialize a new SCITT transparency service.

This command creates:
  - A new ES256 key pair for signing checkpoints
  - An SQLite database for statement metadata
  - A storage directory for Merkle tree tiles
  - A configuration file (scitt.yaml)

Example:
  scitt init --origin https://transparency.example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(opts)
		},
	}

	cmd.Flags().StringVar(&opts.dir, "dir", ".", "directory to initialize service in")
	cmd.Flags().StringVar(&opts.origin, "origin", "", "origin URL for the transparency service (required)")
	cmd.Flags().StringVar(&opts.dbPath, "db", "scitt.db", "path to SQLite database file")
	cmd.Flags().StringVar(&opts.storagePath, "storage", "./storage", "path to storage directory")
	cmd.Flags().BoolVar(&opts.force, "force", false, "overwrite existing files")

	cmd.MarkFlagRequired("origin")

	return cmd
}

func runInit(opts *initOptions) error {
	// Validate origin URL
	if opts.origin == "" {
		return fmt.Errorf("origin URL is required")
	}

	// Create directory structure
	if err := os.MkdirAll(opts.dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if service already initialized
	dbPath := filepath.Join(opts.dir, opts.dbPath)
	if _, err := os.Stat(dbPath); err == nil && !opts.force {
		return fmt.Errorf("service already initialized (use --force to overwrite)")
	}

	// Generate ES256 key pair
	if verbose {
		fmt.Println("Generating ES256 key pair...")
	}
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Export private key to PEM
	privateKeyPEM, err := cose.ExportPrivateKeyToPEM(keyPair.Private)
	if err != nil {
		return fmt.Errorf("failed to export private key: %w", err)
	}

	// Save private key
	keyPath := filepath.Join(opts.dir, "service-key.pem")
	if err := os.WriteFile(keyPath, []byte(privateKeyPEM), 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Export public key to JWK
	publicKeyJWK, err := cose.ExportPublicKeyToJWK(keyPair.Public)
	if err != nil {
		return fmt.Errorf("failed to export public key: %w", err)
	}

	// Marshal JWK to JSON
	publicKeyJSON, err := cose.MarshalJWK(publicKeyJWK)
	if err != nil {
		return fmt.Errorf("failed to marshal public key JWK: %w", err)
	}

	// Save public key
	pubKeyPath := filepath.Join(opts.dir, "service-key.jwk")
	if err := os.WriteFile(pubKeyPath, publicKeyJSON, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	// Initialize database
	if verbose {
		fmt.Println("Initializing database...")
	}
	db, err := database.OpenDatabase(database.DatabaseOptions{
		Path:      dbPath,
		EnableWAL: true,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	database.CloseDatabase(db)

	// Initialize storage
	if verbose {
		fmt.Println("Initializing storage...")
	}
	storagePath := filepath.Join(opts.dir, opts.storagePath)
	_, err = storage.NewLocalStorage(storagePath)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Create configuration file
	if verbose {
		fmt.Println("Creating configuration file...")
	}
	config := fmt.Sprintf(`# SCITT Transparency Service Configuration

# Service origin URL
origin: %s

# Database configuration
database:
  path: %s
  enable_wal: true

# Storage configuration
storage:
  type: local
  path: %s

# Service keys
keys:
  private: service-key.pem
  public: service-key.jwk

# HTTP server configuration
server:
  host: 0.0.0.0
  port: 8080
  cors:
    enabled: true
    allowed_origins:
      - "*"
`, opts.origin, opts.dbPath, opts.storagePath)

	configPath := filepath.Join(opts.dir, "scitt.yaml")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Success message
	fmt.Println("✓ SCITT transparency service initialized")
	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  Origin:      %s\n", opts.origin)
	fmt.Printf("  Database:    %s\n", dbPath)
	fmt.Printf("  Storage:     %s\n", storagePath)
	fmt.Printf("  Private Key: %s\n", keyPath)
	fmt.Printf("  Public Key:  %s\n", pubKeyPath)
	fmt.Printf("  Config:      %s\n", configPath)
	fmt.Printf("\nTo start the service, run:\n")
	fmt.Printf("  scitt serve --config %s\n", configPath)

	return nil
}
