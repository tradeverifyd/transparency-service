package generator

import (
	"fmt"
	"time"
)

// T007: Generate Wafer Batch Documents
func GenerateWaferBatches(company Company, rng *SeededRand) []WaferBatch {
	count := rng.IntRange(8, 12)
	wafers := make([]WaferBatch, count)
	
	for i := 0; i < count; i++ {
		lotID := fmt.Sprintf("WF-2025-%04d", 1001+i)
		timestamp := GetTimestamp()
		
		wafers[i] = WaferBatch{
			BaseDocument: BaseDocument{
				DocumentID:      fmt.Sprintf("wafer-batch-%03d", i+1),
				DocumentType:    "wafer_batch",
				URN:             GenerateSubjectURN(company.Directory, "wafer-batch", lotID),
				Issuer:          company.URN,
				Timestamp:       timestamp,
				ContentType:     "application/json",
				ContentLocation: GenerateContentLocation(company.Directory, "wafer-batch", lotID),
			},
			LotNumber:          lotID,
			WaferDiameterMM:    300,
			ThicknessUM:        775,
			Material:           "silicon",
			CrystalOrientation: "100",
			ResistivityOhmCM:   float64(rng.IntRange(1, 20)),
			ProducedDate:       time.Now().AddDate(0, 0, -rng.IntRange(1, 30)).Format("2006-01-02"),
		}
	}
	
	return wafers
}

// T008: Generate Mineral Sourcing Documents
func GenerateMineralSourcing(company Company, rng *SeededRand) []MineralSourcing {
	count := rng.IntRange(6, 8)
	minerals := make([]MineralSourcing, count)
	
	mineralTypes := []string{"tantalum", "tin", "tungsten", "gold"}
	countries := []string{"Rwanda", "Democratic Republic of Congo", "Australia", "Peru"}
	mines := []string{"Rutongo Mine", "Bisie Mine", "Wodgina Mine", "Yanacocha Mine"}
	
	for i := 0; i < count; i++ {
		mineralType := mineralTypes[i%len(mineralTypes)]
		mineralID := fmt.Sprintf("MS-2025-%03d", i+1)
		timestamp := GetTimestamp()
		
		minerals[i] = MineralSourcing{
			BaseDocument: BaseDocument{
				DocumentID:      fmt.Sprintf("mineral-source-%03d", i+1),
				DocumentType:    "mineral_sourcing",
				URN:             GenerateSubjectURN(company.Directory, "mineral", mineralID),
				Issuer:          company.URN,
				Timestamp:       timestamp,
				ContentType:     "application/json",
				ContentLocation: GenerateContentLocation(company.Directory, "mineral", mineralID),
			},
			MineralType:   mineralType,
			OriginCountry: countries[i%len(countries)],
			MineName:      mines[i%len(mines)],
			Certification: "RMI-Compliant",
			QuantityKG:    float64(rng.IntRange(100, 500)) + rng.Float64(),
			ConflictFree:  true,
		}
	}
	
	return minerals
}

// T009: Generate Chip Specification Documents
func GenerateChipSpecifications(company Company, rng *SeededRand) []ChipSpecification {
	count := rng.IntRange(10, 14)
	chips := make([]ChipSpecification, count)
	
	chipTypes := []string{"CPU", "GPU", "NPU", "Memory Controller", "I/O Controller"}
	
	for i := 0; i < count; i++ {
		partNum := fmt.Sprintf("APX-%04dK", 9700+i)
		chipType := chipTypes[i%len(chipTypes)]
		timestamp := GetTimestamp()
		
		processNodes := []int{5, 7, 10}
		chip := ChipSpecification{
			BaseDocument: BaseDocument{
				DocumentID:      fmt.Sprintf("chip-spec-%03d", i+1),
				DocumentType:    "chip_specification",
				URN:             GenerateSubjectURN(company.Directory, "cpu", partNum),
				Issuer:          company.URN,
				Timestamp:       timestamp,
				ContentType:     "application/json",
				ContentLocation: GenerateContentLocation(company.Directory, "cpu", partNum),
			},
			PartNumber:   partNum,
			ChipType:     chipType,
			ProcessNodeNM: processNodes[rng.Intn(len(processNodes))],
		}
		
		// Add CPU-specific fields
		if chipType == "CPU" {
			chip.Cores = rng.IntRange(8, 16)
			chip.Threads = chip.Cores * 2
			chip.BaseFrequencyGHz = 3.0 + rng.Float64()*0.8
			chip.BoostFrequencyGHz = chip.BaseFrequencyGHz + 1.0 + rng.Float64()*0.5
			chip.TDPW = rng.IntRange(45, 125)
			
			// Some CPUs have NPU
			if rng.IntRange(0, 2) == 0 {
				chip.NPUIncluded = true
				chip.NPUTOPS = float64(rng.IntRange(10, 20))
			}
		}
		
		chips[i] = chip
	}
	
	return chips
}

// T010: Generate Firmware Manifest Documents
func GenerateFirmwareManifests(company Company, rng *SeededRand) []FirmwareManifest {
	count := rng.IntRange(8, 10)
	firmwares := make([]FirmwareManifest, count)
	
	firmwareTypes := []string{"UEFI", "BIOS", "NPU Firmware", "GPU Firmware", "Microcode"}
	
	for i := 0; i < count; i++ {
		version := fmt.Sprintf("2025.%02d.%02d", rng.IntRange(1, 12), rng.IntRange(1, 31))
		fwType := firmwareTypes[i%len(firmwareTypes)]
		fwID := fmt.Sprintf("%s-%s", fwType, version)
		timestamp := GetTimestamp()
		
		// Generate pseudo-random SHA256 (64 hex chars)
		hash := fmt.Sprintf("%064x", rng.Intn(1<<30))
		
		firmwares[i] = FirmwareManifest{
			BaseDocument: BaseDocument{
				DocumentID:      fmt.Sprintf("firmware-%03d", i+1),
				DocumentType:    "firmware_manifest",
				URN:             GenerateSubjectURN(company.Directory, "firmware", fwID),
				Issuer:          company.URN,
				Timestamp:       timestamp,
				ContentType:     "application/json",
				ContentLocation: GenerateContentLocation(company.Directory, "firmware", fwID),
			},
			FirmwareType:     fwType,
			Version:          version,
			SHA256:           hash,
			SigningAuthority: company.Name,
			ReleaseDate:      time.Now().AddDate(0, 0, -rng.IntRange(1, 90)).Format("2006-01-02"),
			FileSizeBytes:    int64(rng.IntRange(1024*1024, 10*1024*1024)), // 1-10 MB
		}
	}
	
	return firmwares
}

// T011: Generate SBOM Documents (SPDX 2.3 format)
func GenerateSBOMs(company Company, rng *SeededRand) []SBOM {
	count := rng.IntRange(12, 16)
	sboms := make([]SBOM, count)
	
	componentNames := []string{"CPU", "Memory", "GPU", "NPU", "Firmware", "Chipset", "I/O Controller"}
	
	for i := 0; i < count; i++ {
		docName := fmt.Sprintf("%s-sbom-%03d", company.Directory, i+1)
		timestamp := GetTimestamp()
		
		// Generate 3-5 packages per SBOM
		packageCount := rng.IntRange(3, 5)
		packages := make([]Package, packageCount)
		for j := 0; j < packageCount; j++ {
			compName := componentNames[j%len(componentNames)]
			packages[j] = Package{
				SPDXID:        fmt.Sprintf("SPDXRef-Package-%s", compName),
				Name:          fmt.Sprintf("%s-%s", compName, fmt.Sprintf("v%d.%d", rng.IntRange(1, 3), rng.IntRange(0, 9))),
				VersionInfo:   fmt.Sprintf("1.%d", rng.IntRange(0, 9)),
				Supplier:      fmt.Sprintf("Organization: %s", company.Name),
				FilesAnalyzed: false,
			}
		}
		
		sboms[i] = SBOM{
			SPDXVersion:       "SPDX-2.3",
			DataLicense:       "CC0-1.0",
			SPDXID:            "SPDXRef-DOCUMENT",
			DocumentName:      docName,
			DocumentNamespace: fmt.Sprintf("https://%s.example/sbom/%s", company.Directory, docName),
			CreationInfo: CreationInfo{
				LicenseListVersion: "3.21",
				Creators:           []string{"Tool: scitt-feed-generator"},
				Created:            timestamp,
			},
			Packages: packages,
		}
	}
	
	return sboms
}

// T012: Generate Memory Specification Documents
func GenerateMemorySpecifications(company Company, rng *SeededRand) []MemorySpecification {
	count := rng.IntRange(6, 8)
	memories := make([]MemorySpecification, count)
	
	capacities := []int{8, 16, 32}
	speeds := []int{4800, 5200, 5600, 6000, 6400}
	
	for i := 0; i < count; i++ {
		capacity := capacities[i%len(capacities)]
		speed := speeds[i%len(speeds)]
		partNum := fmt.Sprintf("QCD-DDR5-%dGB", capacity)
		timestamp := GetTimestamp()
		
		memories[i] = MemorySpecification{
			BaseDocument: BaseDocument{
				DocumentID:      fmt.Sprintf("memory-%03d", i+1),
				DocumentType:    "memory_specification",
				URN:             GenerateSubjectURN(company.Directory, "memory", partNum),
				Issuer:          company.URN,
				Timestamp:       timestamp,
				ContentType:     "application/json",
				ContentLocation: GenerateContentLocation(company.Directory, "memory", partNum),
			},
			PartNumber: partNum,
			MemoryType: "DDR5",
			CapacityGB: capacity,
			SpeedMHz:   speed,
			CASLatency: rng.IntRange(36, 42),
			Voltage:    1.1,
		}
	}
	
	return memories
}

// T013: Generate AI Training Dataset Documents
func GenerateAITrainingDatasets(company Company, rng *SeededRand) []AITrainingDataset {
	count := rng.IntRange(4, 6)
	datasets := make([]AITrainingDataset, count)
	
	datasetNames := []string{"ImageNet-2024-Subset", "COCO-2024", "OpenWebText", "Common Crawl", "Wikipedia-Dump"}
	licenses := []string{"CC-BY-4.0", "MIT", "Apache-2.0", "CC0-1.0"}
	modalities := []string{"image", "text", "multimodal"}
	
	for i := 0; i < count; i++ {
		datasetName := datasetNames[i%len(datasetNames)]
		datasetID := fmt.Sprintf("%s-%d", datasetName, 2024+i)
		timestamp := GetTimestamp()
		
		datasets[i] = AITrainingDataset{
			BaseDocument: BaseDocument{
				DocumentID:      fmt.Sprintf("ai-dataset-%03d", i+1),
				DocumentType:    "ai_training_dataset",
				URN:             GenerateSubjectURN(company.Directory, "ai-dataset", datasetID),
				Issuer:          company.URN,
				Timestamp:       timestamp,
				ContentType:     "application/json",
				ContentLocation: GenerateContentLocation(company.Directory, "ai-dataset", datasetID),
			},
			DatasetName:    datasetName,
			Source:         fmt.Sprintf("%s Consortium", datasetName),
			License:        licenses[i%len(licenses)],
			SizeGB:         rng.IntRange(50, 500),
			DataProvenance: "Academic research dataset",
			Usage:          "Pre-training laptop AI models",
			RecordCount:    int64(rng.IntRange(100000, 10000000)),
			Modality:       modalities[i%len(modalities)],
		}
	}
	
	return datasets
}

// T014: Generate AI Model Specification Documents
func GenerateAIModelSpecifications(company Company, rng *SeededRand) []AIModelSpecification {
	count := rng.IntRange(3, 5)
	models := make([]AIModelSpecification, count)
	
	architectures := []string{"Transformer", "CNN", "LSTM", "Diffusion", "GAN"}
	quantizations := []string{"FP32", "FP16", "INT8", "INT4"}
	
	for i := 0; i < count; i++ {
		modelName := fmt.Sprintf("%s-Model-v%d", architectures[i%len(architectures)], i+1)
		modelID := fmt.Sprintf("model-%d", i+1)
		timestamp := GetTimestamp()
		
		models[i] = AIModelSpecification{
			BaseDocument: BaseDocument{
				DocumentID:      fmt.Sprintf("ai-model-%03d", i+1),
				DocumentType:    "ai_model_specification",
				URN:             GenerateSubjectURN(company.Directory, "ai-model", modelID),
				Issuer:          company.URN,
				Timestamp:       timestamp,
				ContentType:     "application/json",
				ContentLocation: GenerateContentLocation(company.Directory, "ai-model", modelID),
			},
			ModelName:          modelName,
			Architecture:       architectures[i%len(architectures)],
			ParametersMillions: rng.IntRange(100, 1000),
			Quantization:       quantizations[i%len(quantizations)],
			InferenceLatencyMS: rng.IntRange(10, 100),
			TargetHardware:     "NPU",
			InferenceTOPS:      float64(rng.IntRange(5, 20)),
		}
	}
	
	return models
}

// T015: Generate CVE/CWE Vulnerability Documents
func GenerateCVEDocuments(company Company, rng *SeededRand) []CVEDocument {
	count := rng.IntRange(5, 7)
	cves := make([]CVEDocument, count)
	
	// Real CVE IDs from research.md
	realCVEs := []struct {
		id          string
		cweID       string
		title       string
		cvss        float64
		severity    string
		component   string
		versions    []string
		patch       string
		description string
	}{
		{"CVE-2024-0519", "CWE-787", "NVIDIA GPU Driver Privilege Escalation", 8.8, "HIGH", "NVIDIA GPU Driver", []string{"535.x", "545.x"}, "550.54.14", "Out-of-bounds write in GPU driver"},
		{"CVE-2024-3660", "CWE-502", "TensorFlow Keras Arbitrary Code Execution", 9.8, "CRITICAL", "TensorFlow Keras", []string{"2.12.x", "2.13.x"}, "2.13.1", "Deserialization of untrusted data in model files"},
		{"CVE-2024-5480", "CWE-502", "PyTorch Distributed RPC Remote Code Execution", 9.8, "CRITICAL", "PyTorch RPC", []string{"2.0.x", "2.1.x"}, "2.1.2", "Unsafe deserialization in distributed RPC"},
		{"CVE-2024-22476", "CWE-94", "Intel Neural Compressor Vulnerability", 10.0, "CRITICAL", "Intel Neural Compressor", []string{"2.3.x", "2.4.x"}, "2.5.0", "Code injection vulnerability"},
		{"CVE-2024-35198", "CWE-20", "NPU Firmware Input Validation", 7.5, "HIGH", "NPU Firmware", []string{"1.0.x"}, "1.1.0", "Improper input validation"},
		{"CVE-2024-35199", "CWE-119", "GPU Memory Corruption", 8.1, "HIGH", "GPU Firmware", []string{"3.2.x"}, "3.3.0", "Buffer overflow in GPU memory management"},
	}
	
	for i := 0; i < count; i++ {
		cveData := realCVEs[i%len(realCVEs)]
		timestamp := GetTimestamp()
		
		cves[i] = CVEDocument{
			BaseDocument: BaseDocument{
				DocumentID:      fmt.Sprintf("cve-%03d", i+1),
				DocumentType:    "vulnerability_disclosure",
				URN:             GenerateSubjectURN(company.Directory, "cve", cveData.id),
				Issuer:          company.URN,
				Timestamp:       timestamp,
				ContentType:     "application/json",
				ContentLocation: GenerateContentLocation(company.Directory, "cve", cveData.id),
			},
			VulnerabilityID:   cveData.id,
			CWEID:             cveData.cweID,
			Title:             cveData.title,
			CVSSScore:         cveData.cvss,
			Severity:          cveData.severity,
			AffectedComponent: cveData.component,
			AffectedVersions:  cveData.versions,
			PatchedVersion:    cveData.patch,
			DisclosureDate:    time.Now().AddDate(0, -rng.IntRange(1, 6), 0).Format("2006-01-02"),
			Description:       cveData.description,
		}
	}
	
	return cves
}

// T016: Generate Logistics Tracking Documents
func GenerateLogisticsTracking(company Company, rng *SeededRand) []LogisticsTracking {
	count := rng.IntRange(8, 12)
	logistics := make([]LogisticsTracking, count)
	
	origins := []string{"Taiwan Fab", "South Korea Fab", "Arizona Fab", "Ireland Fab"}
	destinations := []string{"Assembly Plant - Vietnam", "Assembly Plant - Malaysia", "Distribution Center - US", "Distribution Center - EU"}
	statuses := []string{"in_transit", "delivered", "customs", "departed"}
	
	for i := 0; i < count; i++ {
		shipmentID := fmt.Sprintf("SHIP-2025-%04d", 1001+i)
		timestamp := GetTimestamp()
		departDays := rng.IntRange(5, 30)
		
		logistics[i] = LogisticsTracking{
			BaseDocument: BaseDocument{
				DocumentID:      fmt.Sprintf("logistics-%03d", i+1),
				DocumentType:    "logistics_tracking",
				URN:             GenerateSubjectURN(company.Directory, "shipment", shipmentID),
				Issuer:          company.URN,
				Timestamp:       timestamp,
				ContentType:     "application/json",
				ContentLocation: GenerateContentLocation(company.Directory, "shipment", shipmentID),
			},
			ShipmentID:    shipmentID,
			Origin:        origins[i%len(origins)],
			Destination:   destinations[i%len(destinations)],
			DepartureDate: time.Now().AddDate(0, 0, -departDays).Format("2006-01-02"),
			ArrivalDate:   time.Now().AddDate(0, 0, -departDays+rng.IntRange(2, 7)).Format("2006-01-02"),
			Contents:      fmt.Sprintf("Wafer batch WF-2025-%04d", 1001+i),
			Status:        statuses[i%len(statuses)],
		}
	}
	
	return logistics
}
