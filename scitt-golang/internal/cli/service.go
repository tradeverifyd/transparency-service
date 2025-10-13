package cli

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/config"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/server"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/database"
)

// NewServiceCommand creates the service command
func NewServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage SCITT transparency service",
		Long: `Manage SCITT transparency service configuration and lifecycle.

Subcommands:
  definition create - Create a new service definition
  start            - Start the transparency service`,
	}

	cmd.AddCommand(NewServiceDefinitionCommand())
	cmd.AddCommand(NewServiceStartCommand())

	return cmd
}

// NewServiceDefinitionCommand creates the service definition command
func NewServiceDefinitionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "definition",
		Short: "Manage service definitions",
		Long:  `Manage SCITT transparency service definitions.`,
	}

	cmd.AddCommand(NewServiceDefinitionCreateCommand())

	return cmd
}

type serviceDefinitionCreateOptions struct {
	receiptIssuer          string
	receiptSigningKey      string
	receiptVerificationKey string
	tileStorage            string
	metadataStorage        string
	definition             string
}

// NewServiceDefinitionCreateCommand creates the service definition create command
func NewServiceDefinitionCreateCommand() *cobra.Command {
	opts := &serviceDefinitionCreateOptions{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new service definition",
		Long: `Create a new SCITT transparency service definition.

This command initializes a complete transparency service configuration including:
  - YAML configuration file with all service parameters
  - Tile storage directory for Merkle tree data
  - SQLite database for statement metadata

The generated configuration can be used with 'scitt serve' to start the service.

Example:
  scitt service definition create \
    --receipt-issuer https://transparency.example \
    --receipt-signing-key ./demo/priv.cbor \
    --receipt-verification-key ./demo/pub.cbor \
    --tile-storage ./demo/tiles \
    --metadata-storage ./demo/scitt.db \
    --definition ./demo/scitt.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServiceDefinitionCreate(opts)
		},
	}

	cmd.Flags().StringVar(&opts.receiptIssuer, "receipt-issuer", "", "receipt issuer URL (e.g., https://transparency.example)")
	cmd.Flags().StringVar(&opts.receiptSigningKey, "receipt-signing-key", "", "path to receipt signing key (CBOR format)")
	cmd.Flags().StringVar(&opts.receiptVerificationKey, "receipt-verification-key", "", "path to receipt verification key (CBOR format)")
	cmd.Flags().StringVar(&opts.tileStorage, "tile-storage", "", "path to tile storage directory")
	cmd.Flags().StringVar(&opts.metadataStorage, "metadata-storage", "", "path to metadata database file (SQLite)")
	cmd.Flags().StringVar(&opts.definition, "definition", "", "path to output definition file (YAML)")

	cmd.MarkFlagRequired("receipt-issuer")
	cmd.MarkFlagRequired("receipt-signing-key")
	cmd.MarkFlagRequired("receipt-verification-key")
	cmd.MarkFlagRequired("tile-storage")
	cmd.MarkFlagRequired("metadata-storage")
	cmd.MarkFlagRequired("definition")

	return cmd
}

func runServiceDefinitionCreate(opts *serviceDefinitionCreateOptions) error {
	if verbose {
		fmt.Println("Creating service definition...")
	}

	// Validate receipt issuer URL
	parsedURL, err := url.Parse(opts.receiptIssuer)
	if err != nil {
		return fmt.Errorf("invalid receipt-issuer URL: %w", err)
	}
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return fmt.Errorf("receipt-issuer must use http or https scheme")
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("receipt-issuer must have a valid host")
	}

	// Validate signing key exists
	if _, err := os.Stat(opts.receiptSigningKey); os.IsNotExist(err) {
		return fmt.Errorf("receipt-signing-key not found: %s", opts.receiptSigningKey)
	}

	// Validate verification key exists
	if _, err := os.Stat(opts.receiptVerificationKey); os.IsNotExist(err) {
		return fmt.Errorf("receipt-verification-key not found: %s", opts.receiptVerificationKey)
	}

	// Create tile storage directory
	if err := os.MkdirAll(opts.tileStorage, 0755); err != nil {
		return fmt.Errorf("failed to create tile storage directory: %w", err)
	}

	// Create metadata storage directory (for database file)
	metadataDir := filepath.Dir(opts.metadataStorage)
	if metadataDir != "." && metadataDir != "" {
		if err := os.MkdirAll(metadataDir, 0755); err != nil {
			return fmt.Errorf("failed to create metadata storage directory: %w", err)
		}
	}

	// Initialize database
	db, err := database.OpenDatabase(database.DatabaseOptions{
		Path:      opts.metadataStorage,
		EnableWAL: true,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.CloseDatabase(db)

	if verbose {
		fmt.Printf("  Database initialized: %s\n", opts.metadataStorage)
	}

	// Create definition directory
	definitionDir := filepath.Dir(opts.definition)
	if definitionDir != "." && definitionDir != "" {
		if err := os.MkdirAll(definitionDir, 0755); err != nil {
			return fmt.Errorf("failed to create definition directory: %w", err)
		}
	}

	// Create configuration
	cfg := &config.Config{
		Origin: opts.receiptIssuer,
		Database: config.DatabaseConfig{
			Path:      opts.metadataStorage,
			EnableWAL: true,
		},
		Storage: config.StorageConfig{
			Type: "local",
			Path: opts.tileStorage,
		},
		Keys: config.KeysConfig{
			Private: opts.receiptSigningKey,
			Public:  opts.receiptVerificationKey,
		},
		Server: config.ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
			CORS: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
			},
		},
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Save configuration
	if err := config.SaveConfig(cfg, opts.definition); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("✓ Service definition created successfully\n")
	fmt.Printf("  Definition:   %s\n", opts.definition)
	fmt.Printf("  Origin:       %s\n", opts.receiptIssuer)
	fmt.Printf("  Tile storage: %s\n", opts.tileStorage)
	fmt.Printf("  Database:     %s\n", opts.metadataStorage)
	fmt.Printf("  Signing key:  %s\n", opts.receiptSigningKey)
	fmt.Printf("  Verify key:   %s\n", opts.receiptVerificationKey)
	fmt.Printf("\nStart the service with:\n")
	fmt.Printf("  scitt service start --definition %s\n", opts.definition)

	return nil
}

type serviceStartOptions struct {
	definition string
	host       string
	port       int
}

// NewServiceStartCommand creates the service start command
func NewServiceStartCommand() *cobra.Command {
	opts := &serviceStartOptions{}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the SCITT transparency service",
		Long: `Start the SCITT transparency service HTTP server.

This command starts an HTTP server that implements the SCRAPI
(SCITT Reference APIs) specification. The server provides:
  - POST /entries - Register statements
  - GET /entries/{id} - Retrieve receipts
  - GET /checkpoint - Get current signed tree head
  - GET /.well-known/scitt-configuration - Service configuration
  - GET /.well-known/scitt-keys - Service verification keys

Example:
  scitt service start --definition ./demo/scitt.yaml
  scitt service start --definition ./demo/scitt.yaml --host 127.0.0.1 --port 9000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServiceStart(opts)
		},
	}

	cmd.Flags().StringVar(&opts.definition, "definition", "", "path to service definition file (YAML)")
	cmd.Flags().StringVar(&opts.host, "host", "", "host to bind to (overrides definition)")
	cmd.Flags().IntVarP(&opts.port, "port", "p", 0, "port to listen on (overrides definition)")

	cmd.MarkFlagRequired("definition")

	return cmd
}

func runServiceStart(opts *serviceStartOptions) error {
	// Load configuration from definition file
	cfg, err := config.LoadConfig(opts.definition)
	if err != nil {
		return fmt.Errorf("failed to load service definition: %w", err)
	}

	// Override config with command line flags
	if opts.host != "" {
		cfg.Server.Host = opts.host
	}
	if opts.port != 0 {
		cfg.Server.Port = opts.port
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if verbose {
		fmt.Println("Starting SCITT transparency service...")
		fmt.Printf("  Origin:   %s\n", cfg.Origin)
		fmt.Printf("  Database: %s\n", cfg.Database.Path)
		fmt.Printf("  Storage:  %s (%s)\n", cfg.Storage.Type, cfg.Storage.Path)
		fmt.Printf("  Server:   %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	}

	// Create server
	srv, err := server.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	defer srv.Close()

	// Start server (blocks until error or shutdown)
	log.Fatal(srv.Start())
	return nil
}
