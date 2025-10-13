package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/config"
)

// Global flags
var (
	cfgFile string
	verbose bool
	cfg     *config.Config
)

// NewRootCommand creates the root cobra command
func NewRootCommand(version, commit, date string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "scitt",
		Short: "SCITT Transparency Service CLI",
		Long: `SCITT (Supply Chain Integrity, Transparency, and Trust) CLI tool.

This command-line interface provides tools for interacting with
SCITT transparency services, including:
  - Initializing a new transparency service
  - Starting an HTTP server
  - Managing statements and receipts
  - Verifying proofs and checkpoints`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		SilenceUsage: true,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./scitt.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Initialize configuration
	cobra.OnInitialize(initConfig)

	// Add subcommands
	rootCmd.AddCommand(NewInitCommand())
	rootCmd.AddCommand(NewServeCommand())
	rootCmd.AddCommand(NewIssuerCommand())
	rootCmd.AddCommand(NewStatementCommand())
	rootCmd.AddCommand(NewReceiptCommand())

	return rootCmd
}

// initConfig loads configuration from file
func initConfig() {
	if cfgFile == "" {
		// Try default locations
		if _, err := os.Stat("scitt.yaml"); err == nil {
			cfgFile = "scitt.yaml"
		} else if _, err := os.Stat("scitt.yml"); err == nil {
			cfgFile = "scitt.yml"
		}
	}

	if cfgFile != "" {
		var err error
		cfg, err = config.LoadConfig(cfgFile)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
			}
		}
	}
}

// GetConfig returns the loaded configuration
func GetConfig() *config.Config {
	return cfg
}
