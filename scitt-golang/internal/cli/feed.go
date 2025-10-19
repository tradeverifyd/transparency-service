package cli

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/generator"
)

var (
	noSign     bool
	noRegister bool
	serviceURL string
)

// NewFeedCommand creates the feed subcommand
func NewFeedCommand() *cobra.Command {
	feedCmd := &cobra.Command{
		Use:   "feed",
		Short: "Synthetic Supply Chain Feed Generator",
		Long: `Generate synthetic supply chain datasets for AI-capable laptops.

This command creates a complete feed directory with:
  - 90-110 documents across 10 categories
  - 3 semiconductor company identities (foundry, IDM, fabless)
  - ES256 key pairs for each company
  - Interactive workflows for signing and registration`,
	}

	// Add subcommands
	feedCmd.AddCommand(NewFeedGenerateCommand())

	return feedCmd
}

// NewFeedGenerateCommand creates the 'feed generate' subcommand
func NewFeedGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate synthetic supply chain dataset",
		Long: `Generate a complete synthetic supply chain dataset with documents, keys, and optional signing/registration.

The generator creates:
  - Timestamped feed directory (feed-YYYY-MM-DD-HHMMSS/)
  - Company subdirectories with document folders
  - 96+ JSON documents across 10 types (wafers, minerals, chips, firmware, SBOMs, memory, AI datasets/models, CVEs, logistics)
  - Metadata file with timestamp and seed

Interactive Workflow:
  1. Generate documents for 3 companies (foundry, IDM, fabless)
  2. Optional: Sign documents with ES256 keys (creates .cbor files)
  3. Optional: Register to SCITT service (creates .receipt.cbor files)

Examples:
  # Generate feed with interactive prompts for signing and registration
  scitt feed generate

  # Generate and sign, but skip registration
  scitt feed generate --no-register

  # Generate documents only (no signing or registration)
  scitt feed generate --no-sign --no-register

  # Generate, sign, and register to custom service
  scitt feed generate --service-url http://localhost:3000

Output Structure:
  feed-YYYY-MM-DD-HHMMSS/
  ├── metadata.json                          # Feed metadata
  ├── pacific-silicon-foundry/               # Foundry company
  │   ├── private_key.cbor                   # ES256 private key (if signed)
  │   ├── public_key.cbor                    # ES256 public key (if signed)
  │   └── documents/
  │       ├── wafer-batch-001.json           # JSON documents
  │       ├── wafer-batch-001.cbor           # Signed statements (if signed)
  │       └── wafer-batch-001.receipt.cbor   # Receipts (if registered)
  ├── apex-semiconductor-corp/               # IDM company
  └── quantum-chip-design/                   # Fabless company`,
		RunE: runFeedGenerate,
	}

	// Add flags
	cmd.Flags().BoolVar(&noSign, "no-sign", false, "Skip document signing workflow")
	cmd.Flags().BoolVar(&noRegister, "no-register", false, "Skip registration workflow")
	cmd.Flags().StringVar(&serviceURL, "service-url", "http://127.0.0.1:8000", "SCITT service URL for registration")

	return cmd
}

// runFeedGenerate executes the feed generation workflow
func runFeedGenerate(cmd *cobra.Command, args []string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	fmt.Println("Synthetic Supply Chain Feed Generator")
	fmt.Println()

	// Step 1: Create feed directory
	fmt.Println("[1/3] Creating feed directory...")
	feedDir, timestamp, err := generator.CreateFeedDirectory(workDir)
	if err != nil {
		return fmt.Errorf("failed to create feed directory: %w", err)
	}

	feedDirName := filepath.Base(feedDir)
	fmt.Printf("   Created: %s\n", feedDirName)
	fmt.Println()

	// Step 2: Generate companies
	companies := generator.GenerateCompanies()
	fmt.Printf("[2/3] Generating documents for %d companies...\n", len(companies))
	fmt.Println()

	// Step 3: Generate documents for each company
	rng := generator.NewSeededRand(timestamp)
	totalDocs := 0

	for _, company := range companies {
		fmt.Printf("   %s (%s)\n", company.Name, company.Role)
		documentsDir := filepath.Join(feedDir, company.Directory, "documents")

		var allDocs []generator.Document

		// Generate all document types
		switch company.Role {
		case "foundry":
			// Foundry generates: wafers, minerals, logistics
			for _, doc := range generator.GenerateWaferBatches(company, rng) {
				allDocs = append(allDocs, &doc)
			}
			for _, doc := range generator.GenerateMineralSourcing(company, rng) {
				allDocs = append(allDocs, &doc)
			}
			for _, doc := range generator.GenerateLogisticsTracking(company, rng) {
				allDocs = append(allDocs, &doc)
			}

		case "IDM":
			// IDM generates: chips, firmware, SBOMs
			for _, doc := range generator.GenerateChipSpecifications(company, rng) {
				allDocs = append(allDocs, &doc)
			}
			for _, doc := range generator.GenerateFirmwareManifests(company, rng) {
				allDocs = append(allDocs, &doc)
			}
			for _, doc := range generator.GenerateSBOMs(company, rng) {
				allDocs = append(allDocs, &doc)
			}

		case "fabless":
			// Fabless generates: memory, AI datasets, AI models, CVEs
			for _, doc := range generator.GenerateMemorySpecifications(company, rng) {
				allDocs = append(allDocs, &doc)
			}
			for _, doc := range generator.GenerateAITrainingDatasets(company, rng) {
				allDocs = append(allDocs, &doc)
			}
			for _, doc := range generator.GenerateAIModelSpecifications(company, rng) {
				allDocs = append(allDocs, &doc)
			}
			for _, doc := range generator.GenerateCVEDocuments(company, rng) {
				allDocs = append(allDocs, &doc)
			}
		}

		// Write documents with progress bar
		bar := progressbar.NewOptions(len(allDocs),
			progressbar.OptionSetDescription(fmt.Sprintf("      Writing %d documents", len(allDocs))),
			progressbar.OptionSetWidth(40),
			progressbar.OptionShowCount(),
			progressbar.OptionClearOnFinish(),
		)

		for _, doc := range allDocs {
			if err := generator.WriteDocumentToFile(doc, documentsDir); err != nil {
				return fmt.Errorf("failed to write document for %s: %w", company.Name, err)
			}
			bar.Add(1)
		}

		fmt.Printf("       Wrote %d documents\n", len(allDocs))
		totalDocs += len(allDocs)
	}

	fmt.Println()
	fmt.Printf("[3/3] Feed generated successfully!\n")
	fmt.Printf("   Location: %s\n", feedDirName)
	fmt.Printf("   Total documents: %d\n", totalDocs)
	fmt.Printf("   Timestamp: %s\n", timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println()

	// Step 4: Interactive signing workflow (T022)
	if !noSign {
		fmt.Println("Ready to sign documents? (yes/no):")
		var response string
		fmt.Scanln(&response)

		if response == "yes" || response == "y" {
			// Generate keys for all companies
			if err := generateCompanyKeys(feedDir, companies); err != nil {
				return fmt.Errorf("key generation failed: %w", err)
			}

			fmt.Println()
			fmt.Println("Signing documents for companies...")

			// Sign documents for each company
			for i, company := range companies {
				fmt.Printf("   [%d/%d] Signing documents for %s...\n", i+1, len(companies), company.Name)
				if err := signDocumentsForCompany(feedDir, company); err != nil {
					return fmt.Errorf("signing failed for %s: %w", company.Name, err)
				}
				fmt.Printf("      Signed documents successfully!\n")
			}

			fmt.Println()
			fmt.Println("All documents signed successfully!")
		} else {
			fmt.Println("Skipping document signing.")
		}
	}

	// Step 5: Interactive registration workflow (T025)
	if !noRegister {
		// Check if documents were signed
		hasSignedStatements := false
		for _, company := range companies {
			documentsDir := filepath.Join(feedDir, company.Directory, "documents")
			entries, err := os.ReadDir(documentsDir)
			if err == nil {
				for _, entry := range entries {
					if filepath.Ext(entry.Name()) == ".cbor" {
						hasSignedStatements = true
						break
					}
				}
			}
			if hasSignedStatements {
				break
			}
		}

		if !hasSignedStatements {
			fmt.Println()
			fmt.Println("NOTE: No signed statements found. Sign documents first to enable registration.")
			return nil
		}

		fmt.Println()
		fmt.Printf("Ready to register statements to service at %s? (yes/no):\n", serviceURL)
		var response string
		fmt.Scanln(&response)

		if response == "yes" || response == "y" {
			// Validate service connection
			fmt.Println()
			fmt.Printf("Validating connection to %s...\n", serviceURL)
			if err := validateServiceConnection(serviceURL); err != nil {
				return fmt.Errorf("service validation failed: %w\nPlease check that the service is running and accessible", err)
			}
			fmt.Println("   Service is reachable")

			fmt.Println()
			fmt.Println("Registering statements for companies...")

			// Register statements for each company
			totalRegistered := 0
			for i, company := range companies {
				fmt.Printf("   [%d/%d] Registering statements for %s...\n", i+1, len(companies), company.Name)
				count, err := registerStatementsForCompany(feedDir, company, serviceURL)
				if err != nil {
					return fmt.Errorf("registration failed for %s: %w", company.Name, err)
				}
				fmt.Printf("      Registered %d statements successfully!\n", count)
				totalRegistered += count
			}

			fmt.Println()
			fmt.Printf("All statements registered successfully! (Total: %d)\n", totalRegistered)
			fmt.Println("Receipts saved to company documents directories (.receipt.cbor files)")
		} else {
			fmt.Println("Skipping statement registration.")
		}
	}

	return nil
}

// generateCompanyKeys generates ES256 key pairs for each company
// Uses exec.Command to call existing "scitt issuer key generate" command
func generateCompanyKeys(feedDir string, companies []generator.Company) error {
	fmt.Println()
	fmt.Println("Generating ES256 keys for companies...")

	// Get the path to the scitt binary
	scittPath, err := exec.LookPath("scitt")
	if err != nil {
		// Try relative path
		scittPath = "./scitt"
	}

	for i, company := range companies {
		fmt.Printf("   [%d/%d] Creating keys for %s...\n", i+1, len(companies), company.Name)

		companyDir := filepath.Join(feedDir, company.Directory)
		privateKeyPath := filepath.Join(companyDir, "private_key.cbor")
		publicKeyPath := filepath.Join(companyDir, "public_key.cbor")

		// Execute: scitt issuer key generate --private-key <path> --public-key <path>
		cmd := exec.Command(scittPath, "issuer", "key", "generate",
			"--private-key", privateKeyPath,
			"--public-key", publicKeyPath)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to generate keys for %s: %w\nOutput: %s", company.Name, err, string(output))
		}
	}

	fmt.Println("   Keys generated successfully!")
	return nil
}

// signDocumentsForCompany signs all JSON documents for a company using scitt statement sign
func signDocumentsForCompany(feedDir string, company generator.Company) error {
	companyDir := filepath.Join(feedDir, company.Directory)
	documentsDir := filepath.Join(companyDir, "documents")
	privateKeyPath := filepath.Join(companyDir, "private_key.cbor")

	// Check if private key exists
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("private key not found for %s: %s", company.Name, privateKeyPath)
	}

	// Get all JSON files
	entries, err := os.ReadDir(documentsDir)
	if err != nil {
		return fmt.Errorf("failed to read documents directory for %s: %w", company.Name, err)
	}

	var jsonFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			jsonFiles = append(jsonFiles, entry.Name())
		}
	}

	if len(jsonFiles) == 0 {
		return fmt.Errorf("no JSON documents found for %s", company.Name)
	}

	// Get scitt binary path
	scittPath, err := exec.LookPath("scitt")
	if err != nil {
		scittPath = "./scitt"
	}

	// Progress bar for signing
	bar := progressbar.NewOptions(len(jsonFiles),
		progressbar.OptionSetDescription(fmt.Sprintf("      Signing %d documents", len(jsonFiles))),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionClearOnFinish(),
	)

	signedCount := 0
	for _, jsonFile := range jsonFiles {
		jsonPath := filepath.Join(documentsDir, jsonFile)
		
		// Derive signed statement path (replace .json with .cbor)
		cborFile := jsonFile[:len(jsonFile)-5] + ".cbor"
		cborPath := filepath.Join(documentsDir, cborFile)

		// For content-location, use the URN from the document
		// For now, use a simplified approach with company directory
		contentLocation := fmt.Sprintf("https://%s.example/documents/%s", company.Directory, jsonFile)

		// Execute: scitt statement sign
		cmd := exec.Command(scittPath, "statement", "sign",
			"--content", jsonPath,
			"--content-type", "application/json",
			"--content-location", contentLocation,
			"--issuer", company.URN,
			"--subject", company.URN, // Simplified - ideally should be document-specific
			"--signing-key", privateKeyPath,
			"--signed-statement", cborPath)

		output, err := cmd.CombinedOutput()
		if err != nil {
			// Log error but continue with remaining documents
			fmt.Printf("\n   Warning: Failed to sign %s: %v\n   Output: %s\n", jsonFile, err, string(output))
			continue
		}

		signedCount++
		bar.Add(1)
	}

	if signedCount == 0 {
		return fmt.Errorf("failed to sign any documents for %s", company.Name)
	}

	return nil
}

// validateServiceConnection checks if the SCITT service is reachable (T023)
func validateServiceConnection(serviceURL string) error {
	// Check well-known configuration endpoint
	configURL := serviceURL + "/.well-known/scitt-configuration"
	
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	resp, err := client.Get(configURL)
	if err != nil {
		return fmt.Errorf("failed to connect to service: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service returned status %d", resp.StatusCode)
	}
	
	return nil
}

// registerStatementsForCompany registers all signed statements for a company (T024)
func registerStatementsForCompany(feedDir string, company generator.Company, serviceURL string) (int, error) {
	companyDir := filepath.Join(feedDir, company.Directory)
	documentsDir := filepath.Join(companyDir, "documents")
	
	// Get all CBOR files (signed statements)
	entries, err := os.ReadDir(documentsDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read documents directory for %s: %w", company.Name, err)
	}
	
	var cborFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".cbor" {
			cborFiles = append(cborFiles, entry.Name())
		}
	}
	
	if len(cborFiles) == 0 {
		return 0, fmt.Errorf("no signed statements (.cbor) found for %s", company.Name)
	}
	
	// Get scitt binary path
	scittPath, err := exec.LookPath("scitt")
	if err != nil {
		scittPath = "./scitt"
	}
	
	// Progress bar for registration
	bar := progressbar.NewOptions(len(cborFiles),
		progressbar.OptionSetDescription(fmt.Sprintf("      Registering %d statements", len(cborFiles))),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionClearOnFinish(),
	)
	
	registeredCount := 0
	for _, cborFile := range cborFiles {
		cborPath := filepath.Join(documentsDir, cborFile)
		
		// Derive receipt path (replace .cbor with .receipt.cbor)
		receiptFile := cborFile[:len(cborFile)-5] + ".receipt.cbor"
		receiptPath := filepath.Join(documentsDir, receiptFile)
		
		// Execute: scitt receipt register
		cmd := exec.Command(scittPath, "receipt", "register",
			"--signed-statement", cborPath,
			"--receipt", receiptPath,
			"--service-url", serviceURL)
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Log error but continue with remaining statements
			fmt.Printf("\n   Warning: Failed to register %s: %v\n   Output: %s\n", cborFile, err, string(output))
			continue
		}
		
		registeredCount++
		bar.Add(1)
	}
	
	if registeredCount == 0 {
		return 0, fmt.Errorf("failed to register any statements for %s", company.Name)
	}
	
	return registeredCount, nil
}
