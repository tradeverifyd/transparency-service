package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	cmd.AddCommand(NewStatementHashEnvelopeCommand())

	return cmd
}

type statementSignOptions struct {
	input      string
	output     string
	keyPath    string
	kid        string
	contentType string
	issuer     string
	subject    string
}

// NewStatementSignCommand creates the statement sign command
func NewStatementSignCommand() *cobra.Command {
	opts := &statementSignOptions{}

	cmd := &cobra.Command{
		Use:   "sign",
		Short: "Sign a statement with COSE Sign1",
		Long: `Sign a statement payload with COSE Sign1 using ES256.

The signed statement can be registered with a transparency service.

Example:
  scitt statement sign --input payload.json --key private-key.pem --output statement.cbor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatementSign(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.input, "input", "i", "", "input payload file (required)")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output COSE Sign1 file (required)")
	cmd.Flags().StringVarP(&opts.keyPath, "key", "k", "", "private key file (PEM format, required)")
	cmd.Flags().StringVar(&opts.kid, "kid", "", "key ID")
	cmd.Flags().StringVar(&opts.contentType, "content-type", "application/json", "content type")
	cmd.Flags().StringVar(&opts.issuer, "issuer", "", "issuer (iss claim)")
	cmd.Flags().StringVar(&opts.subject, "subject", "", "subject (sub claim)")

	cmd.MarkFlagRequired("input")
	cmd.MarkFlagRequired("output")
	cmd.MarkFlagRequired("key")

	return cmd
}

func runStatementSign(opts *statementSignOptions) error {
	// Read input payload
	payload, err := os.ReadFile(opts.input)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Read private key
	keyPEM, err := os.ReadFile(opts.keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	privateKey, err := cose.ImportPrivateKeyFromPEM(string(keyPEM))
	if err != nil {
		return fmt.Errorf("failed to import private key: %w", err)
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

	// Create protected headers
	headerOpts := cose.ProtectedHeadersOptions{
		Alg:       cose.AlgorithmES256,
		Cty:       opts.contentType,
		CWTClaims: cwtClaims,
	}

	if opts.kid != "" {
		headerOpts.Kid = opts.kid
	}

	headers := cose.CreateProtectedHeaders(headerOpts)

	// COSE Sign1 options
	coseOpts := cose.CoseSign1Options{}

	// Sign payload
	if verbose {
		fmt.Printf("Signing payload (%d bytes)...\n", len(payload))
	}

	coseSign1Struct, err := cose.CreateCoseSign1(headers, payload, signer, coseOpts)
	if err != nil {
		return fmt.Errorf("failed to create COSE Sign1: %w", err)
	}

	// Encode COSE Sign1 to CBOR
	coseSign1, err := cose.EncodeCoseSign1(coseSign1Struct)
	if err != nil {
		return fmt.Errorf("failed to encode COSE Sign1: %w", err)
	}

	// Write output
	if err := os.WriteFile(opts.output, coseSign1, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	// Compute statement hash
	hash := sha256.Sum256(coseSign1)
	hashHex := hex.EncodeToString(hash[:])

	fmt.Printf("✓ Statement signed successfully\n")
	fmt.Printf("  Input:  %s (%d bytes)\n", opts.input, len(payload))
	fmt.Printf("  Output: %s (%d bytes)\n", opts.output, len(coseSign1))
	fmt.Printf("  Hash:   %s\n", hashHex)

	return nil
}

type statementVerifyOptions struct {
	input   string
	keyPath string
}

// NewStatementVerifyCommand creates the statement verify command
func NewStatementVerifyCommand() *cobra.Command {
	opts := &statementVerifyOptions{}

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a COSE Sign1 statement",
		Long: `Verify a COSE Sign1 statement signature.

Example:
  scitt statement verify --input statement.cbor --key public-key.jwk`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatementVerify(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.input, "input", "i", "", "input COSE Sign1 file (required)")
	cmd.Flags().StringVarP(&opts.keyPath, "key", "k", "", "public key file (JWK format, required)")

	cmd.MarkFlagRequired("input")
	cmd.MarkFlagRequired("key")

	return cmd
}

func runStatementVerify(opts *statementVerifyOptions) error {
	// Read COSE Sign1
	coseSign1, err := os.ReadFile(opts.input)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Read public key
	keyJWKBytes, err := os.ReadFile(opts.keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	keyJWK, err := cose.UnmarshalJWK(keyJWKBytes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JWK: %w", err)
	}

	publicKey, err := cose.ImportPublicKeyFromJWK(keyJWK)
	if err != nil {
		return fmt.Errorf("failed to import public key: %w", err)
	}

	// Create verifier
	verifier, err := cose.NewES256Verifier(publicKey)
	if err != nil {
		return fmt.Errorf("failed to create verifier: %w", err)
	}

	// Decode COSE Sign1
	coseSign1Struct, err := cose.DecodeCoseSign1(coseSign1)
	if err != nil {
		return fmt.Errorf("failed to decode COSE Sign1: %w", err)
	}

	// Verify signature
	if verbose {
		fmt.Printf("Verifying statement (%d bytes)...\n", len(coseSign1))
	}

	valid, err := cose.VerifyCoseSign1(coseSign1Struct, verifier, nil)
	if err != nil {
		return fmt.Errorf("failed to verify: %w", err)
	}

	if valid {
		fmt.Printf("✓ Statement signature is valid\n")
		fmt.Printf("  Payload: %d bytes\n", len(coseSign1Struct.Payload))

		// Try to decode protected headers
		headers, err := cose.GetProtectedHeaders(coseSign1Struct)
		if err == nil {
			if cty, ok := headers[cose.HeaderLabelContentType].(string); ok {
				fmt.Printf("  Content-Type: %s\n", cty)
			}
			if kid, ok := headers[cose.HeaderLabelKid]; ok {
				fmt.Printf("  Key ID: %v\n", kid)
			}
		}
	} else {
		fmt.Printf("✗ Statement signature is invalid\n")
		return fmt.Errorf("signature verification failed")
	}

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

type statementHashEnvelopeOptions struct {
	content         string
	contentType     string
	contentLocation string
	issuer          string
	subject         string
	signingKey      string
	signedStatement string
}

// NewStatementHashEnvelopeCommand creates the statement hash-envelope command
func NewStatementHashEnvelopeCommand() *cobra.Command {
	opts := &statementHashEnvelopeOptions{}

	cmd := &cobra.Command{
		Use:   "hash-envelope",
		Short: "Sign a content hash using COSE hash envelope",
		Long: `Sign a content hash using COSE hash envelope (RFC draft-ietf-cose-hash-envelope).

This command creates a COSE Sign1 structure with hash envelope parameters:
  - Label 258: payload-hash-alg (SHA-256)
  - Label 259: preimage-content-type (content type of the file)
  - Label 260: payload-location (URL or location hint)

The payload is the SHA-256 hash of the content, not the content itself.
CWT claims (issuer, subject) are included in the protected headers (label 15).

Example:
  scitt statement hash-envelope \
    --content ./demo/test.parquet \
    --content-type application/vnd.apache.parquet \
    --content-location https://example.com/test.parquet \
    --issuer "https://example.com" \
    --subject "urn:example:dataset:2025-10-11" \
    --signing-key ./demo/priv.cbor \
    --signed-statement ./demo/statement.cbor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatementHashEnvelope(opts)
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

func runStatementHashEnvelope(opts *statementHashEnvelopeOptions) error {
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

	// Compute statement hash
	statementHash := sha256.Sum256(coseSign1)
	statementHashHex := hex.EncodeToString(statementHash[:])

	fmt.Printf("✓ Hash envelope created successfully\n")
	fmt.Printf("  Content:         %s (%d bytes)\n", opts.content, len(content))
	fmt.Printf("  Content Hash:    %s\n", contentHashHex)
	fmt.Printf("  Content Type:    %s\n", opts.contentType)
	fmt.Printf("  Location:        %s\n", opts.contentLocation)
	if opts.issuer != "" {
		fmt.Printf("  Issuer:          %s\n", opts.issuer)
	}
	if opts.subject != "" {
		fmt.Printf("  Subject:         %s\n", opts.subject)
	}
	fmt.Printf("  Signed Statement: %s (%d bytes)\n", opts.signedStatement, len(coseSign1))
	fmt.Printf("  Statement Hash:   %s\n", statementHashHex)

	return nil
}
