package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/cli"
)

// Version information (set by build flags)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Load .env file from current directory if it exists
	// Silently ignore if .env doesn't exist
	_ = godotenv.Load()

	rootCmd := cli.NewRootCommand(version, commit, date)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
