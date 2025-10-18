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

// redactMongoURI redacts sensitive information (username and password) from MongoDB connection URIs
// Example: mongodb+srv://user:pass@host/db -> mongodb+srv://***:***@host/db
func redactMongoURI(uri string) string {
	// Parse the URI
	parsedURL, err := url.Parse(uri)
	if err != nil {
		// If we can't parse it, redact the whole thing to be safe
		return "[REDACTED]"
	}

	// Check if there's user info
	if parsedURL.User != nil {
		// Replace username and password with asterisks
		parsedURL.User = url.UserPassword("***", "***")
	}

	return parsedURL.String()
}

// NewServiceCommand creates the service command
func NewServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage SCITT transparency service",
		Long: `Manage SCITT transparency service configuration and lifecycle.

Subcommands:
  create - Create a new service definition
  start  - Start the transparency service`,
	}

	cmd.AddCommand(NewServiceCreateCommand())
	cmd.AddCommand(NewServiceStartCommand())
	cmd.AddCommand(NewServiceResetCommand())

	return cmd
}

type serviceDefinitionCreateOptions struct {
	receiptIssuer          string
	receiptSigningKey      string
	receiptVerificationKey string
	tileStorage            string
	metadataStorage        string
	databaseType           string
	mongodbURI             string
	mongodbDatabase        string
	definition             string
}

// NewServiceCreateCommand creates the service create command
func NewServiceCreateCommand() *cobra.Command {
	opts := &serviceDefinitionCreateOptions{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new service definition",
		Long: `Create a new SCITT transparency service definition.

This command initializes a complete transparency service configuration including:
  - YAML configuration file with all service parameters
  - Tile storage directory for Merkle tree data
  - Database for statement metadata (SQLite or MongoDB)

The generated configuration can be used with 'scitt service start' to start the service.

Examples:
  # Create service with SQLite (default)
  scitt service create \
    --receipt-issuer https://transparency.example \
    --receipt-signing-key ./demo/priv.cbor \
    --receipt-verification-key ./demo/pub.cbor \
    --tile-storage ./demo/tiles \
    --metadata-storage ./demo/scitt.db \
    --definition ./demo/scitt.yaml

  # Create service with MongoDB
  scitt service create \
    --receipt-issuer https://transparency.example \
    --receipt-signing-key ./demo/priv.cbor \
    --receipt-verification-key ./demo/pub.cbor \
    --tile-storage ./demo/tiles \
    --database-type mongodb \
    --mongodb-uri "mongodb://localhost:27017" \
    --mongodb-database scitt \
    --definition ./demo/scitt.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServiceDefinitionCreate(opts)
		},
	}

	cmd.Flags().StringVar(&opts.receiptIssuer, "receipt-issuer", "", "receipt issuer URL (e.g., https://transparency.example)")
	cmd.Flags().StringVar(&opts.receiptSigningKey, "receipt-signing-key", "", "path to receipt signing key (CBOR format)")
	cmd.Flags().StringVar(&opts.receiptVerificationKey, "receipt-verification-key", "", "path to receipt verification key (CBOR format)")
	cmd.Flags().StringVar(&opts.tileStorage, "tile-storage", "", "path to tile storage directory")
	cmd.Flags().StringVar(&opts.metadataStorage, "metadata-storage", "", "path to metadata database file (for SQLite)")
	cmd.Flags().StringVar(&opts.databaseType, "database-type", "sqlite", "database type: sqlite or mongodb")
	cmd.Flags().StringVar(&opts.mongodbURI, "mongodb-uri", "", "MongoDB connection URI (required if database-type is mongodb)")
	cmd.Flags().StringVar(&opts.mongodbDatabase, "mongodb-database", "", "MongoDB database name (required if database-type is mongodb)")
	cmd.Flags().StringVar(&opts.definition, "definition", "", "path to output definition file (YAML)")

	cmd.MarkFlagRequired("receipt-issuer")
	cmd.MarkFlagRequired("receipt-signing-key")
	cmd.MarkFlagRequired("receipt-verification-key")
	cmd.MarkFlagRequired("tile-storage")
	cmd.MarkFlagRequired("definition")

	return cmd
}

func runServiceDefinitionCreate(opts *serviceDefinitionCreateOptions) error {
	if verbose {
		fmt.Println("Creating service definition...")
	}

	// Validate database type
	if opts.databaseType != "sqlite" && opts.databaseType != "mongodb" {
		return fmt.Errorf("database-type must be 'sqlite' or 'mongodb', got: %s", opts.databaseType)
	}

	// Validate database-specific options
	if opts.databaseType == "sqlite" {
		if opts.metadataStorage == "" {
			return fmt.Errorf("--metadata-storage is required for SQLite database")
		}
	} else if opts.databaseType == "mongodb" {
		if opts.mongodbURI == "" {
			return fmt.Errorf("--mongodb-uri is required when using MongoDB")
		}
		if opts.mongodbDatabase == "" {
			return fmt.Errorf("--mongodb-database is required when using MongoDB")
		}
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

	// Initialize database based on type
	if opts.databaseType == "sqlite" {
		// Create metadata storage directory (for database file)
		metadataDir := filepath.Dir(opts.metadataStorage)
		if metadataDir != "." && metadataDir != "" {
			if err := os.MkdirAll(metadataDir, 0755); err != nil {
				return fmt.Errorf("failed to create metadata storage directory: %w", err)
			}
		}

		// Initialize SQLite database
		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      opts.metadataStorage,
			EnableWAL: true,
		})
		if err != nil {
			return fmt.Errorf("failed to initialize SQLite database: %w", err)
		}
		defer database.CloseDatabase(db)

		if verbose {
			fmt.Printf("  SQLite database initialized: %s\n", opts.metadataStorage)
		}
	} else {
		// For MongoDB, we don't need to initialize anything here
		// The database will be created when the service starts
		if verbose {
			redactedURI := redactMongoURI(opts.mongodbURI)
			fmt.Printf("  MongoDB connection configured: %s/%s\n", redactedURI, opts.mongodbDatabase)
		}
	}

	// Create definition directory
	definitionDir := filepath.Dir(opts.definition)
	if definitionDir != "." && definitionDir != "" {
		if err := os.MkdirAll(definitionDir, 0755); err != nil {
			return fmt.Errorf("failed to create definition directory: %w", err)
		}
	}

	// Generate cryptographically secure API key for .env file
	generatedAPIKey, err := config.GenerateAPIKey()
	if err != nil {
		return fmt.Errorf("failed to generate API key: %w", err)
	}

	// Create database configuration based on type
	var dbConfig config.DatabaseConfig
	if opts.databaseType == "sqlite" {
		dbConfig = config.DatabaseConfig{
			Type:      "sqlite",
			Path:      opts.metadataStorage,
			EnableWAL: true,
		}
	} else {
		// Use environment variable references for MongoDB credentials
		// This keeps the YAML config safe to share without exposing secrets
		dbConfig = config.DatabaseConfig{
			Type: "mongodb",
			MongoDB: &config.MongoDBConfig{
				URI:      "${SCITT_MONGODB_URI}",
				Database: "${SCITT_MONGODB_DATABASE}",
			},
		}
	}

	// Create configuration
	cfg := &config.Config{
		Issuer:   opts.receiptIssuer,
		Database: dbConfig,
		Storage: config.StorageConfig{
			Type: "local",
			Path: opts.tileStorage,
		},
		Keys: config.KeysConfig{
			Private: opts.receiptSigningKey,
			Public:  opts.receiptVerificationKey,
		},
		Server: config.ServerConfig{
			Host:   "127.0.0.1",
			Port:   56177,
			APIKey: "${SCITT_API_KEY}", // Reference environment variable
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
	fmt.Printf("  Issuer:       %s\n", opts.receiptIssuer)
	fmt.Printf("  Database:     %s", opts.databaseType)
	if opts.databaseType == "sqlite" {
		fmt.Printf(" (%s)\n", opts.metadataStorage)
	} else {
		fmt.Printf(" (configured via environment variables)\n")
	}
	fmt.Printf("  Tiles:        %s\n", opts.tileStorage)
	fmt.Printf("  Definition:   %s\n", opts.definition)
	fmt.Printf("\n✓ Generated API Key (add to .env file):\n")
	fmt.Printf("  SCITT_API_KEY=%s\n", generatedAPIKey)
	if opts.databaseType == "mongodb" {
		fmt.Printf("\n⚠ MongoDB Configuration Required:\n")
		fmt.Printf("  Add these to your .env file:\n")
		fmt.Printf("    SCITT_MONGODB_URI=%s\n", opts.mongodbURI)
		fmt.Printf("    SCITT_MONGODB_DATABASE=%s\n", opts.mongodbDatabase)
	}
	fmt.Printf("\nStart the service with:\n")
	fmt.Printf("  ./scitt service start --definition %s\n", opts.definition)

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
		fmt.Printf("  Issuer:   %s\n", cfg.Issuer)
		if cfg.Database.Type == "sqlite" {
			fmt.Printf("  Database: %s (%s)\n", cfg.Database.Type, cfg.Database.Path)
		} else {
			// Redact sensitive information from MongoDB URI
			redactedURI := redactMongoURI(cfg.Database.MongoDB.URI)
			fmt.Printf("  Database: %s (%s/%s)\n", cfg.Database.Type, redactedURI, cfg.Database.MongoDB.Database)
		}
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

type serviceResetOptions struct {
	definition string
	force      bool
}

// NewServiceResetCommand creates the service reset command
func NewServiceResetCommand() *cobra.Command {
	opts := &serviceResetOptions{}

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset the transparency service (delete all data)",
		Long: `Reset the SCITT transparency service by deleting all data.

⚠️  WARNING: This operation is destructive and cannot be undone!

This command will:
  - Delete all registered statements from the database
  - Reset the tree size to 0
  - Delete all tile files from storage

This is useful for development and testing when you want to start with a clean slate.

Example:
  # Reset with confirmation prompt
  scitt service reset --definition ./demo/scitt.yaml

  # Reset without confirmation (use with caution!)
  scitt service reset --definition ./demo/scitt.yaml --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServiceReset(opts)
		},
	}

	cmd.Flags().StringVar(&opts.definition, "definition", "", "path to service definition file (YAML)")
	cmd.Flags().BoolVar(&opts.force, "force", false, "skip confirmation prompt")

	cmd.MarkFlagRequired("definition")

	return cmd
}

func runServiceReset(opts *serviceResetOptions) error {
	// Load configuration from definition file
	cfg, err := config.LoadConfig(opts.definition)
	if err != nil {
		return fmt.Errorf("failed to load service definition: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Confirmation prompt unless --force is used
	if !opts.force {
		fmt.Println("⚠️  WARNING: This will delete all data from the transparency service!")
		fmt.Printf("  Database: %s", cfg.Database.Type)
		if cfg.Database.Type == "sqlite" {
			fmt.Printf(" (%s)\n", cfg.Database.Path)
		} else {
			redactedURI := redactMongoURI(cfg.Database.MongoDB.URI)
			fmt.Printf(" (%s/%s)\n", redactedURI, cfg.Database.MongoDB.Database)
		}
		fmt.Printf("  Storage:  %s", cfg.Storage.Type)
		if cfg.Storage.Type == "local" {
			fmt.Printf(" (%s)\n", cfg.Storage.Path)
		} else if cfg.Storage.Type == "azure" {
			fmt.Printf(" (Azure container: %s)\n", cfg.Storage.Azure.Container)
		} else {
			fmt.Println()
		}
		fmt.Print("\nAre you sure you want to continue? (yes/no): ")

		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			fmt.Println("Reset cancelled.")
			return nil
		}
	}

	if verbose {
		fmt.Println("Resetting SCITT transparency service...")
	}

	// Create service (this will initialize repository and storage)
	srv, err := server.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	defer srv.Close()

	// Call the Reset method on the service
	if err := srv.Service.Reset(); err != nil {
		return fmt.Errorf("failed to reset service: %w", err)
	}

	fmt.Println("✓ Service reset successfully")
	fmt.Println("  All statements deleted")
	fmt.Println("  Tree size reset to 0")
	fmt.Println("  All tiles removed from storage")

	return nil
}
