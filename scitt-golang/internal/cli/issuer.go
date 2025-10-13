package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
)

// NewIssuerCommand creates the issuer command
func NewIssuerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issuer",
		Short: "Manage issuer keys",
		Long: `Manage issuer keys for signing SCITT statements.

Subcommands:
  key generate - Generate a new ES256 key pair`,
	}

	cmd.AddCommand(NewIssuerKeyCommand())

	return cmd
}

// NewIssuerKeyCommand creates the issuer key command
func NewIssuerKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage issuer keys",
		Long:  `Manage issuer keys for signing SCITT statements.`,
	}

	cmd.AddCommand(NewIssuerKeyGenerateCommand())

	return cmd
}

type issuerKeyGenerateOptions struct {
	privateKeyPath string
	publicKeyPath  string
}

// NewIssuerKeyGenerateCommand creates the issuer key generate command
func NewIssuerKeyGenerateCommand() *cobra.Command {
	opts := &issuerKeyGenerateOptions{
		privateKeyPath: "private_key.cbor",
		publicKeyPath:  "public_key.cbor",
	}

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a new ES256 key pair",
		Long: `Generate a new ES256 (ECDSA P-256 with SHA-256) key pair for signing SCITT statements.

The keys are exported as COSE_Key in CBOR format, which can be used with the
SCITT transparency service for signing statements.

By default, this generates:
  - private_key.cbor (EC2 private key with ES256 algorithm)
  - public_key.cbor  (EC2 public key with ES256 algorithm)

Example:
  scitt issuer key generate
  scitt issuer key generate --private-key mykey.cbor --public-key mykey-pub.cbor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIssuerKeyGenerate(opts)
		},
	}

	cmd.Flags().StringVar(&opts.privateKeyPath, "private-key", opts.privateKeyPath, "path to save private key (CBOR format)")
	cmd.Flags().StringVar(&opts.publicKeyPath, "public-key", opts.publicKeyPath, "path to save public key (CBOR format)")

	return cmd
}

func runIssuerKeyGenerate(opts *issuerKeyGenerateOptions) error {
	// Generate ES256 key pair
	if verbose {
		fmt.Println("Generating ES256 (ECDSA P-256) key pair...")
	}

	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Export private key to COSE CBOR format
	privateKeyCBOR, err := cose.ExportPrivateKeyToCOSECBOR(keyPair.Private)
	if err != nil {
		return fmt.Errorf("failed to export private key: %w", err)
	}

	// Export public key to COSE CBOR format
	publicKeyCBOR, err := cose.ExportPublicKeyToCOSECBOR(keyPair.Public)
	if err != nil {
		return fmt.Errorf("failed to export public key: %w", err)
	}

	// Write private key
	if err := os.WriteFile(opts.privateKeyPath, privateKeyCBOR, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Write public key
	if err := os.WriteFile(opts.publicKeyPath, publicKeyCBOR, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	// Compute COSE key thumbprint (RFC 9679) for reference
	thumbprint, err := cose.ComputeCOSEKeyThumbprint(keyPair.Public)

	if err != nil {
		return fmt.Errorf("failed to compute COSE key thumbprint: %w", err)
	}

	fmt.Printf("✓ Key pair generated successfully\n")
	fmt.Printf("  Thumbprint:  %s\n", thumbprint)
	fmt.Printf("  Algorithm:   ES256 (ECDSA P-256 with SHA-256)\n")
	fmt.Printf("  Private key: %s (%d bytes)\n", opts.privateKeyPath, len(privateKeyCBOR))
	fmt.Printf("  Public key:  %s (%d bytes)\n", opts.publicKeyPath, len(publicKeyCBOR))

	return nil
}
