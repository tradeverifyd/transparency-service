package lib

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CLIResult represents the result of executing a CLI command
type CLIResult struct {
	Command    []string
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int
	Error      error
}

// RunGoCLI executes a Go CLI command with the given arguments and environment
// Returns stdout, stderr, exit code, and any error
func RunGoCLI(args []string, workDir string, env map[string]string, timeoutSec int) *CLIResult {
	binaryPath := GetGoBinaryPath()
	return runCLI(binaryPath, args, workDir, env, timeoutSec)
}

// RunTsCLI executes a TypeScript CLI command with the given arguments and environment
// Returns stdout, stderr, exit code, and any error
func RunTsCLI(args []string, workDir string, env map[string]string, timeoutSec int) *CLIResult {
	// TypeScript CLI is invoked via bun
	cliCommand := GetTsCLICommand()

	// Parse the CLI command (e.g., "bun run path/to/cli.ts")
	parts := strings.Fields(cliCommand)
	if len(parts) == 0 {
		return &CLIResult{
			Error: fmt.Errorf("invalid TypeScript CLI command: %s", cliCommand),
		}
	}

	// Combine CLI command with args
	fullArgs := append(parts[1:], args...)
	return runCLI(parts[0], fullArgs, workDir, env, timeoutSec)
}

// runCLI is the internal function that executes a CLI command
func runCLI(command string, args []string, workDir string, env map[string]string, timeoutSec int) *CLIResult {
	startTime := time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Prepare command
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = workDir

	// Set environment variables
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), envMapToSlice(env)...)
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()
	durationMs := int(time.Since(startTime).Milliseconds())

	// Determine exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Non-exit error (e.g., command not found, timeout)
			exitCode = -1
		}
	}

	return &CLIResult{
		Command:    append([]string{command}, args...),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   exitCode,
		DurationMs: durationMs,
		Error:      err,
	}
}

// envMapToSlice converts a map of environment variables to a slice of "KEY=VALUE" strings
func envMapToSlice(env map[string]string) []string {
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

// Success returns true if the CLI command succeeded (exit code 0)
func (r *CLIResult) Success() bool {
	return r.ExitCode == 0
}

// Failed returns true if the CLI command failed (non-zero exit code)
func (r *CLIResult) Failed() bool {
	return r.ExitCode != 0
}

// ToImplementationResult converts a CLIResult to an ImplementationResult
func (r *CLIResult) ToImplementationResult(implementation string) *ImplementationResult {
	return &ImplementationResult{
		Implementation: implementation,
		Command:        r.Command,
		ExitCode:       r.ExitCode,
		Stdout:         r.Stdout,
		Stderr:         r.Stderr,
		OutputFormat:   "text", // Default, can be overridden
		DurationMs:     r.DurationMs,
		Success:        r.Success(),
	}
}
