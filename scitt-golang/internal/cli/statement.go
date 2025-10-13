package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
)

// NewStatementCommand creates the statement command
func NewStatementCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "statement",
		Short: "Manage SCITT statements",
		Long: `Manage SCITT statements including signing, verification, and registration.

Subcommands:
  sign    - Sign a statement with COSE Sign1
  verify  - Verify a COSE Sign1 statement
  hash    - Compute statement hash`,
	}

	cmd.AddCommand(NewStatementSignCommand())
	cmd.AddCommand(NewStatementVerifyCommand())
	cmd.AddCommand(NewStatementHashCommand())
	cmd.AddCommand(NewStatementRegisterCommand())

	return cmd
}

type statementSignOptions struct {
	content         string
	contentType     string
	contentLocation string
	issuer          string
	subject         string
	signingKey      string
	signedStatement string
}

// NewStatementSignCommand creates the statement sign command
func NewStatementSignCommand() *cobra.Command {
	opts := &statementSignOptions{}

	cmd := &cobra.Command{
		Use:   "sign",
		Short: "Sign a content hash using COSE hash envelope",
		Long: `Sign a content hash using COSE hash envelope (RFC draft-ietf-cose-hash-envelope).

This command creates a COSE Sign1 structure with hash envelope parameters:
  - Label 258: payload-hash-alg (SHA-256)
  - Label 259: preimage-content-type (content type of the file)
  - Label 260: payload-location (URL or location hint)

The payload is the SHA-256 hash of the content, not the content itself.
CWT claims (issuer, subject) are included in the protected headers (label 15).

Example:
  scitt statement sign \
    --content ./demo/test.parquet \
    --content-type application/vnd.apache.parquet \
    --content-location https://example.com/test.parquet \
    --issuer "https://example.com" \
    --subject "urn:example:dataset:2025-10-11" \
    --signing-key ./demo/priv.cbor \
    --signed-statement ./demo/statement.cbor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatementSign(opts)
		},
	}

	cmd.Flags().StringVar(&opts.content, "content", "", "input content file (required)")
	cmd.Flags().StringVar(&opts.contentType, "content-type", "", "content type (required)")
	cmd.Flags().StringVar(&opts.contentLocation, "content-location", "", "content location URL (required)")
	cmd.Flags().StringVar(&opts.issuer, "issuer", "", "issuer claim (iss) - placed in CWT claims")
	cmd.Flags().StringVar(&opts.subject, "subject", "", "subject claim (sub) - placed in CWT claims")
	cmd.Flags().StringVar(&opts.signingKey, "signing-key", "", "private key file (CBOR COSE_Key format, required)")
	cmd.Flags().StringVar(&opts.signedStatement, "signed-statement", "", "output COSE Sign1 file (required)")

	cmd.MarkFlagRequired("content")
	cmd.MarkFlagRequired("content-type")
	cmd.MarkFlagRequired("content-location")
	cmd.MarkFlagRequired("signing-key")
	cmd.MarkFlagRequired("signed-statement")

	return cmd
}

func runStatementSign(opts *statementSignOptions) error {
	// Read content file
	content, err := os.ReadFile(opts.content)
	if err != nil {
		return fmt.Errorf("failed to read content file: %w", err)
	}

	// Read private key (CBOR COSE_Key format)
	keyBytes, err := os.ReadFile(opts.signingKey)
	if err != nil {
		return fmt.Errorf("failed to read signing key: %w", err)
	}

	privateKey, err := cose.ImportPrivateKeyFromCOSECBOR(keyBytes)
	if err != nil {
		return fmt.Errorf("failed to import private key from CBOR: %w", err)
	}

	// Extract kid from key file (must be present)
	kid, err := cose.GetKidFromCOSEKey(keyBytes)
	if err != nil {
		return fmt.Errorf("failed to extract kid from signing key: %w", err)
	}

	// Create signer
	signer, err := cose.NewES256Signer(privateKey)
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}

	// Create CWT claims if specified
	var cwtClaims cose.CWTClaimsSet
	if opts.issuer != "" || opts.subject != "" {
		cwtClaimsOpts := cose.CWTClaimsOptions{}
		if opts.issuer != "" {
			cwtClaimsOpts.Iss = opts.issuer
		}
		if opts.subject != "" {
			cwtClaimsOpts.Sub = opts.subject
		}
		cwtClaims = cose.CreateCWTClaims(cwtClaimsOpts)
	}

	// Create hash envelope options
	hashEnvelopeOpts := cose.HashEnvelopeOptions{
		ContentType:   opts.contentType,
		Location:      opts.contentLocation,
		HashAlgorithm: cose.HashAlgorithmSHA256,
	}

	// Sign hash envelope
	if verbose {
		fmt.Printf("Creating hash envelope for content (%d bytes)...\n", len(content))
		fmt.Printf("  Content Type: %s\n", opts.contentType)
		fmt.Printf("  Location: %s\n", opts.contentLocation)
	}

	coseSign1Struct, err := cose.SignHashEnvelope(
		content,
		hashEnvelopeOpts,
		signer,
		kid, // key identifier from key file
		cwtClaims,
		false, // not detached
	)
	if err != nil {
		return fmt.Errorf("failed to sign hash envelope: %w", err)
	}

	// Encode COSE Sign1 to CBOR
	coseSign1, err := cose.EncodeCoseSign1(coseSign1Struct)
	if err != nil {
		return fmt.Errorf("failed to encode COSE Sign1: %w", err)
	}

	// Write output
	if err := os.WriteFile(opts.signedStatement, coseSign1, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	// Compute content hash for display
	contentHash := sha256.Sum256(content)
	contentHashHex := hex.EncodeToString(contentHash[:])

	// Compute leaf hash (used for tile log registration)
	// This is SHA-256 of the signed statement CBOR
	leafHash := sha256.Sum256(coseSign1)
	leafHashHex := hex.EncodeToString(leafHash[:])

	fmt.Printf("✓ Hash envelope created successfully\n")
	fmt.Printf("  Content:          %s (%d bytes)\n", opts.content, len(content))
	fmt.Printf("  Content Hash:     %s\n", contentHashHex)
	fmt.Printf("  Content Type:     %s\n", opts.contentType)
	fmt.Printf("  Content Location: %s\n", opts.contentLocation)
	if opts.issuer != "" {
		fmt.Printf("  Issuer:           %s\n", opts.issuer)
	}
	if opts.subject != "" {
		fmt.Printf("  Subject:          %s\n", opts.subject)
	}
	fmt.Printf("  Signed Statement: %s (%d bytes)\n", opts.signedStatement, len(coseSign1))
	fmt.Printf("  Leaf Hash:        %s (stored in the tile log)\n", leafHashHex)

	return nil
}

type statementVerifyOptions struct {
	artifact        string
	signedStatement string
	verificationKey string
}

// NewStatementVerifyCommand creates the statement verify command
func NewStatementVerifyCommand() *cobra.Command {
	opts := &statementVerifyOptions{}

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a COSE hash envelope statement",
		Long: `Verify a COSE hash envelope statement signature and artifact hash.

This command verifies:
  1. The COSE Sign1 signature is valid
  2. The artifact hash matches the payload in the signed statement
  3. The hash envelope parameters are present

Example:
  scitt statement verify \
    --artifact ./demo/test.parquet \
    --signed-statement ./demo/statement.cbor \
    --verification-key ./demo/pub.cbor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatementVerify(opts)
		},
	}

	cmd.Flags().StringVar(&opts.artifact, "artifact", "", "artifact file to verify (required)")
	cmd.Flags().StringVar(&opts.signedStatement, "signed-statement", "", "signed statement CBOR file (required)")
	cmd.Flags().StringVar(&opts.verificationKey, "verification-key", "", "public key file (CBOR COSE_Key format, required)")

	cmd.MarkFlagRequired("artifact")
	cmd.MarkFlagRequired("signed-statement")
	cmd.MarkFlagRequired("verification-key")

	return cmd
}

func runStatementVerify(opts *statementVerifyOptions) error {
	// Read artifact
	artifact, err := os.ReadFile(opts.artifact)
	if err != nil {
		return fmt.Errorf("failed to read artifact file: %w", err)
	}

	// Read signed statement
	coseSign1Bytes, err := os.ReadFile(opts.signedStatement)
	if err != nil {
		return fmt.Errorf("failed to read signed statement: %w", err)
	}

	// Read public key (CBOR COSE_Key format)
	keyBytes, err := os.ReadFile(opts.verificationKey)
	if err != nil {
		return fmt.Errorf("failed to read verification key: %w", err)
	}

	publicKey, err := cose.ImportPublicKeyFromCOSECBOR(keyBytes)
	if err != nil {
		return fmt.Errorf("failed to import public key from CBOR: %w", err)
	}

	// Create verifier
	verifier, err := cose.NewES256Verifier(publicKey)
	if err != nil {
		return fmt.Errorf("failed to create verifier: %w", err)
	}

	// Decode COSE Sign1
	coseSign1Struct, err := cose.DecodeCoseSign1(coseSign1Bytes)
	if err != nil {
		return fmt.Errorf("failed to decode COSE Sign1: %w", err)
	}

	// Verify hash envelope using the dedicated function
	if verbose {
		fmt.Printf("Verifying hash envelope (%d bytes)...\n", len(coseSign1Bytes))
	}

	result, err := cose.VerifyHashEnvelope(coseSign1Struct, artifact, verifier)
	if err != nil {
		return fmt.Errorf("failed to verify hash envelope: %w", err)
	}

	// Check both signature and hash validity
	if !result.SignatureValid {
		fmt.Printf("✗ Signature verification failed\n")
		return fmt.Errorf("signature is invalid")
	}

	if !result.HashValid {
		fmt.Printf("✗ Hash verification failed\n")
		fmt.Printf("  The artifact hash does not match the signed statement payload\n")
		return fmt.Errorf("artifact hash mismatch")
	}

	// Both checks passed
	fmt.Printf("✓ Verification successful\n")

	// Compute and display hashes
	artifactHash := sha256.Sum256(artifact)
	artifactHashHex := hex.EncodeToString(artifactHash[:])

	leafHash := sha256.Sum256(coseSign1Bytes)
	leafHashHex := hex.EncodeToString(leafHash[:])

	fmt.Printf("  Signature:        Valid\n")
	fmt.Printf("  Artifact Hash:    %s (matches)\n", artifactHashHex)

	// Extract and display hash envelope parameters
	headers, err := cose.GetProtectedHeaders(coseSign1Struct)
	if err == nil {
		// Hash envelope parameters
		if hashAlg, ok := headers[uint64(cose.HeaderLabelPayloadHashAlg)]; ok {
			fmt.Printf("  Hash Algorithm:   SHA-256 (label %d)\n", hashAlg)
		}
		if contentType, ok := headers[uint64(cose.HeaderLabelPayloadPreimageContentType)].(string); ok {
			fmt.Printf("  Content Type:     %s\n", contentType)
		}
		if location, ok := headers[uint64(cose.HeaderLabelPayloadLocation)].(string); ok {
			fmt.Printf("  Content Location: %s\n", location)
		}

		// CWT claims
		if cwtClaims, ok := headers[uint64(cose.HeaderLabelCWTClaims)].(map[interface{}]interface{}); ok {
			if iss, ok := cwtClaims[uint64(1)].(string); ok {
				fmt.Printf("  Issuer:           %s\n", iss)
			}
			if sub, ok := cwtClaims[uint64(2)].(string); ok {
				fmt.Printf("  Subject:          %s\n", sub)
			}
		}
	}
	fmt.Printf("  Leaf Hash:        %s (stored in the tile log)\n", leafHashHex)

	return nil
}

type statementHashOptions struct {
	input string
}

// NewStatementHashCommand creates the statement hash command
func NewStatementHashCommand() *cobra.Command {
	opts := &statementHashOptions{}

	cmd := &cobra.Command{
		Use:   "hash",
		Short: "Compute statement hash",
		Long: `Compute the SHA-256 hash of a COSE Sign1 statement.

The statement hash is used as the unique identifier for registration.

Example:
  scitt statement hash --input statement.cbor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatementHash(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.input, "input", "i", "", "input COSE Sign1 file (required)")

	cmd.MarkFlagRequired("input")

	return cmd
}

func runStatementHash(opts *statementHashOptions) error {
	// Read COSE Sign1
	coseSign1, err := os.ReadFile(opts.input)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Compute hash
	hash := sha256.Sum256(coseSign1)
	hashHex := hex.EncodeToString(hash[:])

	fmt.Printf("Statement Hash (SHA-256):\n")
	fmt.Printf("  %s\n", hashHex)
	fmt.Printf("\nFile: %s (%d bytes)\n", opts.input, len(coseSign1))

	return nil
}

type statementRegisterOptions struct {
	service   string
	apiKey    string
	statement string
	receipt   string
}

// NewStatementRegisterCommand creates the statement register command
func NewStatementRegisterCommand() *cobra.Command {
	opts := &statementRegisterOptions{}

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a statement with a transparency service",
		Long: `Register a SCITT statement with a transparency service.

This command:
  1. Reads the signed statement CBOR file
  2. POSTs it to the service /entries endpoint
  3. Authenticates using the API key in the Authorization header
  4. Saves the returned receipt to a file

Example:
  scitt statement register \
    --service http://0.0.0.0:8080 \
    --api-key f1d1784415b3021fca4bde0fbcb8ac6ef4a2210d4e49252c204c4c9f8812a95e \
    --statement ./demo/statement.cbor \
    --receipt ./demo/statement.receipt.cbor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatementRegister(opts)
		},
	}

	cmd.Flags().StringVar(&opts.service, "service", "", "transparency service URL (required)")
	cmd.Flags().StringVar(&opts.apiKey, "api-key", "", "API key for authentication (required)")
	cmd.Flags().StringVar(&opts.statement, "statement", "", "signed statement CBOR file (required)")
	cmd.Flags().StringVar(&opts.receipt, "receipt", "", "output receipt CBOR file (required)")

	cmd.MarkFlagRequired("service")
	cmd.MarkFlagRequired("api-key")
	cmd.MarkFlagRequired("statement")
	cmd.MarkFlagRequired("receipt")

	return cmd
}

func runStatementRegister(opts *statementRegisterOptions) error {
	// Read statement file
	statementBytes, err := os.ReadFile(opts.statement)
	if err != nil {
		return fmt.Errorf("failed to read statement file: %w", err)
	}

	// Compute leaf hash for display
	leafHash := sha256.Sum256(statementBytes)
	leafHashHex := hex.EncodeToString(leafHash[:])

	if verbose {
		fmt.Printf("Registering statement (%d bytes)...\n", len(statementBytes))
		fmt.Printf("  Leaf Hash: %s\n", leafHashHex)
	}

	// Create HTTP request
	url := opts.service + "/entries"
	req, err := http.NewRequest("POST", url, bytes.NewReader(statementBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/cose")
	req.Header.Set("Authorization", "Bearer "+opts.apiKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode == http.StatusUnauthorized {
		fmt.Printf("✗ Registration failed: Unauthorized (401)\n")
		fmt.Printf("  The API key is invalid or missing\n")
		return fmt.Errorf("authentication failed: invalid API key")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("✗ Registration failed: HTTP %d\n", resp.StatusCode)
		fmt.Printf("  Response: %s\n", string(bodyBytes))
		return fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}

	// Read receipt response
	receiptBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Save receipt to file
	if err := os.WriteFile(opts.receipt, receiptBytes, 0644); err != nil {
		return fmt.Errorf("failed to write receipt file: %w", err)
	}

	fmt.Printf("✓ Statement registered successfully\n")
	fmt.Printf("  Statement:  %s (%d bytes)\n", opts.statement, len(statementBytes))
	fmt.Printf("  Leaf Hash:  %s\n", leafHashHex)
	fmt.Printf("  Receipt:    %s (%d bytes)\n", opts.receipt, len(receiptBytes))
	fmt.Printf("  Service:    %s\n", opts.service)

	return nil
}
