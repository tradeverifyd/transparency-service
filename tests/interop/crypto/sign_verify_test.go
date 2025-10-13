package crypto

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tradeverifyd/scitt/tests/interop/lib"
)

// TestGoSignsTypeScriptVerifies validates that statements signed by Go implementation
// can be successfully verified by TypeScript implementation (FR-017)
func TestGoSignsTypeScriptVerifies(t *testing.T) {
	env := lib.SetupTestEnv(t)
	defer env.Cleanup()

	// Load all test keypairs
	keypairs := []string{"alice", "bob", "charlie", "dave", "eve"}
	payloads := []string{"small", "medium", "large"}

	testCount := 0
	for _, kp := range keypairs {
		for _, payload := range payloads {
			testCount++
			t.Run(kp+"_"+payload, func(t *testing.T) {
				// Load keypair
				keypairPath := filepath.Join("fixtures", "keys", "keypair_"+kp+".json")
				keypairData, err := os.ReadFile(keypairPath)
				if err != nil {
					t.Fatalf("Failed to read keypair: %v", err)
				}

				var keypair map[string]interface{}
				if err := json.Unmarshal(keypairData, &keypair); err != nil {
					t.Fatalf("Failed to parse keypair: %v", err)
				}

				// Load payload
				payloadPath := filepath.Join("fixtures", "payloads", payload+".json")
				payloadData, err := os.ReadFile(payloadPath)
				if err != nil {
					t.Fatalf("Failed to read payload: %v", err)
				}

				// Write private key to temp file for Go CLI
				privateKeyFile := filepath.Join(env.TempDir, "private_key.json")
				if err := os.WriteFile(privateKeyFile, keypairData, 0600); err != nil {
					t.Fatalf("Failed to write private key: %v", err)
				}

				// Write payload to temp file
				payloadFile := filepath.Join(env.TempDir, "payload.json")
				if err := os.WriteFile(payloadFile, payloadData, 0600); err != nil {
					t.Fatalf("Failed to write payload: %v", err)
				}

				// Sign with Go CLI
				signedFile := filepath.Join(env.TempDir, "signed.cose")
				goSignCmd := []string{
					"statement", "sign",
					"--key", privateKeyFile,
					"--payload", payloadFile,
					"--output", signedFile,
				}

				goOutput, goErr, goExitCode := lib.RunGoCLI(goSignCmd, env.TempDir, 30000)
				if goExitCode != 0 {
					t.Fatalf("Go sign failed (exit %d): stdout=%s, stderr=%s", goExitCode, goOutput, goErr)
				}

				// Verify signed statement exists
				if _, err := os.Stat(signedFile); os.IsNotExist(err) {
					t.Fatalf("Signed statement not created by Go: %v", err)
				}

				// Write public key for TypeScript verification
				publicKeyFile := filepath.Join(env.TempDir, "public_key.json")
				publicKeyData, err := json.Marshal(keypair["public_key"])
				if err != nil {
					t.Fatalf("Failed to marshal public key: %v", err)
				}
				if err := os.WriteFile(publicKeyFile, publicKeyData, 0600); err != nil {
					t.Fatalf("Failed to write public key: %v", err)
				}

				// Verify with TypeScript CLI
				tsVerifyCmd := []string{
					"statement", "verify",
					"--key", publicKeyFile,
					"--statement", signedFile,
				}

				tsOutput, tsErr, tsExitCode := lib.RunTsCLI(tsVerifyCmd, env.TempDir, 30000)
				if tsExitCode != 0 {
					t.Errorf("TypeScript verification failed for Go-signed statement (exit %d): stdout=%s, stderr=%s", tsExitCode, tsOutput, tsErr)
					t.Logf("Keypair: %s, Payload: %s", kp, payload)
					return
				}

				t.Logf("✓ Go signed, TypeScript verified: %s/%s", kp, payload)
			})
		}
	}

	t.Logf("Completed %d Go→TypeScript sign/verify combinations", testCount)
}

// TestTypeScriptSignsGoVerifies validates that statements signed by TypeScript implementation
// can be successfully verified by Go implementation (FR-018)
func TestTypeScriptSignsGoVerifies(t *testing.T) {
	env := lib.SetupTestEnv(t)
	defer env.Cleanup()

	// Load all test keypairs
	keypairs := []string{"alice", "bob", "charlie", "dave", "eve"}
	payloads := []string{"small", "medium", "large"}

	testCount := 0
	for _, kp := range keypairs {
		for _, payload := range payloads {
			testCount++
			t.Run(kp+"_"+payload, func(t *testing.T) {
				// Load keypair
				keypairPath := filepath.Join("fixtures", "keys", "keypair_"+kp+".json")
				keypairData, err := os.ReadFile(keypairPath)
				if err != nil {
					t.Fatalf("Failed to read keypair: %v", err)
				}

				var keypair map[string]interface{}
				if err := json.Unmarshal(keypairData, &keypair); err != nil {
					t.Fatalf("Failed to parse keypair: %v", err)
				}

				// Load payload
				payloadPath := filepath.Join("fixtures", "payloads", payload+".json")
				payloadData, err := os.ReadFile(payloadPath)
				if err != nil {
					t.Fatalf("Failed to read payload: %v", err)
				}

				// Write private key to temp file for TypeScript CLI
				privateKeyFile := filepath.Join(env.TempDir, "private_key.json")
				if err := os.WriteFile(privateKeyFile, keypairData, 0600); err != nil {
					t.Fatalf("Failed to write private key: %v", err)
				}

				// Write payload to temp file
				payloadFile := filepath.Join(env.TempDir, "payload.json")
				if err := os.WriteFile(payloadFile, payloadData, 0600); err != nil {
					t.Fatalf("Failed to write payload: %v", err)
				}

				// Sign with TypeScript CLI
				signedFile := filepath.Join(env.TempDir, "signed.cose")
				tsSignCmd := []string{
					"statement", "sign",
					"--key", privateKeyFile,
					"--payload", payloadFile,
					"--output", signedFile,
				}

				tsOutput, tsErr, tsExitCode := lib.RunTsCLI(tsSignCmd, env.TempDir, 30000)
				if tsExitCode != 0 {
					t.Fatalf("TypeScript sign failed (exit %d): stdout=%s, stderr=%s", tsExitCode, tsOutput, tsErr)
				}

				// Verify signed statement exists
				if _, err := os.Stat(signedFile); os.IsNotExist(err) {
					t.Fatalf("Signed statement not created by TypeScript: %v", err)
				}

				// Write public key for Go verification
				publicKeyFile := filepath.Join(env.TempDir, "public_key.json")
				publicKeyData, err := json.Marshal(keypair["public_key"])
				if err != nil {
					t.Fatalf("Failed to marshal public key: %v", err)
				}
				if err := os.WriteFile(publicKeyFile, publicKeyData, 0600); err != nil {
					t.Fatalf("Failed to write public key: %v", err)
				}

				// Verify with Go CLI
				goVerifyCmd := []string{
					"statement", "verify",
					"--key", publicKeyFile,
					"--statement", signedFile,
				}

				goOutput, goErr, goExitCode := lib.RunGoCLI(goVerifyCmd, env.TempDir, 30000)
				if goExitCode != 0 {
					t.Errorf("Go verification failed for TypeScript-signed statement (exit %d): stdout=%s, stderr=%s", goExitCode, goOutput, goErr)
					t.Logf("Keypair: %s, Payload: %s", kp, payload)
					return
				}

				t.Logf("✓ TypeScript signed, Go verified: %s/%s", kp, payload)
			})
		}
	}

	t.Logf("Completed %d TypeScript→Go sign/verify combinations", testCount)
}
