package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/generator"
)

// NewFeedCommand creates the feed subcommand
func NewFeedCommand() *cobra.Command {
	feedCmd := &cobra.Command{
		Use:   "feed",
		Short: "Synthetic Supply Chain Feed Generator",
		Long: `Generate synthetic supply chain datasets for AI-capable laptops.

This command creates a complete feed directory with:
  - 80-110 documents across 10 categories
  - 3 semiconductor company identities (foundry, IDM, fabless)
  - ES256 key pairs for each company
  - Separate workflows for generation, signing, and registration`,
	}

	// Add subcommands
	feedCmd.AddCommand(NewFeedGenerateCommand())
	feedCmd.AddCommand(NewFeedSignCommand())
	feedCmd.AddCommand(NewFeedRegisterCommand())

	return feedCmd
}

// NewFeedGenerateCommand creates the 'feed generate' subcommand
func NewFeedGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate synthetic supply chain dataset",
		Long: `Generate a synthetic supply chain dataset with documents only.

The generator creates:
  - Timestamped feed directory (feed-YYYY-MM-DD-HHMMSS/)
  - Company subdirectories with document folders
  - 80-110 JSON documents across 10 types (wafers, minerals, chips, firmware, SBOMs, memory, AI datasets/models, CVEs, logistics)
  - Metadata file with timestamp and seed

After generation, use 'feed sign' to create COSE statements and 'feed register' to register to transparency services.

Examples:
  # Generate feed
  scitt feed generate

  # Then sign documents
  scitt feed sign feed-2025-10-18-143022

  # Then register to service(s)
  scitt feed register feed-2025-10-18-143022 --service-url http://localhost:8000
  scitt feed register feed-2025-10-18-143022 --service-url http://localhost:9000

Output Structure:
  feed-YYYY-MM-DD-HHMMSS/
  ├── metadata.json                          # Feed metadata
  ├── pacific-silicon-foundry/               # Foundry company
  │   └── documents/
  │       └── wafer-batch-001.json           # JSON documents
  ├── apex-semiconductor-corp/               # IDM company
  └── quantum-chip-design/                   # Fabless company`,
		RunE: runFeedGenerate,
	}

	return cmd
}

// NewFeedSignCommand creates the 'feed sign' subcommand
func NewFeedSignCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign [feed-directory]",
		Short: "Sign all documents in a feed",
		Long: `Sign all JSON documents in a feed directory with ES256 keys.

This command:
  1. Generates ES256 key pairs for each company (if not present)
  2. Signs all JSON documents with COSE Sign1
  3. Outputs .cbor files for each signed document

Examples:
  # Sign all documents in a feed
  scitt feed sign feed-2025-10-18-143022

  # After signing, register to one or more services
  scitt feed register feed-2025-10-18-143022 --service-url http://localhost:8000`,
		Args: cobra.ExactArgs(1),
		RunE: runFeedSign,
	}

	return cmd
}

// NewFeedRegisterCommand creates the 'feed register' subcommand
func NewFeedRegisterCommand() *cobra.Command {
	var serviceURL string
	var apiKey string
	var sampleReceipts int

	cmd := &cobra.Command{
		Use:   "register [feed-directory]",
		Short: "Register signed statements to a SCITT service",
		Long: `Register all signed statements (.cbor files) in a feed to a SCITT transparency service.

This command:
  1. Validates the SCITT service is accessible
  2. Registers all .cbor statements for each company with API key authentication
  3. Optionally saves sample receipts (controlled by --sample-receipts flag)

You can run this command multiple times with different service URLs to compare tile generation across services.

Examples:
  # Register to local service with API key (no receipts)
  scitt feed register feed-2025-10-18-143022 --service-url http://localhost:8000 --api-key YOUR_API_KEY

  # Register and save first 5 receipts per company
  scitt feed register feed-2025-10-18-143022 --service-url http://localhost:8000 --api-key YOUR_API_KEY --sample-receipts 5

  # Register same feed to another service for comparison
  scitt feed register feed-2025-10-18-143022 --service-url http://localhost:9000 --api-key OTHER_API_KEY`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFeedRegister(cmd, args, serviceURL, apiKey, sampleReceipts)
		},
	}

	cmd.Flags().StringVar(&serviceURL, "service-url", "http://127.0.0.1:56177", "SCITT service URL for registration")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication (required)")
	cmd.Flags().IntVar(&sampleReceipts, "sample-receipts", 0, "Number of sample receipts to download per company (0 = none)")
	cmd.MarkFlagRequired("api-key")

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

		fmt.Printf("       Wrote %d documents\n", len(allDocs))
		totalDocs += len(allDocs)
	}

	fmt.Println()
	fmt.Printf("[3/3] Feed generated successfully!\n")
	fmt.Printf("   Location: %s\n", feedDirName)
	fmt.Printf("   Total documents: %d\n", totalDocs)
	fmt.Printf("   Timestamp: %s\n", timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Sign documents:    scitt feed sign %s\n", feedDirName)
	fmt.Printf("  2. Register to SCITT: scitt feed register %s --service-url http://localhost:8000 --api-key YOUR_API_KEY\n", feedDirName)
	fmt.Println()

	return nil
}

// runFeedSign executes the signing workflow for a feed
func runFeedSign(cmd *cobra.Command, args []string) error {
	feedDir := args[0]

	// Verify feed directory exists
	if _, err := os.Stat(feedDir); os.IsNotExist(err) {
		return fmt.Errorf("feed directory not found: %s", feedDir)
	}

	// Read metadata to get company list
	companies, err := readFeedCompanies(feedDir)
	if err != nil {
		return fmt.Errorf("failed to read feed companies: %w", err)
	}

	fmt.Printf("Signing Feed: %s\n", filepath.Base(feedDir))
	fmt.Println()

	// Generate keys for all companies
	fmt.Println("[1/2] Generating ES256 keys for companies...")
	if err := generateCompanyKeys(feedDir, companies); err != nil {
		return fmt.Errorf("key generation failed: %w", err)
	}

	fmt.Println()
	fmt.Println("[2/2] Signing documents for companies...")
	fmt.Println()

	// Sign documents for each company
	totalSigned := 0
	for i, company := range companies {
		fmt.Printf("   [%d/%d] Signing documents for %s...\n", i+1, len(companies), company.Name)
		count, err := signDocumentsForCompany(feedDir, company)
		if err != nil {
			return fmt.Errorf("signing failed for %s: %w", company.Name, err)
		}
		fmt.Printf("      Signed %d documents successfully!\n", count)
		totalSigned += count
	}

	fmt.Println()
	fmt.Printf("All documents signed successfully! (Total: %d)\n", totalSigned)
	fmt.Println()
	fmt.Println("Next step:")
	fmt.Printf("  Register to SCITT: scitt feed register %s --service-url http://localhost:8000 --api-key YOUR_API_KEY\n", filepath.Base(feedDir))
	fmt.Println()

	return nil
}

// runFeedRegister executes the registration workflow for a feed
func runFeedRegister(cmd *cobra.Command, args []string, serviceURL string, apiKey string, sampleReceipts int) error {
	feedDir := args[0]

	// Verify feed directory exists
	if _, err := os.Stat(feedDir); os.IsNotExist(err) {
		return fmt.Errorf("feed directory not found: %s", feedDir)
	}

	// Read metadata to get company list
	companies, err := readFeedCompanies(feedDir)
	if err != nil {
		return fmt.Errorf("failed to read feed companies: %w", err)
	}

	fmt.Printf("Registering Feed: %s\n", filepath.Base(feedDir))
	fmt.Printf("Service URL: %s\n", serviceURL)
	if sampleReceipts > 0 {
		fmt.Printf("Sample Receipts: %d per company\n", sampleReceipts)
	} else {
		fmt.Println("Sample Receipts: none (receipts not saved)")
	}
	fmt.Println()

	// Validate service connection
	fmt.Println("[1/2] Validating connection to SCITT service...")
	if err := validateServiceConnection(serviceURL); err != nil {
		return fmt.Errorf("service validation failed: %w\nPlease check that the service is running and accessible", err)
	}
	fmt.Println("   Service is reachable")

	fmt.Println()
	fmt.Println("[2/2] Registering statements for companies...")
	fmt.Println()

	// Register statements for each company
	totalRegistered := 0
	for i, company := range companies {
		fmt.Printf("   [%d/%d] Registering statements for %s...\n", i+1, len(companies), company.Name)
		count, err := registerStatementsForCompany(feedDir, company, serviceURL, apiKey, sampleReceipts)
		if err != nil {
			return fmt.Errorf("registration failed for %s: %w", company.Name, err)
		}
		fmt.Printf("      Registered %d statements successfully!\n", count)
		totalRegistered += count
	}

	fmt.Println()
	fmt.Printf("All statements registered successfully! (Total: %d)\n", totalRegistered)
	if sampleReceipts > 0 {
		fmt.Printf("Sample receipts saved to company documents directories (.receipt.cbor files)\n")
	}
	fmt.Println()

	return nil
}

// readFeedCompanies reads the metadata.json file and returns the list of companies
func readFeedCompanies(feedDir string) ([]generator.Company, error) {
	metadataPath := filepath.Join(feedDir, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata.json: %w", err)
	}

	var metadata struct {
		Companies []string `json:"companies"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata.json: %w", err)
	}

	// Generate company structs from metadata
	companies := generator.GenerateCompanies()

	return companies, nil
}

// generateCompanyKeys generates ES256 key pairs for each company
func generateCompanyKeys(feedDir string, companies []generator.Company) error {
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

		// Skip if keys already exist
		if _, err := os.Stat(privateKeyPath); err == nil {
			fmt.Printf("      Keys already exist, skipping\n")
			continue
		}

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
func signDocumentsForCompany(feedDir string, company generator.Company) (int, error) {
	companyDir := filepath.Join(feedDir, company.Directory)
	documentsDir := filepath.Join(companyDir, "documents")
	privateKeyPath := filepath.Join(companyDir, "private_key.cbor")

	// Check if private key exists
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("private key not found for %s: %s", company.Name, privateKeyPath)
	}

	// Get all JSON files
	entries, err := os.ReadDir(documentsDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read documents directory for %s: %w", company.Name, err)
	}

	var jsonFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			jsonFiles = append(jsonFiles, entry.Name())
		}
	}

	if len(jsonFiles) == 0 {
		return 0, fmt.Errorf("no JSON documents found for %s", company.Name)
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

		// Skip if already signed
		if _, err := os.Stat(cborPath); err == nil {
			signedCount++
			bar.Add(1)
			continue
		}

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
		return 0, fmt.Errorf("failed to sign any documents for %s", company.Name)
	}

	return signedCount, nil
}

// validateServiceConnection checks if the SCITT service is reachable
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

// registerStatementsForCompany registers all signed statements for a company
func registerStatementsForCompany(feedDir string, company generator.Company, serviceURL string, apiKey string, sampleReceipts int) (int, error) {
	companyDir := filepath.Join(feedDir, company.Directory)
	documentsDir := filepath.Join(companyDir, "documents")

	// Get all CBOR files (signed statements)
	entries, err := os.ReadDir(documentsDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read documents directory for %s: %w", company.Name, err)
	}

	var cborFiles []string
	for _, entry := range entries {
		name := entry.Name()
		// Include .cbor files but exclude .receipt.cbor files
		if !entry.IsDir() && filepath.Ext(name) == ".cbor" && !strings.HasSuffix(name, ".receipt.cbor") {
			cborFiles = append(cborFiles, name)
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
	for idx, cborFile := range cborFiles {
		cborPath := filepath.Join(documentsDir, cborFile)

		// Only download receipt for first N statements if sampleReceipts > 0
		shouldDownloadReceipt := sampleReceipts > 0 && idx < sampleReceipts

		if shouldDownloadReceipt {
			// Derive receipt path (replace .cbor with .receipt.cbor)
			receiptFile := cborFile[:len(cborFile)-5] + ".receipt.cbor"
			receiptPath := filepath.Join(documentsDir, receiptFile)

			// Execute: scitt statement register with receipt
			cmd := exec.Command(scittPath, "statement", "register",
				"--statement", cborPath,
				"--receipt", receiptPath,
				"--service", serviceURL,
				"--api-key", apiKey)

			output, err := cmd.CombinedOutput()
			if err != nil {
				// Log error but continue with remaining statements
				fmt.Printf("\n   Warning: Failed to register %s: %v\n   Output: %s\n", cborFile, err, string(output))
				continue
			}
		} else {
			// Register without downloading receipt (use /dev/null)
			cmd := exec.Command(scittPath, "statement", "register",
				"--statement", cborPath,
				"--receipt", "/dev/null",
				"--service", serviceURL,
				"--api-key", apiKey)

			output, err := cmd.CombinedOutput()
			if err != nil {
				// Log error but continue with remaining statements
				fmt.Printf("\n   Warning: Failed to register %s: %v\n   Output: %s\n", cborFile, err, string(output))
				continue
			}
		}

		registeredCount++
		bar.Add(1)
	}

	if registeredCount == 0 {
		return 0, fmt.Errorf("failed to register any statements for %s", company.Name)
	}

	return registeredCount, nil
}
