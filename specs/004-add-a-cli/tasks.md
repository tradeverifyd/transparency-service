# Implementation Tasks: Synthetic Supply Chain Feed Generator

**Feature**: 004-add-a-cli
**Branch**: `004-add-a-cli`
**Generated**: 2025-10-18
**Status**: Ready for Implementation

This task list follows Test-First Development (Constitution Principle III). All tasks include test requirements.

---

## Task Organization

Tasks are organized by **User Story Priority** (P1 → P2 → P3) with dependencies tracked. Tasks marked `[P]` can be executed in parallel within their phase.

---

## Phase 1: Setup & Prerequisites

### T001: Project Structure Setup
**Type**: Setup
**Priority**: Blocking
**Estimated Effort**: 30 minutes
**Dependencies**: None

**Description**: Create directory structure and package scaffolding for feed generator.

**Tasks**:
1. Create `internal/generator/` package directory
2. Create `internal/cli/feed.go` file for Cobra command
3. Add placeholder files: `companies.go`, `documents.go`, `schema.go`, `urns.go`, `seeded_rand.go`
4. Add test files: `generator_test.go`, `fixtures_test.go`
5. Update `cmd/scitt/main.go` to register `feed` subcommand

**Test Requirements**:
```go
// Test package imports successfully
func TestGeneratorPackageExists(t *testing.T) {
    // Verify internal/generator package compiles
}
```

**Acceptance Criteria**:
- [ ] `internal/generator/` directory exists
- [ ] All placeholder files compile without errors
- [ ] `./scitt feed --help` displays help text (empty command OK)

**Files Modified**:
- `cmd/scitt/main.go`
- `internal/cli/feed.go` (new)
- `internal/generator/*.go` (new)

---

### T002: Add Progress Bar Dependency
**Type**: Setup
**Priority**: Blocking
**Estimated Effort**: 15 minutes
**Dependencies**: None

**Description**: Add `github.com/schollz/progressbar/v3` dependency to project.

**Tasks**:
1. Run `go get github.com/schollz/progressbar/v3`
2. Update `go.mod` and `go.sum`
3. Create helper function `createProgressBar()` in `internal/cli/feed.go`

**Test Requirements**:
```go
func TestProgressBarCreation(t *testing.T) {
    bar := createProgressBar("Test", 100)
    assert.NotNil(t, bar)
}
```

**Acceptance Criteria**:
- [ ] Dependency resolves without errors
- [ ] `go mod tidy` completes successfully
- [ ] Helper function compiles

**Files Modified**:
- `go.mod`, `go.sum`
- `internal/cli/feed.go`

---

## Phase 2: User Story 1 (P1) - Generate Synthetic Supply Chain Dataset

**Goal**: Complete document generation workflow independently testable without signing/registration.

---

### T003: Implement Seeded PRNG [P]
**Type**: Core Logic
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 1 hour
**Dependencies**: T001

**Description**: Implement deterministic random number generator using timestamp seed (per research.md § 3).

**Tasks**:
1. Create `SeededRand` struct with `rand.Rand` wrapper
2. Implement `NewSeededRand(timestamp time.Time)` constructor using `UnixNano()` as seed
3. Implement helper methods: `IntRange(min, max)`, `Choose(options []string)`, `Shuffle(slice []interface{})`
4. **TDD**: Write tests FIRST (test should fail before implementation)

**Test Requirements** (WRITE FIRST):
```go
func TestSeededRandDeterminism(t *testing.T) {
    t.Parallel()
    timestamp := time.Date(2025, 10, 18, 14, 30, 22, 0, time.UTC)

    rng1 := generator.NewSeededRand(timestamp)
    rng2 := generator.NewSeededRand(timestamp)

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
    rng := generator.NewSeededRand(timestamp)

    for i := 0; i < 1000; i++ {
        val := rng.IntRange(8, 12)
        assert.GreaterOrEqual(t, val, 8)
        assert.LessOrEqual(t, val, 12)
    }
}

func TestSeededRandChoose(t *testing.T) {
    t.Parallel()
    timestamp := time.Now()
    rng := generator.NewSeededRand(timestamp)

    options := []string{"foundry", "IDM", "fabless"}
    chosen := rng.Choose(options)
    assert.Contains(t, options, chosen)
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Same timestamp produces identical sequences across multiple runs
- [ ] `IntRange()` respects min/max bounds
- [ ] `Choose()` returns values from provided slice

**Files Modified**:
- `internal/generator/seeded_rand.go`
- `internal/generator/generator_test.go`

---

### T004: Define Company Identities [P]
**Type**: Core Logic
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 45 minutes
**Dependencies**: T001

**Description**: Define structs for 3 company identities with roles and URN generation (per data-model.md § Company Identity).

**Tasks**:
1. Create `Company` struct with fields: Name, Role, URN, Directory
2. Define constants for 3 companies: Pacific Silicon Foundry (foundry), Apex Semiconductor Corp (IDM), Quantum Chip Design (fabless)
3. Implement `GenerateCompanies()` returning slice of Company structs
4. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestCompanyGeneration(t *testing.T) {
    t.Parallel()
    companies := generator.GenerateCompanies()

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
    companies := generator.GenerateCompanies()

    for _, c := range companies {
        assert.True(t, strings.HasPrefix(c.URN, "urn:supply-chain:"),
            "URN should start with urn:supply-chain: prefix")
        assert.NotContains(t, c.URN, " ", "URN should not contain spaces")
    }
}

func TestCompanyNames(t *testing.T) {
    t.Parallel()
    companies := generator.GenerateCompanies()
    names := []string{}
    for _, c := range companies {
        names = append(names, c.Name)
    }

    assert.Contains(t, names, "Pacific Silicon Foundry")
    assert.Contains(t, names, "Apex Semiconductor Corp")
    assert.Contains(t, names, "Quantum Chip Design")
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] 3 companies generated with correct names and roles
- [ ] URNs follow pattern: `urn:supply-chain:{company}`
- [ ] Directory names are lowercase-with-hyphens

**Files Modified**:
- `internal/generator/companies.go`
- `internal/generator/generator_test.go`

---

### T005: Implement URN Generation Utilities [P]
**Type**: Core Logic
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 1 hour
**Dependencies**: T001

**Description**: Create helper functions for generating URN identifiers for documents (per data-model.md § URN Format).

**Tasks**:
1. Implement `GenerateIssuerURN(companySlug string) string`
2. Implement `GenerateSubjectURN(companySlug, docType, docID string) string`
3. Implement `GenerateContentLocation(companySlug, docType, docID string) string`
4. Implement helper `toSlug(name string) string` for name → slug conversion
5. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestGenerateIssuerURN(t *testing.T) {
    t.Parallel()
    urn := generator.GenerateIssuerURN("pacific-silicon-foundry")
    assert.Equal(t, "urn:supply-chain:pacific-silicon-foundry", urn)
}

func TestGenerateSubjectURN(t *testing.T) {
    t.Parallel()
    urn := generator.GenerateSubjectURN("apex-semiconductor-corp", "cpu", "APX-9700K")
    assert.Equal(t, "urn:supply-chain:apex-semiconductor-corp:cpu:APX-9700K", urn)
}

func TestGenerateContentLocation(t *testing.T) {
    t.Parallel()
    url := generator.GenerateContentLocation("quantum-chip-design", "memory", "QCD-DDR5-32GB")
    expected := "https://quantum-chip-design.example/supply-chain/memory/QCD-DDR5-32GB.json"
    assert.Equal(t, expected, url)
}

func TestSlugConversion(t *testing.T) {
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
        slug := generator.ToSlug(tt.input)
        assert.Equal(t, tt.expected, slug)
    }
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] URNs follow RFC 8141 format
- [ ] Content locations use HTTPS and `.json` extension
- [ ] Slug conversion handles spaces, capitals, special chars

**Files Modified**:
- `internal/generator/urns.go`
- `internal/generator/generator_test.go`

---

### T006: Define Document Schemas [P]
**Type**: Core Logic
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 2 hours
**Dependencies**: T001

**Description**: Define Go structs for all 10 document types (per data-model.md § Supply Chain Document Types).

**Tasks**:
1. Create structs for: WaferBatch, MineralSourcing, ChipSpecification, FirmwareManifest, SBOM (SPDX), MemorySpecification, AITrainingDataset, AIModelSpecification, CVEDocument, LogisticsTracking
2. Implement `Document` interface with `GetSubjectURN()`, `GetContentType()` methods
3. Add JSON struct tags for all fields
4. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestDocumentSchemaSerialization(t *testing.T) {
    t.Parallel()

    wafer := &generator.WaferBatch{
        BatchID:     "WF-2025-1001",
        LotNumber:   "LOT-12345",
        Dimensions:  "300mm",
        Material:    "silicon",
        SubjectURN:  "urn:supply-chain:pacific-silicon-foundry:wafer-batch:WF-2025-1001",
    }

    jsonBytes, err := json.Marshal(wafer)
    require.NoError(t, err)

    var decoded generator.WaferBatch
    err = json.Unmarshal(jsonBytes, &decoded)
    require.NoError(t, err)

    assert.Equal(t, wafer.BatchID, decoded.BatchID)
    assert.Equal(t, wafer.SubjectURN, decoded.SubjectURN)
}

func TestSPDXSBOMFormat(t *testing.T) {
    t.Parallel()

    sbom := &generator.SBOM{
        SPDXVersion:       "SPDX-2.3",
        DataLicense:       "CC0-1.0",
        SPDXID:            "SPDXRef-DOCUMENT",
        DocumentName:      "laptop-sbom",
        DocumentNamespace: "https://apex-semiconductor-corp.example/sbom/laptop-2025",
        CreationInfo: generator.CreationInfo{
            Created:  "2025-10-18T14:30:22Z",
            Creators: []string{"Tool: scitt-feed-generator"},
        },
        Packages: []generator.Package{
            {SPDXID: "SPDXRef-Package-CPU", Name: "APX-9700K-CPU"},
        },
    }

    jsonBytes, err := json.Marshal(sbom)
    require.NoError(t, err)

    // Verify SPDX-2.3 required fields present
    assert.Contains(t, string(jsonBytes), "SPDX-2.3")
    assert.Contains(t, string(jsonBytes), "CC0-1.0")
}

func TestAllDocumentTypesImplementInterface(t *testing.T) {
    t.Parallel()

    var _ generator.Document = &generator.WaferBatch{}
    var _ generator.Document = &generator.MineralSourcing{}
    var _ generator.Document = &generator.ChipSpecification{}
    // ... test all 10 types
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] All 10 document types defined as structs
- [ ] SBOM follows SPDX 2.3 minimal schema (per research.md § 1)
- [ ] All structs implement Document interface
- [ ] JSON serialization/deserialization works correctly

**Files Modified**:
- `internal/generator/schema.go`
- `internal/generator/generator_test.go`

---

### T007: Generate Wafer Batch Documents [P]
**Type**: Document Generation
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 1.5 hours
**Dependencies**: T003, T004, T005, T006

**Description**: Implement generator for 8-12 wafer batch documents for foundry company.

**Tasks**:
1. Implement `GenerateWaferBatches(company Company, rng *SeededRand) []WaferBatch`
2. Generate realistic data: batch IDs (WF-2025-XXXX), lot numbers, 300mm dimensions, silicon material
3. Use `rng.IntRange(8, 12)` for count
4. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestGenerateWaferBatches(t *testing.T) {
    t.Parallel()

    company := generator.Company{
        Name: "Pacific Silicon Foundry",
        Role: "foundry",
        URN:  "urn:supply-chain:pacific-silicon-foundry",
    }

    timestamp := time.Date(2025, 10, 18, 14, 30, 22, 0, time.UTC)
    rng := generator.NewSeededRand(timestamp)

    wafers := generator.GenerateWaferBatches(company, rng)

    assert.GreaterOrEqual(t, len(wafers), 8)
    assert.LessOrEqual(t, len(wafers), 12)

    for _, w := range wafers {
        assert.True(t, strings.HasPrefix(w.BatchID, "WF-2025-"))
        assert.Equal(t, "300mm", w.Dimensions)
        assert.Contains(t, w.SubjectURN, "wafer-batch")
    }
}

func TestWaferBatchesDeterministic(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Test", Role: "foundry", URN: "urn:test"}
    timestamp := time.Date(2025, 10, 18, 14, 30, 22, 0, time.UTC)

    rng1 := generator.NewSeededRand(timestamp)
    wafers1 := generator.GenerateWaferBatches(company, rng1)

    rng2 := generator.NewSeededRand(timestamp)
    wafers2 := generator.GenerateWaferBatches(company, rng2)

    assert.Equal(t, len(wafers1), len(wafers2))
    for i := range wafers1 {
        assert.Equal(t, wafers1[i].BatchID, wafers2[i].BatchID)
    }
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Generates 8-12 documents per call
- [ ] Same timestamp produces identical documents
- [ ] All fields populated with realistic data

**Files Modified**:
- `internal/generator/documents.go`
- `internal/generator/generator_test.go`

---

### T008: Generate Mineral Sourcing Documents [P]
**Type**: Document Generation
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 1.5 hours
**Dependencies**: T003, T004, T005, T006

**Description**: Implement generator for 6-8 mineral sourcing documents for foundry company.

**Tasks**:
1. Implement `GenerateMineralSourcing(company Company, rng *SeededRand) []MineralSourcing`
2. Generate data for 4 conflict minerals: tin, tungsten, tantalum, gold
3. Include origin countries, certifications (e.g., RMI certified), quantities
4. Use `rng.IntRange(6, 8)` for count
5. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestGenerateMineralSourcing(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Pacific Silicon Foundry", Role: "foundry"}
    timestamp := time.Now()
    rng := generator.NewSeededRand(timestamp)

    minerals := generator.GenerateMineralSourcing(company, rng)

    assert.GreaterOrEqual(t, len(minerals), 6)
    assert.LessOrEqual(t, len(minerals), 8)

    // Verify conflict minerals present
    mineralTypes := make(map[string]bool)
    for _, m := range minerals {
        mineralTypes[m.MineralType] = true
    }
    assert.True(t, mineralTypes["tin"] || mineralTypes["tungsten"] ||
                mineralTypes["tantalum"] || mineralTypes["gold"])
}

func TestMineralSourcingCertification(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Test", Role: "foundry"}
    rng := generator.NewSeededRand(time.Now())

    minerals := generator.GenerateMineralSourcing(company, rng)

    for _, m := range minerals {
        assert.NotEmpty(t, m.Certification)
        assert.NotEmpty(t, m.OriginCountry)
        assert.Greater(t, m.Quantity, 0.0)
    }
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Generates 6-8 documents
- [ ] Includes 4 conflict minerals (tin, tungsten, tantalum, gold)
- [ ] Realistic origin countries and certifications

**Files Modified**:
- `internal/generator/documents.go`
- `internal/generator/generator_test.go`

---

### T009: Generate Chip Specification Documents [P]
**Type**: Document Generation
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 1.5 hours
**Dependencies**: T003, T004, T005, T006

**Description**: Implement generator for 10-14 chip specification documents for IDM company.

**Tasks**:
1. Implement `GenerateChipSpecifications(company Company, rng *SeededRand) []ChipSpecification`
2. Generate CPU specs: part numbers (APX-XXXX), frequencies (3.5-5.0 GHz), core counts (8-16), TDP (45-125W)
3. Include NPU specifications for AI-capable laptop
4. Use `rng.IntRange(10, 14)` for count
5. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestGenerateChipSpecifications(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Apex Semiconductor Corp", Role: "IDM"}
    rng := generator.NewSeededRand(time.Now())

    chips := generator.GenerateChipSpecifications(company, rng)

    assert.GreaterOrEqual(t, len(chips), 10)
    assert.LessOrEqual(t, len(chips), 14)

    for _, c := range chips {
        assert.True(t, strings.HasPrefix(c.PartNumber, "APX-"))
        assert.GreaterOrEqual(t, c.FrequencyGHz, 3.5)
        assert.LessOrEqual(t, c.FrequencyGHz, 5.0)
        assert.GreaterOrEqual(t, c.CoreCount, 8)
        assert.LessOrEqual(t, c.CoreCount, 16)
    }
}

func TestChipSpecsIncludeNPU(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Test", Role: "IDM"}
    rng := generator.NewSeededRand(time.Now())

    chips := generator.GenerateChipSpecifications(company, rng)

    // At least one chip should have NPU specs
    hasNPU := false
    for _, c := range chips {
        if c.NPU != nil {
            hasNPU = true
            assert.Greater(t, c.NPU.TOPS, 0.0, "NPU TOPS should be positive")
        }
    }
    assert.True(t, hasNPU, "At least one chip should include NPU")
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Generates 10-14 documents
- [ ] Realistic CPU specs with frequencies, core counts, TDP
- [ ] At least one chip includes NPU specifications

**Files Modified**:
- `internal/generator/documents.go`
- `internal/generator/generator_test.go`

---

### T010: Generate Firmware Manifest Documents [P]
**Type**: Document Generation
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 1.5 hours
**Dependencies**: T003, T004, T005, T006

**Description**: Implement generator for 8-10 firmware manifest documents for IDM company.

**Tasks**:
1. Implement `GenerateFirmwareManifests(company Company, rng *SeededRand) []FirmwareManifest`
2. Generate firmware data: versions (1.x.x), SHA-256 hashes, file sizes
3. Include firmware types: UEFI/BIOS, NPU firmware, GPU firmware
4. Use `rng.IntRange(8, 10)` for count
5. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestGenerateFirmwareManifests(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Apex Semiconductor Corp", Role: "IDM"}
    rng := generator.NewSeededRand(time.Now())

    firmwares := generator.GenerateFirmwareManifests(company, rng)

    assert.GreaterOrEqual(t, len(firmwares), 8)
    assert.LessOrEqual(t, len(firmwares), 10)

    for _, f := range firmwares {
        assert.Regexp(t, `^\d+\.\d+\.\d+$`, f.Version)
        assert.Len(t, f.Hash, 64, "SHA-256 hash should be 64 hex chars")
        assert.Greater(t, f.FileSizeBytes, int64(0))
    }
}

func TestFirmwareTypesVariety(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Test", Role: "IDM"}
    rng := generator.NewSeededRand(time.Now())

    firmwares := generator.GenerateFirmwareManifests(company, rng)

    types := make(map[string]bool)
    for _, f := range firmwares {
        types[f.FirmwareType] = true
    }

    // Should have variety of firmware types
    assert.GreaterOrEqual(t, len(types), 2)
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Generates 8-10 documents
- [ ] Semantic version format (x.y.z)
- [ ] Valid SHA-256 hashes (64 hex characters)
- [ ] Multiple firmware types (UEFI, NPU, GPU)

**Files Modified**:
- `internal/generator/documents.go`
- `internal/generator/generator_test.go`

---

### T011: Generate SBOM/HBOM Documents (SPDX) [P]
**Type**: Document Generation
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 2 hours
**Dependencies**: T003, T004, T005, T006

**Description**: Implement generator for 12-16 SBOM/HBOM documents in SPDX 2.3 format for IDM company (per research.md § 1).

**Tasks**:
1. Implement `GenerateSBOMs(company Company, rng *SeededRand) []SBOM`
2. Follow SPDX 2.3 minimal schema from research.md
3. Generate SBOMs for: laptop, CPU, firmware, components
4. Include 3-5 packages per SBOM with suppliers, versions
5. Use `rng.IntRange(12, 16)` for count
6. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestGenerateSBOMs(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Apex Semiconductor Corp", Role: "IDM"}
    rng := generator.NewSeededRand(time.Now())

    sboms := generator.GenerateSBOMs(company, rng)

    assert.GreaterOrEqual(t, len(sboms), 12)
    assert.LessOrEqual(t, len(sboms), 16)

    for _, s := range sboms {
        assert.Equal(t, "SPDX-2.3", s.SPDXVersion)
        assert.Equal(t, "CC0-1.0", s.DataLicense)
        assert.Equal(t, "SPDXRef-DOCUMENT", s.SPDXID)
        assert.NotEmpty(t, s.DocumentName)
        assert.True(t, strings.HasPrefix(s.DocumentNamespace, "https://"))
    }
}

func TestSBOMCreationInfo(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Test", Role: "IDM"}
    rng := generator.NewSeededRand(time.Now())

    sboms := generator.GenerateSBOMs(company, rng)

    for _, s := range sboms {
        assert.NotEmpty(t, s.CreationInfo.Created)
        assert.Contains(t, s.CreationInfo.Creators, "Tool: scitt-feed-generator")
        assert.GreaterOrEqual(t, len(s.Packages), 1)
        assert.LessOrEqual(t, len(s.Packages), 5)
    }
}

func TestSBOMPackageStructure(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Test", Role: "IDM"}
    rng := generator.NewSeededRand(time.Now())

    sboms := generator.GenerateSBOMs(company, rng)

    for _, s := range sboms {
        for _, pkg := range s.Packages {
            assert.True(t, strings.HasPrefix(pkg.SPDXID, "SPDXRef-Package-"))
            assert.NotEmpty(t, pkg.Name)
            assert.False(t, pkg.FilesAnalyzed)
        }
    }
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Generates 12-16 documents
- [ ] SPDX 2.3 compliant (version, license, SPDXID, namespace)
- [ ] Each SBOM has 3-5 packages
- [ ] Realistic component names and suppliers

**Files Modified**:
- `internal/generator/documents.go`
- `internal/generator/generator_test.go`

---

### T012: Generate Memory Specification Documents [P]
**Type**: Document Generation
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 1 hour
**Dependencies**: T003, T004, T005, T006

**Description**: Implement generator for 6-8 memory specification documents for fabless company.

**Tasks**:
1. Implement `GenerateMemorySpecifications(company Company, rng *SeededRand) []MemorySpecification`
2. Generate DDR5 memory specs: capacities (8GB, 16GB, 32GB), speeds (4800-6400 MT/s), timings
3. Use `rng.IntRange(6, 8)` for count
4. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestGenerateMemorySpecifications(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Quantum Chip Design", Role: "fabless"}
    rng := generator.NewSeededRand(time.Now())

    memories := generator.GenerateMemorySpecifications(company, rng)

    assert.GreaterOrEqual(t, len(memories), 6)
    assert.LessOrEqual(t, len(memories), 8)

    for _, m := range memories {
        assert.Contains(t, []int{8, 16, 32}, m.CapacityGB)
        assert.GreaterOrEqual(t, m.SpeedMTs, 4800)
        assert.LessOrEqual(t, m.SpeedMTs, 6400)
        assert.Equal(t, "DDR5", m.MemoryType)
    }
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Generates 6-8 documents
- [ ] DDR5 memory specs with realistic speeds and capacities
- [ ] Timings (CL-RCD-RP-RAS format)

**Files Modified**:
- `internal/generator/documents.go`
- `internal/generator/generator_test.go`

---

### T013: Generate AI Training Dataset Documents [P]
**Type**: Document Generation
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 1.5 hours
**Dependencies**: T003, T004, T005, T006

**Description**: Implement generator for 4-6 AI training dataset documents for fabless company.

**Tasks**:
1. Implement `GenerateAITrainingDatasets(company Company, rng *SeededRand) []AITrainingDataset`
2. Generate datasets: names (ImageNet, COCO, OpenWebText), sources, licensing, data provenance
3. Include dataset metadata: size, record count, modality
4. Use `rng.IntRange(4, 6)` for count
5. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestGenerateAITrainingDatasets(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Quantum Chip Design", Role: "fabless"}
    rng := generator.NewSeededRand(time.Now())

    datasets := generator.GenerateAITrainingDatasets(company, rng)

    assert.GreaterOrEqual(t, len(datasets), 4)
    assert.LessOrEqual(t, len(datasets), 6)

    for _, d := range datasets {
        assert.NotEmpty(t, d.DatasetName)
        assert.NotEmpty(t, d.Source)
        assert.NotEmpty(t, d.License)
        assert.Greater(t, d.RecordCount, int64(0))
    }
}

func TestAIDatasetModalities(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Test", Role: "fabless"}
    rng := generator.NewSeededRand(time.Now())

    datasets := generator.GenerateAITrainingDatasets(company, rng)

    for _, d := range datasets {
        assert.Contains(t, []string{"image", "text", "multimodal"}, d.Modality)
    }
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Generates 4-6 documents
- [ ] Realistic dataset names and sources
- [ ] Licensing information (MIT, Apache, CC-BY, etc.)
- [ ] Data provenance fields

**Files Modified**:
- `internal/generator/documents.go`
- `internal/generator/generator_test.go`

---

### T014: Generate AI Model Specification Documents [P]
**Type**: Document Generation
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 1.5 hours
**Dependencies**: T003, T004, T005, T006

**Description**: Implement generator for 3-5 AI model specification documents for fabless company.

**Tasks**:
1. Implement `GenerateAIModelSpecifications(company Company, rng *SeededRand) []AIModelSpecification`
2. Generate model specs: architectures (Transformer, CNN, LSTM), parameters, quantization (INT8, FP16)
3. Include inference requirements: TOPS, memory footprint
4. Use `rng.IntRange(3, 5)` for count
5. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestGenerateAIModelSpecifications(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Quantum Chip Design", Role: "fabless"}
    rng := generator.NewSeededRand(time.Now())

    models := generator.GenerateAIModelSpecifications(company, rng)

    assert.GreaterOrEqual(t, len(models), 3)
    assert.LessOrEqual(t, len(models), 5)

    for _, m := range models {
        assert.NotEmpty(t, m.ModelName)
        assert.NotEmpty(t, m.Architecture)
        assert.Greater(t, m.ParameterCount, int64(0))
    }
}

func TestAIModelQuantization(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Test", Role: "fabless"}
    rng := generator.NewSeededRand(time.Now())

    models := generator.GenerateAIModelSpecifications(company, rng)

    for _, m := range models {
        assert.Contains(t, []string{"FP32", "FP16", "INT8", "INT4"}, m.Quantization)
        assert.Greater(t, m.InferenceTOPS, 0.0)
    }
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Generates 3-5 documents
- [ ] Realistic model architectures and parameter counts
- [ ] Quantization formats and inference requirements

**Files Modified**:
- `internal/generator/documents.go`
- `internal/generator/generator_test.go`

---

### T015: Generate CVE/CWE Vulnerability Documents [P]
**Type**: Document Generation
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 2 hours
**Dependencies**: T003, T004, T005, T006

**Description**: Implement generator for 5-7 CVE/CWE vulnerability disclosure documents for fabless company (per research.md § 2).

**Tasks**:
1. Implement `GenerateCVEDocuments(company Company, rng *SeededRand) []CVEDocument`
2. Use real CVE IDs from research.md: CVE-2024-0519, CVE-2024-3660, CVE-2024-5480, CVE-2024-22476, CVE-2024-35198, CVE-2024-35199
3. Add CWE classifications: CWE-502, CWE-20
4. Generate CVSS scores (7.0-10.0 for high/critical, including 10.0 for CVE-2024-22476)
5. Reference affected firmware/AI framework versions
6. Use `rng.IntRange(5, 7)` for count
7. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestGenerateCVEDocuments(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Quantum Chip Design", Role: "fabless"}
    rng := generator.NewSeededRand(time.Now())

    cves := generator.GenerateCVEDocuments(company, rng)

    assert.GreaterOrEqual(t, len(cves), 5)
    assert.LessOrEqual(t, len(cves), 7)

    for _, c := range cves {
        // Verify CVE ID format
        if strings.HasPrefix(c.VulnerabilityID, "CVE-") {
            assert.Regexp(t, `^CVE-\d{4}-\d+$`, c.VulnerabilityID)
        } else if strings.HasPrefix(c.VulnerabilityID, "CWE-") {
            assert.Regexp(t, `^CWE-\d+$`, c.VulnerabilityID)
        }

        assert.GreaterOrEqual(t, c.CVSSScore, 7.0)
        assert.LessOrEqual(t, c.CVSSScore, 10.0)
    }
}

func TestCVEDocumentsUseRealIDs(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Test", Role: "fabless"}
    rng := generator.NewSeededRand(time.Now())

    cves := generator.GenerateCVEDocuments(company, rng)

    realCVEs := []string{"CVE-2024-0519", "CVE-2024-3660", "CVE-2024-5480", "CVE-2024-22476", "CVE-2024-35198", "CVE-2024-35199"}
    realCWEs := []string{"CWE-502", "CWE-20"}

    // At least one CVE should be from the real list
    foundReal := false
    for _, c := range cves {
        for _, realID := range append(realCVEs, realCWEs...) {
            if c.VulnerabilityID == realID {
                foundReal = true
                break
            }
        }
    }
    assert.True(t, foundReal, "Should use at least one real CVE/CWE ID")
}

func TestCVEDocumentsAffectedVersions(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Test", Role: "fabless"}
    rng := generator.NewSeededRand(time.Now())

    cves := generator.GenerateCVEDocuments(company, rng)

    for _, c := range cves {
        assert.NotEmpty(t, c.AffectedComponent)
        assert.NotEmpty(t, c.AffectedVersions)
        assert.NotEmpty(t, c.Description)
    }
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Generates 5-7 documents
- [ ] Uses real CVE IDs from research (CVE-2024-0519, CVE-2024-3660, CVE-2024-5480, CVE-2024-22476, etc.)
- [ ] Includes CWE classifications (CWE-502, CWE-20)
- [ ] CVSS scores in high/critical range (7.0-10.0, including 10.0 for CVE-2024-22476)
- [ ] References affected components (GPU driver, PyTorch, TensorFlow, Intel Neural Compressor)

**Files Modified**:
- `internal/generator/documents.go`
- `internal/generator/generator_test.go`

---

### T016: Generate Logistics Tracking Documents [P]
**Type**: Document Generation
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 1.5 hours
**Dependencies**: T003, T004, T005, T006

**Description**: Implement generator for 8-12 logistics tracking documents for foundry company.

**Tasks**:
1. Implement `GenerateLogisticsTracking(company Company, rng *SeededRand) []LogisticsTracking`
2. Generate shipment data: IDs, origins, destinations, timestamps, tracking status
3. Create supply chain journey: foundry → assembly → distribution → retailer
4. Use `rng.IntRange(8, 12)` for count
5. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestGenerateLogisticsTracking(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Pacific Silicon Foundry", Role: "foundry"}
    rng := generator.NewSeededRand(time.Now())

    logistics := generator.GenerateLogisticsTracking(company, rng)

    assert.GreaterOrEqual(t, len(logistics), 8)
    assert.LessOrEqual(t, len(logistics), 12)

    for _, l := range logistics {
        assert.NotEmpty(t, l.ShipmentID)
        assert.NotEmpty(t, l.Origin)
        assert.NotEmpty(t, l.Destination)
        assert.Contains(t, []string{"in_transit", "delivered", "customs"}, l.Status)
    }
}

func TestLogisticsTrackingChain(t *testing.T) {
    t.Parallel()

    company := generator.Company{Name: "Test", Role: "foundry"}
    rng := generator.NewSeededRand(time.Now())

    logistics := generator.GenerateLogisticsTracking(company, rng)

    // Should have variety of locations in supply chain
    locations := make(map[string]bool)
    for _, l := range logistics {
        locations[l.Origin] = true
        locations[l.Destination] = true
    }

    assert.GreaterOrEqual(t, len(locations), 3, "Should have multiple locations in chain")
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Generates 8-12 documents
- [ ] Realistic shipment IDs, locations, statuses
- [ ] Documents form logical supply chain journey

**Files Modified**:
- `internal/generator/documents.go`
- `internal/generator/generator_test.go`

---

### T017: Implement Feed Directory Creation
**Type**: Core Logic
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 1.5 hours
**Dependencies**: T001, T004

**Description**: Create feed directory structure with timestamp and company subdirectories.

**Tasks**:
1. Implement `CreateFeedDirectory() (string, time.Time, error)`
2. Generate timestamp in format: `feed-YYYY-MM-DD-HHMMSS/`
3. Create company subdirectories with `documents/` folders
4. Handle directory collision (append suffix if exists)
5. Create `metadata.json` file
6. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestCreateFeedDirectory(t *testing.T) {
    t.Parallel()

    feedDir, timestamp, err := generator.CreateFeedDirectory(t.TempDir())
    require.NoError(t, err)

    assert.True(t, strings.HasPrefix(filepath.Base(feedDir), "feed-"))
    assert.NotZero(t, timestamp)

    // Verify directory exists
    info, err := os.Stat(feedDir)
    require.NoError(t, err)
    assert.True(t, info.IsDir())
}

func TestFeedDirectoryStructure(t *testing.T) {
    t.Parallel()

    feedDir, _, err := generator.CreateFeedDirectory(t.TempDir())
    require.NoError(t, err)

    companies := []string{"pacific-silicon-foundry", "apex-semiconductor-corp", "quantum-chip-design"}
    for _, company := range companies {
        companyDir := filepath.Join(feedDir, company)
        assert.DirExists(t, companyDir)

        documentsDir := filepath.Join(companyDir, "documents")
        assert.DirExists(t, documentsDir)
    }
}

func TestFeedDirectoryCollision(t *testing.T) {
    t.Parallel()

    baseDir := t.TempDir()

    // Create first feed
    feedDir1, _, err := generator.CreateFeedDirectory(baseDir)
    require.NoError(t, err)

    // Force same timestamp to test collision handling
    feedDir2, _, err := generator.CreateFeedDirectoryWithTimestamp(baseDir, time.Now())
    require.NoError(t, err)

    assert.NotEqual(t, feedDir1, feedDir2, "Colliding directories should get unique names")
}

func TestFeedMetadataJSON(t *testing.T) {
    t.Parallel()

    feedDir, timestamp, err := generator.CreateFeedDirectory(t.TempDir())
    require.NoError(t, err)

    metadataPath := filepath.Join(feedDir, "metadata.json")
    assert.FileExists(t, metadataPath)

    var metadata map[string]interface{}
    data, err := os.ReadFile(metadataPath)
    require.NoError(t, err)

    err = json.Unmarshal(data, &metadata)
    require.NoError(t, err)

    assert.Contains(t, metadata, "timestamp")
    assert.Contains(t, metadata, "seed")
    assert.Contains(t, metadata, "companies")
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Creates timestamped feed directory
- [ ] Creates 3 company subdirectories with documents/ folders
- [ ] Handles directory name collisions (appends suffix)
- [ ] Creates metadata.json with timestamp, seed, companies list

**Files Modified**:
- `internal/generator/documents.go`
- `internal/generator/generator_test.go`

---

### T018: Implement Document-to-JSON Serialization
**Type**: Core Logic
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 1 hour
**Dependencies**: T007-T016

**Description**: Write generated documents to JSON files with proper naming.

**Tasks**:
1. Implement `WriteDocumentToFile(doc Document, dir string) error`
2. Generate filenames based on document type and ID
3. Marshal to pretty-printed JSON
4. Handle file write errors gracefully
5. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestWriteDocumentToFile(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()

    wafer := &generator.WaferBatch{
        BatchID:    "WF-2025-1001",
        SubjectURN: "urn:supply-chain:test:wafer-batch:WF-2025-1001",
    }

    err := generator.WriteDocumentToFile(wafer, dir)
    require.NoError(t, err)

    // Verify file exists
    expectedPath := filepath.Join(dir, "wafer-batch-001.json")
    assert.FileExists(t, expectedPath)

    // Verify content
    data, err := os.ReadFile(expectedPath)
    require.NoError(t, err)

    var decoded generator.WaferBatch
    err = json.Unmarshal(data, &decoded)
    require.NoError(t, err)

    assert.Equal(t, wafer.BatchID, decoded.BatchID)
}

func TestDocumentJSONFormatting(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()

    doc := &generator.ChipSpecification{
        PartNumber: "APX-9700K",
        SubjectURN: "urn:test:chip:APX-9700K",
    }

    err := generator.WriteDocumentToFile(doc, dir)
    require.NoError(t, err)

    files, err := os.ReadDir(dir)
    require.NoError(t, err)
    require.Len(t, files, 1)

    data, err := os.ReadFile(filepath.Join(dir, files[0].Name()))
    require.NoError(t, err)

    // Verify pretty-printed (contains newlines)
    assert.Contains(t, string(data), "\n")
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Documents written as pretty-printed JSON
- [ ] Filenames follow pattern: `{type}-{sequential}.json`
- [ ] File write errors handled gracefully

**Files Modified**:
- `internal/generator/documents.go`
- `internal/generator/generator_test.go`

---

### T019: Wire Up Feed Generation Command
**Type**: CLI Integration
**User Story**: US1
**Priority**: P1
**Estimated Effort**: 2 hours
**Dependencies**: T001, T002, T017, T018

**Description**: Implement `./scitt feed generate` command that orchestrates all document generation (per quickstart.md).

**Tasks**:
1. Create Cobra command in `internal/cli/feed.go`
2. Implement workflow:
   - Create feed directory
   - Generate companies
   - Call key generation (placeholder for T023)
   - Generate all document types (call T007-T016 functions)
   - Write documents to files
   - Display progress bars
3. Add `--no-sign` and `--no-register` flags (functionality in later tasks)
4. **TDD**: Write integration tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestFeedGenerateCommand(t *testing.T) {
    t.Parallel()

    // Run command in temp directory
    cmd := exec.Command("./scitt", "feed", "generate", "--no-sign", "--no-register")
    cmd.Dir = t.TempDir()

    output, err := cmd.CombinedOutput()
    require.NoError(t, err)

    assert.Contains(t, string(output), "Feed generated successfully")
    assert.Contains(t, string(output), "feed-")
}

func TestFeedGenerateCreatesDocuments(t *testing.T) {
    t.Parallel()

    workDir := t.TempDir()

    cmd := exec.Command("./scitt", "feed", "generate", "--no-sign", "--no-register")
    cmd.Dir = workDir
    output, err := cmd.CombinedOutput()
    require.NoError(t, err)

    // Extract feed directory from output
    re := regexp.MustCompile(`feed-\d{4}-\d{2}-\d{2}-\d{6}`)
    feedDir := re.FindString(string(output))
    require.NotEmpty(t, feedDir)

    fullPath := filepath.Join(workDir, feedDir)

    // Count total documents
    totalDocs := 0
    err = filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if !d.IsDir() && strings.HasSuffix(path, ".json") && !strings.Contains(path, "metadata") {
            totalDocs++
        }
        return nil
    })
    require.NoError(t, err)

    assert.GreaterOrEqual(t, totalDocs, 96, "Should generate at least 96 documents")
}

func TestFeedGenerateWithProgressBar(t *testing.T) {
    // Verify progress bar output appears (integration test)
    // May need to capture stderr to test progress bars
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] `./scitt feed generate --no-sign --no-register` creates complete feed directory
- [ ] At least 96 documents generated across 3 companies
- [ ] Progress bar displays during generation
- [ ] Metadata.json created with correct data
- [ ] Command exits successfully with summary message

**Files Modified**:
- `internal/cli/feed.go`
- `cmd/scitt/main.go`
- `internal/cli/feed_test.go` (new)

---

**CHECKPOINT: User Story 1 (P1) Complete**

At this point:
- ✅ Full feed directory with 96+ documents can be generated
- ✅ All 10 document types implemented
- ✅ Deterministic generation working
- ✅ Progress feedback operational
- ✅ Feature delivers immediate value for testing without requiring signing/registration
- ✅ Independently testable

---

## Phase 3: User Story 2 (P2) - Sign Generated Documents

**Goal**: Add signing workflow using existing `scitt issuer keygen` and `scitt statement sign` commands (per research.md § 5).

---

### T020: Implement Company Key Generation via Existing Commands
**Type**: Workflow Orchestration
**User Story**: US2
**Priority**: P2
**Estimated Effort**: 1.5 hours
**Dependencies**: T019

**Description**: Generate ES256 key pairs for each company by calling existing `scitt issuer keygen` command.

**Tasks**:
1. Implement `generateCompanyKeys(feedDir string, companies []Company) error`
2. For each company, execute: `./scitt issuer keygen --private {dir}/priv.cbor --public {dir}/pub.cbor --algorithm ES256`
3. Use `exec.Command()` to call existing CLI command
4. Display progress: "Creating keys for {Company} (1/3)..."
5. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestGenerateCompanyKeys(t *testing.T) {
    t.Parallel()

    feedDir := t.TempDir()
    companies := []generator.Company{
        {Name: "Test Company", Role: "foundry", Directory: "test-company"},
    }

    // Create company directory
    companyDir := filepath.Join(feedDir, "test-company")
    os.MkdirAll(companyDir, 0755)

    err := cli.GenerateCompanyKeys(feedDir, companies)
    require.NoError(t, err)

    // Verify keys exist
    privPath := filepath.Join(companyDir, "priv.cbor")
    pubPath := filepath.Join(companyDir, "pub.cbor")

    assert.FileExists(t, privPath)
    assert.FileExists(t, pubPath)
}

func TestGenerateCompanyKeysES256(t *testing.T) {
    t.Parallel()

    feedDir := t.TempDir()
    companies := []generator.Company{
        {Name: "Test", Role: "foundry", Directory: "test"},
    }

    os.MkdirAll(filepath.Join(feedDir, "test"), 0755)

    err := cli.GenerateCompanyKeys(feedDir, companies)
    require.NoError(t, err)

    // Verify key algorithm by attempting to read with COSE library
    privPath := filepath.Join(feedDir, "test", "priv.cbor")
    keyBytes, err := os.ReadFile(privPath)
    require.NoError(t, err)

    // Should be valid COSE_Key CBOR
    assert.Greater(t, len(keyBytes), 0)
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Generates ES256 key pairs for all 3 companies
- [ ] Keys written as `priv.cbor` and `pub.cbor` in COSE_Key format
- [ ] Uses existing `./scitt issuer keygen` command via exec.Command()
- [ ] Progress messages displayed

**Files Modified**:
- `internal/cli/feed.go`
- `internal/cli/feed_test.go`

---

### T021: Implement Document Signing via Existing Commands
**Type**: Workflow Orchestration
**User Story**: US2
**Priority**: P2
**Estimated Effort**: 2.5 hours
**Dependencies**: T020

**Description**: Sign all documents using existing `scitt statement sign` command for each company (per research.md § 5).

**Tasks**:
1. Implement `signDocumentsForCompany(feedDir string, company Company) error`
2. For each JSON document in company's `documents/` folder:
   - Extract metadata (content-type, content-location, subject URN)
   - Execute: `./scitt statement sign --content {doc.json} --content-type application/json --content-location {url} --issuer {company.urn} --subject {doc.urn} --signing-key {priv.cbor} --signed-statement {doc.cbor}`
3. Use progress bar per company: "Signing documents for {Company} (1/3)... [42/42]"
4. Handle signing errors gracefully (log, continue with remaining)
5. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestSignDocumentsForCompany(t *testing.T) {
    t.Parallel()

    feedDir := t.TempDir()

    // Setup: create test company directory with keys and documents
    company := generator.Company{
        Name:      "Test Company",
        Role:      "foundry",
        URN:       "urn:supply-chain:test-company",
        Directory: "test-company",
    }

    companyDir := filepath.Join(feedDir, "test-company")
    documentsDir := filepath.Join(companyDir, "documents")
    os.MkdirAll(documentsDir, 0755)

    // Generate real keys using scitt issuer keygen
    cmd := exec.Command("./scitt", "issuer", "keygen",
        "--private", filepath.Join(companyDir, "priv.cbor"),
        "--public", filepath.Join(companyDir, "pub.cbor"),
        "--algorithm", "ES256")
    err := cmd.Run()
    require.NoError(t, err)

    // Create test document
    doc := map[string]interface{}{
        "batch_id":    "TEST-001",
        "subject_urn": "urn:supply-chain:test-company:wafer-batch:TEST-001",
    }
    docBytes, _ := json.Marshal(doc)
    docPath := filepath.Join(documentsDir, "wafer-batch-001.json")
    os.WriteFile(docPath, docBytes, 0644)

    // Test signing
    err = cli.SignDocumentsForCompany(feedDir, company)
    require.NoError(t, err)

    // Verify signed statement created
    stmtPath := filepath.Join(documentsDir, "wafer-batch-001.cbor")
    assert.FileExists(t, stmtPath)
}

func TestSignDocumentsProgressBar(t *testing.T) {
    // Integration test to verify progress bar output
    // May need stderr capture
}

func TestSignDocumentsErrorHandling(t *testing.T) {
    t.Parallel()

    feedDir := t.TempDir()
    company := generator.Company{
        Name:      "Test",
        Directory: "test",
    }

    companyDir := filepath.Join(feedDir, "test")
    documentsDir := filepath.Join(companyDir, "documents")
    os.MkdirAll(documentsDir, 0755)

    // Don't create keys - should fail gracefully
    doc := map[string]interface{}{"test": "data"}
    docBytes, _ := json.Marshal(doc)
    os.WriteFile(filepath.Join(documentsDir, "test.json"), docBytes, 0644)

    err := cli.SignDocumentsForCompany(feedDir, company)
    assert.Error(t, err, "Should fail when keys missing")
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Signs all JSON documents in company's documents/ folder
- [ ] Creates `.cbor` signed statement files alongside JSON documents
- [ ] Uses existing `./scitt statement sign` command via exec.Command()
- [ ] Progress bar shows per-company progress
- [ ] Handles missing keys gracefully with clear error
- [ ] Continues signing remaining documents if one fails

**Files Modified**:
- `internal/cli/feed.go`
- `internal/cli/feed_test.go`

---

### T022: Add Interactive Signing Prompt
**Type**: User Interaction
**User Story**: US2
**Priority**: P2
**Estimated Effort**: 1 hour
**Dependencies**: T021

**Description**: Add interactive prompt after generation asking user if they want to sign documents.

**Tasks**:
1. After document generation completes, display: "Ready to sign documents? (yes/no):"
2. Read user input from stdin
3. If "yes", proceed with signing workflow (call T020, T021)
4. If "no", skip signing and proceed to registration prompt (T025)
5. Respect `--no-sign` flag to skip prompt entirely
6. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestInteractiveSigningPromptYes(t *testing.T) {
    // Test with simulated "yes" input
    // Use io.Pipe to simulate stdin
}

func TestInteractiveSigningPromptNo(t *testing.T) {
    // Test with simulated "no" input
    // Verify signing skipped
}

func TestNoSignFlag(t *testing.T) {
    t.Parallel()

    cmd := exec.Command("./scitt", "feed", "generate", "--no-sign", "--no-register")
    output, err := cmd.CombinedOutput()
    require.NoError(t, err)

    // Should not prompt for signing
    assert.NotContains(t, string(output), "Ready to sign documents?")
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Prompt displays after generation
- [ ] "yes" triggers signing workflow
- [ ] "no" skips to registration prompt
- [ ] `--no-sign` flag bypasses prompt entirely
- [ ] Invalid input re-prompts user

**Files Modified**:
- `internal/cli/feed.go`
- `internal/cli/feed_test.go`

---

**CHECKPOINT: User Story 2 (P2) Complete**

At this point:
- ✅ All documents can be cryptographically signed
- ✅ Uses existing `scitt issuer keygen` and `scitt statement sign` commands
- ✅ Interactive workflow with user prompts
- ✅ Progress feedback for signing
- ✅ Independently testable with pre-generated feed directory
- ✅ Delivers value by automating tedious signing of 96+ documents

---

## Phase 4: User Story 3 (P3) - Register Statements to Transparency Log

**Goal**: Register all signed statements to SCITT service using existing `scitt statement register` command.

---

### T023: Implement Service Connection Validation
**Type**: Service Integration
**User Story**: US3
**Priority**: P3
**Estimated Effort**: 1 hour
**Dependencies**: T019

**Description**: Validate SCITT service connectivity before registration.

**Tasks**:
1. Implement `validateServiceConnection(serviceURL string) error`
2. Query `/.well-known/scitt-configuration` endpoint
3. Parse and validate response
4. Display: "Connected to SCITT service: {issuer URL}"
5. Return clear error if connection fails
6. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestValidateServiceConnection(t *testing.T) {
    // Requires running SCITT service
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    serviceURL := "http://127.0.0.1:56177"
    err := cli.ValidateServiceConnection(serviceURL)
    assert.NoError(t, err)
}

func TestValidateServiceConnectionInvalidURL(t *testing.T) {
    t.Parallel()

    err := cli.ValidateServiceConnection("http://localhost:99999")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "Cannot connect to SCITT service")
}

func TestValidateServiceConnectionMalformedResponse(t *testing.T) {
    // Mock server returning invalid JSON
    // Verify error handling
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Queries `/.well-known/scitt-configuration` endpoint
- [ ] Returns nil on successful connection
- [ ] Returns clear error message on failure
- [ ] Error includes service URL for debugging

**Files Modified**:
- `internal/cli/feed.go`
- `internal/cli/feed_test.go`

---

### T024: Implement Statement Registration via Existing Commands
**Type**: Workflow Orchestration
**User Story**: US3
**Priority**: P3
**Estimated Effort**: 2.5 hours
**Dependencies**: T023

**Description**: Register all signed statements using existing `scitt statement register` command (per research.md § 5).

**Tasks**:
1. Implement `registerStatements(feedDir string, serviceURL string, apiKey string) error`
2. Collect all `.cbor` statement files across all companies
3. For each statement:
   - Execute: `./scitt statement register --service {url} --api-key {key} --statement {stmt.cbor} --receipt {stmt.receipt.cbor}`
4. Display progress bar: "Registering 126 statements... [23/126] (18%)"
5. Track registration errors, continue with remaining statements
6. Query service for final tree size and tile count
7. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestRegisterStatements(t *testing.T) {
    // Requires running SCITT service
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    feedDir := setupTestFeedWithSignedStatements(t)

    serviceURL := "http://127.0.0.1:56177"
    apiKey := os.Getenv("SCITT_API_KEY")
    require.NotEmpty(t, apiKey, "SCITT_API_KEY required for integration test")

    err := cli.RegisterStatements(feedDir, serviceURL, apiKey)
    require.NoError(t, err)

    // Verify receipts created
    receiptCount := 0
    filepath.WalkDir(feedDir, func(path string, d fs.DirEntry, err error) error {
        if strings.HasSuffix(path, ".receipt.cbor") {
            receiptCount++
        }
        return nil
    })

    assert.Greater(t, receiptCount, 0, "Should create receipt files")
}

func TestRegisterStatementsInvalidAPIKey(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    feedDir := setupTestFeedWithSignedStatements(t)

    err := cli.RegisterStatements(feedDir, "http://127.0.0.1:56177", "invalid-key")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "401")
}

func TestRegisterStatementsProgressBar(t *testing.T) {
    // Integration test to verify progress bar output
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Registers all `.cbor` statements across all companies
- [ ] Creates `.receipt.cbor` files for successful registrations
- [ ] Uses existing `./scitt statement register` command via exec.Command()
- [ ] Progress bar shows percentage and ETA
- [ ] Handles 401 errors (invalid API key) with clear message
- [ ] Handles 503 errors (service unavailable) gracefully
- [ ] Displays final summary with tree size and tile count

**Files Modified**:
- `internal/cli/feed.go`
- `internal/cli/feed_test.go`

---

### T025: Add Interactive Registration Prompt
**Type**: User Interaction
**User Story**: US3
**Priority**: P3
**Estimated Effort**: 1 hour
**Dependencies**: T024

**Description**: Add interactive prompts for service URL and registration confirmation.

**Tasks**:
1. After signing completes, display: "Ready to register statements? (yes/no):"
2. If "yes", prompt: "SCITT service URL [default: http://127.0.0.1:56177]:"
3. Read service URL (use default if empty)
4. Check for `SCITT_API_KEY` environment variable
5. If missing, prompt: "SCITT API Key:"
6. Validate service connection (T023)
7. Proceed with registration (T024)
8. Respect `--no-register` flag to skip prompt entirely
9. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestInteractiveRegistrationPromptYes(t *testing.T) {
    // Test with simulated "yes" input
    // Use io.Pipe to simulate stdin
}

func TestInteractiveRegistrationPromptNo(t *testing.T) {
    // Test with simulated "no" input
    // Verify registration skipped
}

func TestNoRegisterFlag(t *testing.T) {
    t.Parallel()

    cmd := exec.Command("./scitt", "feed", "generate", "--no-register")
    output, err := cmd.CombinedOutput()
    require.NoError(t, err)

    // Should not prompt for registration
    assert.NotContains(t, string(output), "Ready to register statements?")
}

func TestRegistrationPromptDefaultURL(t *testing.T) {
    // Test empty input uses default URL
}

func TestRegistrationPromptMissingAPIKey(t *testing.T) {
    // Test prompts for API key when env var missing
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Prompt displays after signing
- [ ] "yes" proceeds to service URL prompt
- [ ] "no" skips registration
- [ ] Default service URL works (http://127.0.0.1:56177)
- [ ] Custom service URL accepted
- [ ] Checks `SCITT_API_KEY` environment variable
- [ ] Prompts for API key if not set
- [ ] `--no-register` flag bypasses prompts
- [ ] Invalid input re-prompts user

**Files Modified**:
- `internal/cli/feed.go`
- `internal/cli/feed_test.go`

---

### T026: Verify Tile Creation
**Type**: Validation
**User Story**: US3
**Priority**: P3
**Estimated Effort**: 1 hour
**Dependencies**: T024

**Description**: Verify at least 3 complete tiles exist in tile storage after registration.

**Tasks**:
1. Implement `verifyTileCreation(treeSize int) error`
2. Calculate expected tile count: `treeSize / 32` (rounded up)
3. Display in summary: "Tiles created: 4 (entries 0-31, 32-63, 64-95, 96-126)"
4. If tile count < 3, return warning (not error, since data was registered)
5. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestVerifyTileCreation(t *testing.T) {
    t.Parallel()

    tests := []struct {
        treeSize      int
        expectedTiles int
    }{
        {32, 1},
        {64, 2},
        {96, 3},
        {104, 4},
        {126, 4},
    }

    for _, tt := range tests {
        tiles := cli.CalculateTileCount(tt.treeSize)
        assert.Equal(t, tt.expectedTiles, tiles)
    }
}

func TestVerifyTileCreationWarning(t *testing.T) {
    t.Parallel()

    // Fewer than 3 tiles should produce warning
    warning := cli.VerifyTileCreation(64) // 2 tiles
    assert.NotEmpty(t, warning)
    assert.Contains(t, warning, "fewer than 3")
}

func TestVerifyTileCreationSuccess(t *testing.T) {
    t.Parallel()

    warning := cli.VerifyTileCreation(126) // 4 tiles
    assert.Empty(t, warning)
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Calculates tile count from tree size
- [ ] Displays tile count in summary message
- [ ] Shows tile ranges (0-31, 32-63, etc.)
- [ ] Warns if fewer than 3 complete tiles created

**Files Modified**:
- `internal/cli/feed.go`
- `internal/cli/feed_test.go`

---

**CHECKPOINT: User Story 3 (P3) Complete**

At this point:
- ✅ Complete end-to-end workflow: generate → sign → register
- ✅ All statements registered to transparency log
- ✅ Receipts created for all registrations
- ✅ Tile creation verified
- ✅ Independently testable with pre-signed feed directory
- ✅ Delivers full feature value

---

## Phase 5: Polish & Integration

---

### T027: Add CLI Help Text and Examples
**Type**: Documentation
**Priority**: Polish
**Estimated Effort**: 1 hour
**Dependencies**: T019, T022, T025

**Description**: Add comprehensive help text and usage examples to CLI command.

**Tasks**:
1. Add long description to `feed` command with feature overview
2. Add usage examples to help text
3. Document flags: `--no-sign`, `--no-register`
4. Add troubleshooting section
5. Test help output

**Test Requirements**:
```go
func TestFeedCommandHelp(t *testing.T) {
    t.Parallel()

    cmd := exec.Command("./scitt", "feed", "--help")
    output, err := cmd.CombinedOutput()
    require.NoError(t, err)

    assert.Contains(t, string(output), "Synthetic Supply Chain Feed Generator")
    assert.Contains(t, string(output), "--no-sign")
    assert.Contains(t, string(output), "--no-register")
    assert.Contains(t, string(output), "Examples:")
}

func TestFeedGenerateHelp(t *testing.T) {
    t.Parallel()

    cmd := exec.Command("./scitt", "feed", "generate", "--help")
    output, err := cmd.CombinedOutput()
    require.NoError(t, err)

    assert.Contains(t, string(output), "Generate synthetic supply chain dataset")
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] `./scitt feed --help` displays feature overview
- [ ] `./scitt feed generate --help` displays detailed usage
- [ ] Help text includes examples
- [ ] Flags documented with descriptions
- [ ] Help text mentions required environment variables

**Files Modified**:
- `internal/cli/feed.go`

---

### T028: End-to-End Integration Test
**Type**: Integration Test
**Priority**: Polish
**Estimated Effort**: 2 hours
**Dependencies**: T019, T022, T025

**Description**: Comprehensive integration test for full workflow with running SCITT service.

**Tasks**:
1. Create `tests/integration/feed_e2e_test.go`
2. Test full workflow: generate → sign → register
3. Verify all success criteria from spec.md
4. Test with real SCITT service
5. Verify tile creation in storage
6. Clean up test data

**Test Requirements** (WRITE FIRST):
```go
func TestFeedE2EWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E integration test")
    }

    // Prerequisites: SCITT service running, API key set
    serviceURL := "http://127.0.0.1:56177"
    apiKey := os.Getenv("SCITT_API_KEY")
    require.NotEmpty(t, apiKey)

    // Verify service is running
    resp, err := http.Get(serviceURL + "/.well-known/scitt-configuration")
    require.NoError(t, err)
    require.Equal(t, 200, resp.StatusCode)

    workDir := t.TempDir()

    // Run full workflow with simulated inputs
    cmd := exec.Command("./scitt", "feed", "generate")
    cmd.Dir = workDir
    cmd.Env = append(os.Environ(), "SCITT_API_KEY="+apiKey)

    // Simulate user inputs: yes to sign, yes to register, default service URL
    stdinPipe, err := cmd.StdinPipe()
    require.NoError(t, err)

    go func() {
        defer stdinPipe.Close()
        fmt.Fprintln(stdinPipe, "yes")  // Sign?
        fmt.Fprintln(stdinPipe, "yes")  // Register?
        fmt.Fprintln(stdinPipe, "")     // Service URL (default)
    }()

    output, err := cmd.CombinedOutput()
    require.NoError(t, err, "Output: %s", string(output))

    // Verify success messages
    assert.Contains(t, string(output), "Feed generated successfully")
    assert.Contains(t, string(output), "All documents signed successfully")
    assert.Contains(t, string(output), "Registration complete!")

    // Extract feed directory
    re := regexp.MustCompile(`feed-\d{4}-\d{2}-\d{2}-\d{6}`)
    feedDir := re.FindString(string(output))
    require.NotEmpty(t, feedDir)

    fullPath := filepath.Join(workDir, feedDir)

    // Verify document count >= 96
    docCount := 0
    cbor Count := 0
    receiptCount := 0

    filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
        if d.IsDir() {
            return nil
        }
        if strings.HasSuffix(path, ".json") && !strings.Contains(path, "metadata") {
            docCount++
        }
        if strings.HasSuffix(path, ".cbor") && !strings.Contains(path, "receipt") {
            cborCount++
        }
        if strings.HasSuffix(path, ".receipt.cbor") {
            receiptCount++
        }
        return nil
    })

    assert.GreaterOrEqual(t, docCount, 96, "Should have at least 96 JSON documents")
    assert.Equal(t, docCount, cborCount, "All documents should be signed")
    assert.Equal(t, docCount, receiptCount, "All statements should have receipts")

    // Verify 3+ tiles mentioned in output
    assert.Contains(t, string(output), "Tiles created:")

    // Parse tree size from output
    treeRe := regexp.MustCompile(`Tree size: (\d+)`)
    matches := treeRe.FindStringSubmatch(string(output))
    require.Len(t, matches, 2)

    treeSize, err := strconv.Atoi(matches[1])
    require.NoError(t, err)
    assert.GreaterOrEqual(t, treeSize, 96)

    expectedTiles := (treeSize + 31) / 32
    assert.GreaterOrEqual(t, expectedTiles, 3)
}

func TestFeedE2ECompanyDistribution(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E integration test")
    }

    // Test that document distribution matches spec:
    // Foundry: ~30 docs (wafer, mineral, logistics)
    // IDM: ~36 docs (chip, firmware, SBOM)
    // Fabless: ~24 docs (memory, AI, CVE)
}

func TestFeedE2EURNFormat(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E integration test")
    }

    // Verify all documents have correct URN format
    // Parse all JSON files and validate subject_urn field
}

func TestFeedE2ESPDXCompliance(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E integration test")
    }

    // Verify SBOM documents are SPDX 2.3 compliant
    // Check for required fields: spdxVersion, dataLicense, etc.
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] Full workflow executes successfully with real service
- [ ] All success criteria from spec.md verified
- [ ] Document count >= 96
- [ ] All documents signed and registered
- [ ] Receipts created for all statements
- [ ] Tile count >= 3
- [ ] URN format validation passes
- [ ] SPDX compliance validation passes

**Files Modified**:
- `tests/integration/feed_e2e_test.go` (new)

---

### T029: Error Message Improvements
**Type**: Polish
**Priority**: Nice-to-have
**Estimated Effort**: 1 hour
**Dependencies**: T019, T024

**Description**: Enhance error messages with actionable remediation steps (per SC-014).

**Tasks**:
1. Review all error returns in feed command
2. Add context and remediation steps to errors
3. Test error scenarios
4. Verify error messages are user-friendly

**Test Requirements**:
```go
func TestErrorMessageServiceUnavailable(t *testing.T) {
    // Simulate service unavailable
    // Verify error message includes remediation
    err := cli.RegisterStatements("test", "http://localhost:99999", "key")
    require.Error(t, err)
    assert.Contains(t, err.Error(), "Cannot connect to SCITT service")
    assert.Contains(t, err.Error(), "ensure service is running")
}

func TestErrorMessageInvalidAPIKey(t *testing.T) {
    // Simulate 401 error
    // Verify error message mentions API key
    assert.Contains(t, err.Error(), "API key is invalid")
    assert.Contains(t, err.Error(), "SCITT_API_KEY")
}

func TestErrorMessageInsufficientDiskSpace(t *testing.T) {
    // Verify disk space check error message
    assert.Contains(t, err.Error(), "Insufficient disk space")
    assert.Contains(t, err.Error(), "Need ~5MB")
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] All error messages include remediation steps
- [ ] Service connection errors mention service startup
- [ ] API key errors mention environment variable
- [ ] Disk space errors show required/available space
- [ ] Signing errors mention key path

**Files Modified**:
- `internal/cli/feed.go`
- `internal/cli/feed_test.go`

---

### T030: Performance Verification
**Type**: Validation
**Priority**: Nice-to-have
**Estimated Effort**: 30 minutes
**Dependencies**: T028

**Description**: Verify performance goals from spec.md (SC-001, SC-007).

**Tasks**:
1. Add benchmark test for document generation
2. Verify generation completes in <10 seconds
3. Add benchmark for full workflow
4. Verify full workflow <2 minutes
5. Document performance results

**Test Requirements**:
```go
func BenchmarkDocumentGeneration(b *testing.B) {
    for i := 0; i < b.N; i++ {
        feedDir := b.TempDir()
        // Run generation only (no sign/register)
        cmd := exec.Command("./scitt", "feed", "generate", "--no-sign", "--no-register")
        cmd.Dir = feedDir

        start := time.Now()
        err := cmd.Run()
        elapsed := time.Since(start)

        require.NoError(b, err)
        assert.Less(b, elapsed, 10*time.Second, "Generation should complete in <10s")
    }
}

func BenchmarkFullWorkflow(b *testing.B) {
    if testing.Short() {
        b.Skip("Skipping benchmark requiring SCITT service")
    }

    // Benchmark full workflow with service
    // Verify <2 minutes for 100+ documents
}
```

**Acceptance Criteria**:
- [ ] Document generation <10 seconds
- [ ] Full workflow <2 minutes (with service)
- [ ] Performance goals documented

**Files Modified**:
- `internal/cli/feed_test.go`

---

## Dependency Graph

```
Phase 1: Setup
T001 (Structure) ─┬─→ T003 (PRNG)
                  ├─→ T004 (Companies)
                  ├─→ T005 (URNs)
                  └─→ T006 (Schemas)
T002 (Progress Bar)

Phase 2: US1 (P1)
T003, T004, T005, T006 ─┬─→ T007 (Wafer)
                        ├─→ T008 (Mineral)
                        ├─→ T009 (Chip)
                        ├─→ T010 (Firmware)
                        ├─→ T011 (SBOM)
                        ├─→ T012 (Memory)
                        ├─→ T013 (AI Dataset)
                        ├─→ T014 (AI Model)
                        ├─→ T015 (CVE)
                        └─→ T016 (Logistics)

T001, T004 ───→ T017 (Feed Directory)
T007-T016 ────→ T018 (Serialization)
T001, T002, T017, T018 ───→ T019 (CLI Wiring)

Phase 3: US2 (P2)
T019 ───→ T020 (Key Generation)
T020 ───→ T021 (Signing)
T021 ───→ T022 (Sign Prompt)

Phase 4: US3 (P3)
T019 ───→ T023 (Service Validation)
T023 ───→ T024 (Registration)
T024 ───→ T025 (Register Prompt)
T024 ───→ T026 (Tile Verification)

Phase 5: Polish
T019, T022, T025 ───→ T027 (Help Text)
T019, T022, T025 ───→ T028 (E2E Test)
T019, T024 ───→ T029 (Error Messages)
T028 ───→ T030 (Performance)
```

---

## Parallel Execution Guide

Tasks marked `[P]` can run in parallel within their dependency group:

**Group 1** (after T001, T002):
- T003, T004, T005, T006 can run in parallel

**Group 2** (after T003, T004, T005, T006):
- T007-T016 (all document generators) can run in parallel

**Group 3** (after T018):
- T017 and T019 are sequential

**Group 4** (after T019):
- T020, T023 can run in parallel (different US branches)

**Group 5** (polish):
- T027, T029 can run in parallel
- T028 (E2E) should run after T022, T025
- T030 runs after T028

---

## Task Summary

**Total Tasks**: 30
- Setup: 2 tasks
- US1 (P1): 17 tasks (document generation + CLI)
- US2 (P2): 3 tasks (signing workflow)
- US3 (P3): 4 tasks (registration workflow)
- Polish: 4 tasks (help, tests, error handling, performance)

**Estimated Total Effort**: ~40-45 hours

**Parallelization Opportunities**:
- 10 document generators can run in parallel (T007-T016)
- 4 foundational tasks can run in parallel (T003-T006)
- Total parallelizable effort: ~15 hours

**Critical Path** (sequential): T001 → T019 → T022 → T025 → T028 (~10-12 hours)

---

## Implementation Notes

1. **TDD Discipline**: All implementation tasks include "Write tests FIRST" requirement per Constitution Principle III
2. **Command Reuse**: Tasks T020, T021, T024 use `exec.Command()` to call existing CLI commands per research.md § 5
3. **Determinism**: Tasks T003, T007-T016 implement deterministic generation using seeded PRNG
4. **Progress Bars**: Tasks T019, T021, T024 use `github.com/schollz/progressbar/v3` library
5. **Integration Tests**: Task T028 requires running SCITT service (can run with `go test -short` to skip)
6. **Error Handling**: Task T029 ensures all errors follow Constitution Principle V (Observability)

---

**Ready for Implementation**: All tasks defined with clear acceptance criteria, test requirements, and dependencies.
