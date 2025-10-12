package cli

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/server"
)

type serveOptions struct {
	host string
	port int
}

// NewServeCommand creates the serve command
func NewServeCommand() *cobra.Command {
	opts := &serveOptions{}

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the SCITT HTTP server",
		Long: `Start the SCITT transparency service HTTP server.

This command starts an HTTP server that implements the SCRAPI
(Supply Chain Repository API) specification. The server provides:
  - POST /entries - Register statements
  - GET /entries/{id} - Retrieve receipts
  - GET /checkpoint - Get current signed tree head
  - GET /.well-known/transparency-configuration - Service configuration

Example:
  scitt serve --config scitt.yaml
  scitt serve --host 0.0.0.0 --port 8080`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(opts)
		},
	}

	cmd.Flags().StringVar(&opts.host, "host", "", "host to bind to (overrides config)")
	cmd.Flags().IntVarP(&opts.port, "port", "p", 0, "port to listen on (overrides config)")

	return cmd
}

func runServe(opts *serveOptions) error {
	// Get configuration
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration loaded - use --config flag or create scitt.yaml")
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
		fmt.Printf("  Origin: %s\n", cfg.Origin)
		fmt.Printf("  Database: %s\n", cfg.Database.Path)
		fmt.Printf("  Storage: %s (%s)\n", cfg.Storage.Type, cfg.Storage.Path)
		fmt.Printf("  Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
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
