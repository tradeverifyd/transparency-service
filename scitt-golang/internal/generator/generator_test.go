package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// T003: Implement Seeded PRNG

func TestSeededRandDeterminism(t *testing.T) {
	t.Parallel()
	timestamp := time.Date(2025, 10, 18, 14, 30, 22, 0, time.UTC)

	rng1 := NewSeededRand(timestamp)
	rng2 := NewSeededRand(timestamp)

	// Same seed should produce identical sequences
	for i := 0; i < 100; i++ {
		val1 := rng1.IntRange(1, 1000)
		val2 := rng2.IntRange(1, 1000)
		assert.Equal(t, val1, val2, "Values should match at iteration %d", i)
	}
}

func TestSeededRandRange(t *testing.T) {
	t.Parallel()
	timestamp := time.Now()
	rng := NewSeededRand(timestamp)

	for i := 0; i < 1000; i++ {
		val := rng.IntRange(8, 12)
		assert.GreaterOrEqual(t, val, 8)
		assert.LessOrEqual(t, val, 12)
	}
}

func TestSeededRandChoose(t *testing.T) {
	t.Parallel()
	timestamp := time.Now()
	rng := NewSeededRand(timestamp)

	options := []string{"foundry", "IDM", "fabless"}
	chosen := rng.Choose(options)
	assert.Contains(t, options, chosen)
}

func TestSeededRandChooseEmpty(t *testing.T) {
	t.Parallel()
	timestamp := time.Now()
	rng := NewSeededRand(timestamp)

	options := []string{}
	chosen := rng.Choose(options)
	assert.Equal(t, "", chosen)
}

// T004: Define Company Identities

func TestCompanyGeneration(t *testing.T) {
	t.Parallel()
	companies := GenerateCompanies()

	assert.Len(t, companies, 3, "Should generate 3 companies")

	// Verify roles
	roles := make(map[string]bool)
	for _, c := range companies {
		roles[c.Role] = true
	}
	assert.True(t, roles["foundry"])
	assert.True(t, roles["IDM"])
	assert.True(t, roles["fabless"])
}

func TestCompanyURNFormat(t *testing.T) {
	t.Parallel()
	companies := GenerateCompanies()

	for _, c := range companies {
		assert.True(t, strings.HasPrefix(c.URN, "urn:supply-chain:"),
			"URN should start with urn:supply-chain: prefix")
		assert.NotContains(t, c.URN, " ", "URN should not contain spaces")
	}
}

func TestCompanyNames(t *testing.T) {
	t.Parallel()
	companies := GenerateCompanies()
	names := []string{}
	for _, c := range companies {
		names = append(names, c.Name)
	}

	assert.Contains(t, names, "Pacific Silicon Foundry")
	assert.Contains(t, names, "Apex Semiconductor Corp")
	assert.Contains(t, names, "Quantum Chip Design")
}

// T005: Implement URN Generation Utilities

func TestGenerateIssuerURN(t *testing.T) {
	t.Parallel()
	urn := GenerateIssuerURN("pacific-silicon-foundry")
	assert.Equal(t, "urn:supply-chain:pacific-silicon-foundry", urn)
}

func TestGenerateSubjectURN(t *testing.T) {
	t.Parallel()
	urn := GenerateSubjectURN("apex-semiconductor-corp", "cpu", "APX-9700K")
	assert.Equal(t, "urn:supply-chain:apex-semiconductor-corp:cpu:APX-9700K", urn)
}

func TestGenerateContentLocation(t *testing.T) {
	t.Parallel()
	url := GenerateContentLocation("quantum-chip-design", "memory", "QCD-DDR5-32GB")
	expected := "https://quantum-chip-design.example/supply-chain/memory/QCD-DDR5-32GB.json"
	assert.Equal(t, expected, url)
}

func TestToSlug(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"Pacific Silicon Foundry", "pacific-silicon-foundry"},
		{"Apex Semiconductor Corp", "apex-semiconductor-corp"},
		{"Quantum Chip Design", "quantum-chip-design"},
	}

	for _, tt := range tests {
		slug := ToSlug(tt.input)
		assert.Equal(t, tt.expected, slug)
	}
}

// T006: Define Document Schemas

func TestDocumentSchemaSerialization(t *testing.T) {
	t.Parallel()

	wafer := &WaferBatch{
		BaseDocument: BaseDocument{
			DocumentID:      "wafer-batch-001",
			DocumentType:    "wafer_batch",
			URN:             "urn:supply-chain:pacific-silicon-foundry:wafer-batch:WF-2025-1001",
			ContentType:     "application/json",
			ContentLocation: "https://example.com/wafer.json",
		},
		LotNumber:       "WF-2025-1001",
		WaferDiameterMM: 300,
	}

	// Verify it implements Document interface
	var _ Document = wafer
	assert.Equal(t, wafer.URN, wafer.GetSubjectURN())
	assert.Equal(t, "application/json", wafer.GetContentType())
}

func TestAllDocumentTypesImplementInterface(t *testing.T) {
	t.Parallel()

	// Verify all document types implement Document interface
	var _ Document = &WaferBatch{}
	var _ Document = &MineralSourcing{}
	var _ Document = &ChipSpecification{}
	var _ Document = &FirmwareManifest{}
	var _ Document = &SBOM{}
	var _ Document = &MemorySpecification{}
	var _ Document = &AITrainingDataset{}
	var _ Document = &AIModelSpecification{}
	var _ Document = &CVEDocument{}
	var _ Document = &LogisticsTracking{}
}

// T007-T016: Test Document Generators

func TestGenerateWaferBatches(t *testing.T) {
	t.Parallel()
	company := GenerateCompanies()[0]
	timestamp := time.Date(2025, 10, 18, 14, 30, 22, 0, time.UTC)
	rng := NewSeededRand(timestamp)

	wafers := GenerateWaferBatches(company, rng)

	assert.GreaterOrEqual(t, len(wafers), 8, "Should generate at least 8 wafers")
	assert.LessOrEqual(t, len(wafers), 12, "Should generate at most 12 wafers")

	for _, w := range wafers {
		assert.Equal(t, "wafer_batch", w.DocumentType)
		assert.Equal(t, 300, w.WaferDiameterMM)
		assert.Equal(t, "silicon", w.Material)
		assert.Contains(t, w.URN, company.Directory)
	}
}

func TestGenerateMineralSourcing(t *testing.T) {
	t.Parallel()
	company := GenerateCompanies()[0]
	timestamp := time.Date(2025, 10, 18, 14, 30, 22, 0, time.UTC)
	rng := NewSeededRand(timestamp)

	minerals := GenerateMineralSourcing(company, rng)

	assert.GreaterOrEqual(t, len(minerals), 6, "Should generate at least 6 minerals")
	assert.LessOrEqual(t, len(minerals), 8, "Should generate at most 8 minerals")

	for _, m := range minerals {
		assert.Equal(t, "mineral_sourcing", m.DocumentType)
		assert.True(t, m.ConflictFree)
		assert.Equal(t, "RMI-Compliant", m.Certification)
		assert.Contains(t, []string{"tantalum", "tin", "tungsten", "gold"}, m.MineralType)
	}
}

func TestGenerateChipSpecifications(t *testing.T) {
	t.Parallel()
	company := GenerateCompanies()[0]
	timestamp := time.Date(2025, 10, 18, 14, 30, 22, 0, time.UTC)
	rng := NewSeededRand(timestamp)

	chips := GenerateChipSpecifications(company, rng)

	assert.GreaterOrEqual(t, len(chips), 10, "Should generate at least 10 chips")
	assert.LessOrEqual(t, len(chips), 14, "Should generate at most 14 chips")

	for _, c := range chips {
		assert.Equal(t, "chip_specification", c.DocumentType)
		assert.Contains(t, []int{5, 7, 10}, c.ProcessNodeNM)
		assert.Contains(t, []string{"CPU", "GPU", "NPU", "Memory Controller", "I/O Controller"}, c.ChipType)

		// CPU-specific fields
		if c.ChipType == "CPU" {
			assert.GreaterOrEqual(t, c.Cores, 8)
			assert.LessOrEqual(t, c.Cores, 16)
			assert.Equal(t, c.Cores*2, c.Threads)
		}
	}
}

func TestGenerateFirmwareManifests(t *testing.T) {
	t.Parallel()
	company := GenerateCompanies()[0]
	timestamp := time.Date(2025, 10, 18, 14, 30, 22, 0, time.UTC)
	rng := NewSeededRand(timestamp)

	firmwares := GenerateFirmwareManifests(company, rng)

	assert.GreaterOrEqual(t, len(firmwares), 8, "Should generate at least 8 firmwares")
	assert.LessOrEqual(t, len(firmwares), 10, "Should generate at most 10 firmwares")

	for _, f := range firmwares {
		assert.Equal(t, "firmware_manifest", f.DocumentType)
		assert.Contains(t, []string{"UEFI", "BIOS", "NPU Firmware", "GPU Firmware", "Microcode"}, f.FirmwareType)
		assert.Len(t, f.SHA256, 64, "SHA256 should be 64 hex characters")
		assert.GreaterOrEqual(t, f.FileSizeBytes, int64(1024*1024))
	}
}

func TestGenerateSBOMs(t *testing.T) {
	t.Parallel()
	company := GenerateCompanies()[0]
	timestamp := time.Date(2025, 10, 18, 14, 30, 22, 0, time.UTC)
	rng := NewSeededRand(timestamp)

	sboms := GenerateSBOMs(company, rng)

	assert.GreaterOrEqual(t, len(sboms), 12, "Should generate at least 12 SBOMs")
	assert.LessOrEqual(t, len(sboms), 16, "Should generate at most 16 SBOMs")

	for _, s := range sboms {
		assert.Equal(t, "SPDX-2.3", s.SPDXVersion)
		assert.Equal(t, "CC0-1.0", s.DataLicense)
		assert.Equal(t, "SPDXRef-DOCUMENT", s.SPDXID)
		assert.GreaterOrEqual(t, len(s.Packages), 3)
		assert.LessOrEqual(t, len(s.Packages), 5)
	}
}

func TestGenerateMemorySpecifications(t *testing.T) {
	t.Parallel()
	company := GenerateCompanies()[0]
	timestamp := time.Date(2025, 10, 18, 14, 30, 22, 0, time.UTC)
	rng := NewSeededRand(timestamp)

	memories := GenerateMemorySpecifications(company, rng)

	assert.GreaterOrEqual(t, len(memories), 6, "Should generate at least 6 memory specs")
	assert.LessOrEqual(t, len(memories), 8, "Should generate at most 8 memory specs")

	for _, m := range memories {
		assert.Equal(t, "memory_specification", m.DocumentType)
		assert.Equal(t, "DDR5", m.MemoryType)
		assert.Contains(t, []int{8, 16, 32}, m.CapacityGB)
		assert.Equal(t, 1.1, m.Voltage)
	}
}

func TestGenerateAITrainingDatasets(t *testing.T) {
	t.Parallel()
	company := GenerateCompanies()[0]
	timestamp := time.Date(2025, 10, 18, 14, 30, 22, 0, time.UTC)
	rng := NewSeededRand(timestamp)

	datasets := GenerateAITrainingDatasets(company, rng)

	assert.GreaterOrEqual(t, len(datasets), 4, "Should generate at least 4 datasets")
	assert.LessOrEqual(t, len(datasets), 6, "Should generate at most 6 datasets")

	for _, d := range datasets {
		assert.Equal(t, "ai_training_dataset", d.DocumentType)
		assert.Contains(t, []string{"image", "text", "multimodal"}, d.Modality)
		assert.GreaterOrEqual(t, d.SizeGB, 50)
		assert.LessOrEqual(t, d.SizeGB, 500)
	}
}

func TestGenerateAIModelSpecifications(t *testing.T) {
	t.Parallel()
	company := GenerateCompanies()[0]
	timestamp := time.Date(2025, 10, 18, 14, 30, 22, 0, time.UTC)
	rng := NewSeededRand(timestamp)

	models := GenerateAIModelSpecifications(company, rng)

	assert.GreaterOrEqual(t, len(models), 3, "Should generate at least 3 models")
	assert.LessOrEqual(t, len(models), 5, "Should generate at most 5 models")

	for _, m := range models {
		assert.Equal(t, "ai_model_specification", m.DocumentType)
		assert.Contains(t, []string{"Transformer", "CNN", "LSTM", "Diffusion", "GAN"}, m.Architecture)
		assert.Equal(t, "NPU", m.TargetHardware)
		assert.GreaterOrEqual(t, m.ParametersMillions, 100)
	}
}

func TestGenerateCVEDocuments(t *testing.T) {
	t.Parallel()
	company := GenerateCompanies()[0]
	timestamp := time.Date(2025, 10, 18, 14, 30, 22, 0, time.UTC)
	rng := NewSeededRand(timestamp)

	cves := GenerateCVEDocuments(company, rng)

	assert.GreaterOrEqual(t, len(cves), 5, "Should generate at least 5 CVEs")
	assert.LessOrEqual(t, len(cves), 7, "Should generate at most 7 CVEs")

	realCVEs := []string{"CVE-2024-0519", "CVE-2024-3660", "CVE-2024-5480", "CVE-2024-22476", "CVE-2024-35198", "CVE-2024-35199"}

	for _, c := range cves {
		assert.Equal(t, "vulnerability_disclosure", c.DocumentType)
		assert.Contains(t, realCVEs, c.VulnerabilityID, "Should use real CVE IDs")
		assert.Contains(t, []string{"HIGH", "CRITICAL"}, c.Severity)
		assert.GreaterOrEqual(t, c.CVSSScore, 7.5)
	}
}

func TestGenerateLogisticsTracking(t *testing.T) {
	t.Parallel()
	company := GenerateCompanies()[0]
	timestamp := time.Date(2025, 10, 18, 14, 30, 22, 0, time.UTC)
	rng := NewSeededRand(timestamp)

	logistics := GenerateLogisticsTracking(company, rng)

	assert.GreaterOrEqual(t, len(logistics), 8, "Should generate at least 8 logistics")
	assert.LessOrEqual(t, len(logistics), 12, "Should generate at most 12 logistics")

	for _, l := range logistics {
		assert.Equal(t, "logistics_tracking", l.DocumentType)
		assert.Contains(t, []string{"in_transit", "delivered", "customs", "departed"}, l.Status)
		assert.NotEmpty(t, l.ShipmentID)
		assert.NotEmpty(t, l.Origin)
		assert.NotEmpty(t, l.Destination)
	}
}

// T017: Test Feed Directory Creation

func TestCreateFeedDirectory(t *testing.T) {
	t.Parallel()

	feedDir, timestamp, err := CreateFeedDirectory(t.TempDir())
	assert.NoError(t, err)

	assert.True(t, strings.HasPrefix(filepath.Base(feedDir), "feed-"))
	assert.NotZero(t, timestamp)

	// Verify directory exists
	info, err := os.Stat(feedDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestFeedDirectoryStructure(t *testing.T) {
	t.Parallel()

	feedDir, _, err := CreateFeedDirectory(t.TempDir())
	assert.NoError(t, err)

	companies := []string{"pacific-silicon-foundry", "apex-semiconductor-corp", "quantum-chip-design"}
	for _, company := range companies {
		companyDir := filepath.Join(feedDir, company)
		assert.DirExists(t, companyDir)

		documentsDir := filepath.Join(companyDir, "documents")
		assert.DirExists(t, documentsDir)
	}
}

func TestFeedMetadataJSON(t *testing.T) {
	t.Parallel()

	feedDir, timestamp, err := CreateFeedDirectory(t.TempDir())
	assert.NoError(t, err)

	metadataPath := filepath.Join(feedDir, "metadata.json")
	assert.FileExists(t, metadataPath)

	var metadata map[string]interface{}
	data, err := os.ReadFile(metadataPath)
	assert.NoError(t, err)

	err = json.Unmarshal(data, &metadata)
	assert.NoError(t, err)

	assert.Contains(t, metadata, "timestamp")
	assert.Contains(t, metadata, "seed")
	assert.Contains(t, metadata, "companies")

	// Verify timestamp matches
	assert.Equal(t, timestamp.Format(time.RFC3339), metadata["timestamp"])
}

// T018: Test Document Serialization

func TestWriteDocumentToFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	wafer := &WaferBatch{
		BaseDocument: BaseDocument{
			DocumentID:      "wafer-batch-001",
			DocumentType:    "wafer_batch",
			URN:             "urn:supply-chain:test:wafer-batch:WF-2025-1001",
			ContentType:     "application/json",
			ContentLocation: "https://test.example/wafer.json",
		},
		LotNumber:       "WF-2025-1001",
		WaferDiameterMM: 300,
	}

	err := WriteDocumentToFile(wafer, dir)
	assert.NoError(t, err)

	// Verify file exists
	expectedPath := filepath.Join(dir, "wafer-batch-001.json")
	assert.FileExists(t, expectedPath)

	// Verify content
	data, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)

	var decoded WaferBatch
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, wafer.LotNumber, decoded.LotNumber)
	assert.Equal(t, wafer.URN, decoded.URN)
}

func TestDocumentJSONFormatting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	chip := &ChipSpecification{
		BaseDocument: BaseDocument{
			DocumentID:   "chip-spec-001",
			DocumentType: "chip_specification",
			URN:          "urn:test:chip:APX-9700K",
		},
		PartNumber: "APX-9700K",
		ChipType:   "CPU",
	}

	err := WriteDocumentToFile(chip, dir)
	assert.NoError(t, err)

	files, err := os.ReadDir(dir)
	assert.NoError(t, err)
	assert.Len(t, files, 1)

	data, err := os.ReadFile(filepath.Join(dir, files[0].Name()))
	assert.NoError(t, err)

	// Verify pretty-printed (contains newlines)
	assert.Contains(t, string(data), "\n")
	assert.Contains(t, string(data), "  ") // Check for indentation
}

func TestWriteMultipleDocuments(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	docs := []Document{
		&WaferBatch{
			BaseDocument: BaseDocument{
				DocumentID:   "wafer-batch-001",
				DocumentType: "wafer_batch",
			},
		},
		&ChipSpecification{
			BaseDocument: BaseDocument{
				DocumentID:   "chip-spec-001",
				DocumentType: "chip_specification",
			},
		},
		&MemorySpecification{
			BaseDocument: BaseDocument{
				DocumentID:   "memory-001",
				DocumentType: "memory_specification",
			},
		},
	}

	for _, doc := range docs {
		err := WriteDocumentToFile(doc, dir)
		assert.NoError(t, err)
	}

	files, err := os.ReadDir(dir)
	assert.NoError(t, err)
	assert.Len(t, files, 3, "Should create 3 JSON files")
}
