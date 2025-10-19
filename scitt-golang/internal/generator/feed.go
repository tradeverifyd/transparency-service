package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FeedMetadata represents the metadata.json file structure
type FeedMetadata struct {
	Timestamp string   `json:"timestamp"`
	Seed      int64    `json:"seed"`
	Companies []string `json:"companies"`
}

// CreateFeedDirectory creates a timestamped feed directory with company subdirectories
// Returns: (feedDirPath, timestamp, error)
func CreateFeedDirectory(baseDir string) (string, time.Time, error) {
	timestamp := time.Now()
	return CreateFeedDirectoryWithTimestamp(baseDir, timestamp)
}

// CreateFeedDirectoryWithTimestamp creates a feed directory with a specific timestamp
// Useful for testing and handling collisions
func CreateFeedDirectoryWithTimestamp(baseDir string, timestamp time.Time) (string, time.Time, error) {
	// Format: feed-YYYY-MM-DD-HHMMSS
	dirName := fmt.Sprintf("feed-%s", timestamp.Format("2006-01-02-150405"))
	feedDir := filepath.Join(baseDir, dirName)

	// Handle directory collision by appending suffix
	originalDir := feedDir
	suffix := 1
	for {
		_, err := os.Stat(feedDir)
		if os.IsNotExist(err) {
			break
		}
		feedDir = fmt.Sprintf("%s-%d", originalDir, suffix)
		suffix++
	}

	// Create feed directory
	if err := os.MkdirAll(feedDir, 0755); err != nil {
		return "", timestamp, fmt.Errorf("failed to create feed directory: %w", err)
	}

	// Get companies
	companies := GenerateCompanies()

	// Create company subdirectories
	for _, company := range companies {
		companyDir := filepath.Join(feedDir, company.Directory)
		if err := os.MkdirAll(companyDir, 0755); err != nil {
			return "", timestamp, fmt.Errorf("failed to create company directory %s: %w", company.Directory, err)
		}

		// Create documents subdirectory
		documentsDir := filepath.Join(companyDir, "documents")
		if err := os.MkdirAll(documentsDir, 0755); err != nil {
			return "", timestamp, fmt.Errorf("failed to create documents directory for %s: %w", company.Directory, err)
		}
	}

	// Create metadata.json
	metadata := FeedMetadata{
		Timestamp: timestamp.Format(time.RFC3339),
		Seed:      timestamp.UnixNano(),
		Companies: []string{
			companies[0].Directory,
			companies[1].Directory,
			companies[2].Directory,
		},
	}

	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", timestamp, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataPath := filepath.Join(feedDir, "metadata.json")
	if err := os.WriteFile(metadataPath, metadataBytes, 0644); err != nil {
		return "", timestamp, fmt.Errorf("failed to write metadata.json: %w", err)
	}

	return feedDir, timestamp, nil
}

// WriteDocumentToFile writes a document to a JSON file with pretty printing
// Filename is based on document_id field in the document
func WriteDocumentToFile(doc Document, dir string) error {
	// Marshal document to pretty-printed JSON
	jsonBytes, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	// Extract document ID to generate filename
	// We need to access the BaseDocument fields through type assertion
	var filename string
	switch d := doc.(type) {
	case *WaferBatch:
		filename = d.DocumentID + ".json"
	case *MineralSourcing:
		filename = d.DocumentID + ".json"
	case *ChipSpecification:
		filename = d.DocumentID + ".json"
	case *FirmwareManifest:
		filename = d.DocumentID + ".json"
	case *SBOM:
		// SBOM uses DocumentName instead of DocumentID
		filename = d.DocumentName + ".json"
	case *MemorySpecification:
		filename = d.DocumentID + ".json"
	case *AITrainingDataset:
		filename = d.DocumentID + ".json"
	case *AIModelSpecification:
		filename = d.DocumentID + ".json"
	case *CVEDocument:
		filename = d.DocumentID + ".json"
	case *LogisticsTracking:
		filename = d.DocumentID + ".json"
	default:
		return fmt.Errorf("unsupported document type")
	}

	// Write to file
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write document file: %w", err)
	}

	return nil
}
