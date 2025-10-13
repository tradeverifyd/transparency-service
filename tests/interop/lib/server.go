package lib

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// ServerProcess represents a running server process
type ServerProcess struct {
	Cmd        *exec.Cmd
	Port       int
	WorkDir    string
	ConfigPath string
	LogFile    *os.File
	cancel     context.CancelFunc
}

// StartGoServer starts the Go SCITT server as a background process
// It initializes the service, starts the server, and waits for it to be ready
func StartGoServer(t *testing.T, workDir string, port int) (*ServerProcess, error) {
	t.Helper()

	// First, initialize the service
	origin := fmt.Sprintf("http://localhost:%d", port)
	initArgs := []string{"init", "--dir", workDir, "--origin", origin}
	initResult := RunGoCLI(initArgs, workDir, nil, 30)
	if !initResult.Success() {
		return nil, fmt.Errorf("failed to initialize Go service: %s", initResult.Stderr)
	}

	// Open log file for server output
	logPath := fmt.Sprintf("%s/server.log", workDir)
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Prepare server command
	ctx, cancel := context.WithCancel(context.Background())
	binaryPath := GetGoBinaryPath()
	configPath := fmt.Sprintf("%s/scitt.yaml", workDir)

	cmd := exec.CommandContext(ctx, binaryPath, "serve", "--config", configPath, "--port", fmt.Sprintf("%d", port))
	cmd.Dir = workDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start server
	if err := cmd.Start(); err != nil {
		cancel()
		logFile.Close()
		return nil, fmt.Errorf("failed to start Go server: %w", err)
	}

	server := &ServerProcess{
		Cmd:        cmd,
		Port:       port,
		WorkDir:    workDir,
		ConfigPath: configPath,
		LogFile:    logFile,
		cancel:     cancel,
	}

	// Wait for server to be ready
	if err := waitForServerReady(port, 30*time.Second); err != nil {
		server.Stop()
		return nil, fmt.Errorf("Go server did not become ready: %w", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		server.Stop()
	})

	return server, nil
}

// StartTsServer starts the TypeScript transparency server as a background process
// It initializes the service, starts the server, and waits for it to be ready
func StartTsServer(t *testing.T, workDir string, port int) (*ServerProcess, error) {
	t.Helper()

	// First, initialize the service
	dbPath := fmt.Sprintf("%s/transparency.db", workDir)
	initArgs := []string{"transparency", "init", "--database", dbPath, "--port", fmt.Sprintf("%d", port)}
	initResult := RunTsCLI(initArgs, workDir, nil, 30)
	if !initResult.Success() {
		return nil, fmt.Errorf("failed to initialize TypeScript service: %s", initResult.Stderr)
	}

	// Open log file for server output
	logPath := fmt.Sprintf("%s/server.log", workDir)
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Prepare server command
	ctx, cancel := context.WithCancel(context.Background())
	cliCommand := GetTsCLICommand()
	parts := strings.Fields(cliCommand)
	if len(parts) == 0 {
		cancel()
		logFile.Close()
		return nil, fmt.Errorf("invalid TypeScript CLI command")
	}

	// Combine CLI path with serve command
	serveArgs := append(parts[1:], "transparency", "serve", "--port", fmt.Sprintf("%d", port), "--database", dbPath)
	cmd := exec.CommandContext(ctx, parts[0], serveArgs...)
	cmd.Dir = workDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start server
	if err := cmd.Start(); err != nil {
		cancel()
		logFile.Close()
		return nil, fmt.Errorf("failed to start TypeScript server: %w", err)
	}

	server := &ServerProcess{
		Cmd:        cmd,
		Port:       port,
		WorkDir:    workDir,
		ConfigPath: dbPath,
		LogFile:    logFile,
		cancel:     cancel,
	}

	// Wait for server to be ready
	if err := waitForServerReady(port, 30*time.Second); err != nil {
		server.Stop()
		return nil, fmt.Errorf("TypeScript server did not become ready: %w", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		server.Stop()
	})

	return server, nil
}

// Stop gracefully stops the server process
func (s *ServerProcess) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.Cmd != nil && s.Cmd.Process != nil {
		// Give it a moment to shut down gracefully
		time.Sleep(100 * time.Millisecond)
		s.Cmd.Process.Kill()
		s.Cmd.Wait()
	}
	if s.LogFile != nil {
		s.LogFile.Close()
	}
}

// GetBaseURL returns the base URL for the server
func (s *ServerProcess) GetBaseURL() string {
	return fmt.Sprintf("http://localhost:%d", s.Port)
}

// GetLogContents returns the contents of the server log file
func (s *ServerProcess) GetLogContents() (string, error) {
	if s.LogFile == nil {
		return "", fmt.Errorf("no log file available")
	}

	// Sync to ensure all data is written
	s.LogFile.Sync()

	// Read the log file
	logPath := s.LogFile.Name()
	data, err := os.ReadFile(logPath)
	if err != nil {
		return "", fmt.Errorf("failed to read log file: %w", err)
	}

	return string(data), nil
}

// waitForServerReady polls the server's health endpoint until it responds or times out
func waitForServerReady(port int, timeout time.Duration) error {
	healthURL := fmt.Sprintf("http://localhost:%d/health", port)
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err == nil {
			// Read and discard body to reuse connection
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			// Check if status is 2xx
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}

		// Wait before retrying
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for server on port %d", port)
}

// WaitForServerStopped waits for the server process to exit
func (s *ServerProcess) WaitForServerStopped(timeout time.Duration) error {
	if s.Cmd == nil || s.Cmd.Process == nil {
		return nil // Already stopped
	}

	done := make(chan error, 1)
	go func() {
		done <- s.Cmd.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for server to stop")
	}
}
