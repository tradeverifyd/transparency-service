package main

import (
	"fmt"
	"os"

	"github.com/fxamacker/cbor/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <cbor-file>\n", os.Args[0])
		os.Exit(1)
	}

	filename := os.Args[1]
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Create diagnostic mode
	diagMode, err := cbor.DiagOptions{}.DiagMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating diag mode: %v\n", err)
		os.Exit(1)
	}

	// Convert to diagnostic notation
	diagText, err := diagMode.Diagnose(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting to diagnostic notation: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(diagText)
}
