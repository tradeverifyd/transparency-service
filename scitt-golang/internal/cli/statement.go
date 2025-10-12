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
