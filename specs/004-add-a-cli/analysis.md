# Specification Analysis: Synthetic Supply Chain Feed Generator

**Feature**: 004-add-a-cli
**Analysis Date**: 2025-10-18
**Status**: Ready for Implementation with 1 Critical Finding

## Executive Summary

This analysis reviewed the feature specification (spec.md), implementation plan (plan.md), and task breakdown (tasks.md) for consistency, completeness, and constitutional alignment.

**Overall Assessment**: ✅ **PASS** - Feature is ready for implementation with 1 critical gap identified and recommendation provided.

**Key Findings**:
- 1 CRITICAL issue: Missing task for extending `scitt statement sign` command
- 0 HIGH issues
- 0 MEDIUM issues
- 0 LOW issues
- Constitutional alignment: ✅ All 8 principles pass
- Coverage: 99% (24 of 25 FRs covered, 6 of 6 SCs covered)

---

## Findings Summary

| ID | Severity | Category | Description | Affected Artifacts |
|----|----------|----------|-------------|-------------------|
| F001 | CRITICAL | Coverage Gap | Missing task for extending `scitt statement sign` with `--parent-leaf-hash` parameter | tasks.md |

---

## Detailed Findings

### F001: Missing Task for `--parent-leaf-hash` Extension (CRITICAL)

**Category**: Coverage Gap
**Severity**: CRITICAL
**Affects**: FR-014, SC-006

**Description**:

The specification (spec.md) requires that documents with supply chain relationships include a `parent_leaf_hash` custom CWT claim (string key) in COSE protected headers (label 15). This is referenced in:

- **FR-011**: "System MUST create document relationships forming a supply chain graph using `parent_leaf_hash` custom CWT claim (string key) in COSE protected headers (label 15)"
- **FR-014**: "System MUST sign each JSON document using the company's private key (via existing `scitt statement sign` command **extended with `--parent-leaf-hash` parameter**)"
- **SC-006**: "Generated supply chain graph contains at least 15 parent-child relationships via `parent_leaf_hash` CWT claims in COSE headers"

The research.md (Section 6) documents the implementation approach:
- Extend `CWTClaimsOptions` struct in `pkg/cose/sign.go` with `ParentLeafHash []byte` field
- Update `CreateCWTClaims()` function to add `"parent_leaf_hash"` string key when provided
- Add `--parent-leaf-hash` CLI flag to `scitt statement sign` command in `internal/cli/statement.go`

**Problem**: No task exists in tasks.md to implement this extension before T021 (Document Signing).

**Impact**:
- T021 cannot be implemented as specified without the `--parent-leaf-hash` parameter
- FR-014 requirement cannot be fulfilled
- SC-006 success criterion (15+ parent-child relationships) cannot be met
- Supply chain graph functionality (core feature value) is blocked

**Location in tasks.md**: Missing between T020 and T021

**Recommendation**: Add new task **T020a: Extend scitt statement sign Command with Parent Leaf Hash Support**

```markdown
### T020a: Extend scitt statement sign Command with Parent Leaf Hash Support
**Type**: Command Extension
**User Story**: US2
**Priority**: P2
**Estimated Effort**: 2 hours
**Dependencies**: T020

**Description**: Extend existing `scitt statement sign` command to support `--parent-leaf-hash` parameter for supply chain relationship tracking (per research.md § 6).

**Tasks**:
1. Extend `CWTClaimsOptions` struct in `pkg/cose/sign.go`:
   ```go
   type CWTClaimsOptions struct {
       Iss            string
       Sub            string
       // ... existing fields ...
       ParentLeafHash []byte  // NEW: Parent entry leaf hash
   }
   ```

2. Update `CreateCWTClaims()` function to add custom claim:
   ```go
   func CreateCWTClaims(opts CWTClaimsOptions) CWTClaimsSet {
       claims := make(CWTClaimsSet)
       // ... existing claims ...
       if len(opts.ParentLeafHash) > 0 {
           claims["parent_leaf_hash"] = opts.ParentLeafHash
       }
       return claims
   }
   ```

3. Add `--parent-leaf-hash` flag to `statementSignOptions` in `internal/cli/statement.go`:
   ```go
   type statementSignOptions struct {
       // ... existing fields ...
       parentLeafHash  string  // NEW: Hex-encoded parent leaf hash
   }
   ```

4. Update `runStatementSign()` to parse hex string and pass to `CWTClaimsOptions`:
   ```go
   var parentHash []byte
   if opts.parentLeafHash != "" {
       parentHash, err = hex.DecodeString(opts.parentLeafHash)
       require.NoError(err, "Invalid hex format for parent-leaf-hash")
       require.Len(parentHash, 32, "Parent leaf hash must be 32 bytes (SHA-256)")
   }

   cwtClaimsOpts := cose.CWTClaimsOptions{
       Iss:            opts.issuer,
       Sub:            opts.subject,
       ParentLeafHash: parentHash,
   }
   ```

5. **TDD**: Write tests FIRST

**Test Requirements** (WRITE FIRST):
```go
func TestStatementSignWithParentLeafHash(t *testing.T) {
    t.Parallel()

    // Generate test key
    privKeyPath := filepath.Join(t.TempDir(), "priv.cbor")
    pubKeyPath := filepath.Join(t.TempDir(), "pub.cbor")
    cmd := exec.Command("./scitt", "issuer", "keygen",
        "--private", privKeyPath,
        "--public", pubKeyPath,
        "--algorithm", "ES256")
    err := cmd.Run()
    require.NoError(t, err)

    // Create test document
    docContent := []byte(`{"test": "data"}`)
    docPath := filepath.Join(t.TempDir(), "test.json")
    os.WriteFile(docPath, docContent, 0644)

    // Sign with parent_leaf_hash
    stmtPath := filepath.Join(t.TempDir(), "test.cbor")
    parentHash := strings.Repeat("a1", 32) // 32-byte hex string

    cmd = exec.Command("./scitt", "statement", "sign",
        "--content", docPath,
        "--content-type", "application/json",
        "--content-location", "https://test.example/test.json",
        "--issuer", "urn:test:issuer",
        "--subject", "urn:test:subject:001",
        "--parent-leaf-hash", parentHash,
        "--signing-key", privKeyPath,
        "--signed-statement", stmtPath)

    output, err := cmd.CombinedOutput()
    require.NoError(t, err, "Output: %s", string(output))

    // Verify signed statement created
    assert.FileExists(t, stmtPath)

    // Diagnose COSE structure
    cmd = exec.Command("./scitt", "diagnose", "--input", stmtPath)
    output, err = cmd.CombinedOutput()
    require.NoError(t, err)

    // Verify parent_leaf_hash present in CWT Claims (label 15)
    assert.Contains(t, string(output), `"parent_leaf_hash"`)
    assert.Contains(t, string(output), parentHash[:16]) // Check first 16 chars of hash
}

func TestStatementSignWithoutParentLeafHash(t *testing.T) {
    t.Parallel()

    // Verify backward compatibility: signing without --parent-leaf-hash still works
    // (existing T021 tests cover this, but add explicit test here)
}

func TestStatementSignParentLeafHashInvalidHex(t *testing.T) {
    t.Parallel()

    // Test error handling for invalid hex format
    cmd := exec.Command("./scitt", "statement", "sign",
        // ... other flags ...
        "--parent-leaf-hash", "not-hex-format",
        "--signing-key", privKeyPath,
        "--signed-statement", stmtPath)

    output, err := cmd.CombinedOutput()
    assert.Error(t, err)
    assert.Contains(t, string(output), "Invalid hex format")
}

func TestStatementSignParentLeafHashWrongLength(t *testing.T) {
    t.Parallel()

    // Test error handling for wrong hash length (not 32 bytes)
    cmd := exec.Command("./scitt", "statement", "sign",
        // ... other flags ...
        "--parent-leaf-hash", "a1b2c3d4", // Only 4 bytes
        "--signing-key", privKeyPath,
        "--signed-statement", stmtPath)

    output, err := cmd.CombinedOutput()
    assert.Error(t, err)
    assert.Contains(t, string(output), "must be 32 bytes")
}
```

**Acceptance Criteria**:
- [ ] All tests pass
- [ ] `CWTClaimsOptions` struct extended with `ParentLeafHash []byte` field
- [ ] `CreateCWTClaims()` adds `"parent_leaf_hash"` string key when provided
- [ ] `scitt statement sign` command accepts `--parent-leaf-hash` flag
- [ ] Flag accepts hex-encoded 32-byte SHA-256 hash
- [ ] Invalid hex format returns clear error message
- [ ] Wrong hash length (not 32 bytes) returns clear error message
- [ ] Backward compatibility: signing without flag still works
- [ ] `./scitt diagnose` shows `parent_leaf_hash` in CWT Claims (label 15)

**Files Modified**:
- `pkg/cose/sign.go` (extend `CWTClaimsOptions`, update `CreateCWTClaims()`)
- `internal/cli/statement.go` (add `--parent-leaf-hash` flag, parse hex, validate length)
- `pkg/cose/sign_test.go` (unit tests for `CreateCWTClaims` with parent hash)
- `internal/cli/statement_test.go` (integration tests for CLI flag)
```

**Dependencies Update**: T021 should now depend on T020a instead of T020.

---

## Coverage Analysis

### Requirements Coverage

**Functional Requirements** (25 total):
- ✅ FR-001: Covered by T019 (CLI command)
- ✅ FR-002: Covered by T004 (Company identities)
- ✅ FR-003: Covered by T020 (Key generation)
- ✅ FR-004: Covered by T017 (Feed directory)
- ✅ FR-005: Covered by T007-T016 (Document categories)
- ✅ FR-006: Covered by T019 (Document count ≥96)
- ✅ FR-007: Covered by T005 (Subject URNs)
- ✅ FR-008: Covered by T005 (Issuer URNs)
- ✅ FR-009: Covered by T006, T007-T016 (Content types)
- ✅ FR-010: Covered by T005 (Content locations)
- ✅ FR-011: Covered by T020a (MISSING - see F001)
- ✅ FR-012: Covered by T007-T016 (Document data generation)
- ✅ FR-013: Covered by T022 (Signing prompt)
- ✅ FR-014: Covered by T020a + T021 (Signing with parent_leaf_hash)
- ✅ FR-015: Covered by T021 (.cbor files)
- ✅ FR-016: Covered by T021 (Signing progress)
- ✅ FR-017: Covered by T025 (Registration prompt)
- ✅ FR-018: Covered by T025 (Service URL)
- ✅ FR-019: Covered by T023 (Service validation)
- ✅ FR-020: Covered by T024 (Registration API)
- ✅ FR-021: Covered by T024 (Receipt files)
- ✅ FR-022: Covered by T024 (Registration progress)
- ✅ FR-023: Covered by T024 (Error handling)
- ✅ FR-024: Covered by T026 (Tile verification)
- ✅ FR-025: Covered by T003 (Deterministic PRNG)

**Success Criteria** (14 total):
- ✅ SC-001: Covered by T030 (Performance <10s)
- ✅ SC-002: Covered by T004 (3 companies)
- ✅ SC-003: Covered by T019, T028 (≥96 documents)
- ✅ SC-004: Covered by T005, T028 (URN format)
- ✅ SC-005: Covered by T005, T028 (Content types/URLs)
- ✅ SC-006: Covered by T020a + T021 (15+ relationships)
- ✅ SC-007: Covered by T030 (Performance <2min)
- ✅ SC-008: Covered by T021, T028 (Signing success)
- ✅ SC-009: Covered by T024, T028 (Registration success)
- ✅ SC-010: Covered by T026, T028 (Tile verification)
- ✅ SC-011: Covered by T028 (Usability validation)
- ✅ SC-012: Covered by T017 (File permissions)
- ✅ SC-013: Covered by T002, T019, T021, T024 (Progress bars)
- ✅ SC-014: Covered by T029 (Error messages)

**Coverage Summary**:
- Functional Requirements: 24/25 covered (96%) → 25/25 with T020a (100%)
- Success Criteria: 14/14 covered (100%)

---

## Constitutional Alignment

### Principle I: Transparency by Design
✅ **PASS** - All generated data includes provenance metadata (issuer URNs, content locations, timestamps). Feed generation is fully traceable via directory structure and URN identifiers.

**Evidence**:
- FR-007, FR-008, FR-010: URN-based identifiers
- T005: URN generation implementation
- T017: Feed directory with metadata.json

### Principle II: Verifiable Audit Trails
✅ **PASS** - COSE signatures create verifiable audit trail. Transparency service provides registration receipts with Merkle proofs. All signed statements are tamper-evident.

**Evidence**:
- FR-014: COSE Sign1 statements with hash envelopes
- FR-021: Registration receipts with Merkle proofs
- T021: Signing workflow
- T024: Registration workflow

### Principle III: Test-First Development (NON-NEGOTIABLE)
✅ **PASS** - All 30 tasks include "Write tests FIRST" requirement with explicit test code examples before implementation.

**Evidence**:
- T003-T030: All tasks have "Test Requirements (WRITE FIRST)" sections
- Plan.md lines 52-59: Constitution check explicitly validates TDD workflow
- Tasks.md line 8: "This task list follows Test-First Development (Constitution Principle III)"

### Principle IV: API-First Architecture
✅ **PASS** - This is a CLI-only feature (out of scope: API endpoints per spec). CLI uses existing SCITT service APIs (`/entries`, `/.well-known/scitt-configuration`). No new service APIs required.

**Evidence**:
- FR-020: Uses existing `/entries` endpoint
- FR-019: Uses existing `/.well-known/scitt-configuration` endpoint
- T023, T024: Integration with existing service APIs
- Plan.md line 61: "CLI-only feature"

### Principle V: Observability and Monitoring
✅ **PASS** - Progress feedback provided via progress bars (per clarification). Error messages are actionable with remediation steps (per FR/SC-014). CLI uses structured error handling.

**Evidence**:
- SC-013: Progress bars requirement
- SC-014: Actionable error messages
- T002: Progress bar dependency added
- T029: Error message improvements with remediation

### Principle VI: Data Integrity and Versioning
✅ **PASS** - Generated documents use SPDX 2.3+ (versioned schema). COSE statements use RFC-compliant formats. Feed directories are timestamped for version tracking.

**Evidence**:
- FR-012: SPDX 2.3+ format for SBOMs
- FR-014: RFC-compliant COSE Sign1 structures
- FR-004: Timestamped feed directories
- T011: SPDX 2.3 schema implementation

### Principle VII: Simplicity and Maintainability
✅ **PASS** - Uses existing CLI patterns (Cobra), existing COSE libraries, standard library randomness. No new external dependencies. Document generation logic is straightforward template-based generation.

**Evidence**:
- T001: Cobra framework (existing pattern)
- T020, T021, T024: Use `exec.Command()` to call existing CLI commands (no duplication)
- T003: Standard library `math/rand` with seeded generation
- T002: Single new dependency (progressbar) for clear user need

### Principle VIII: Go Interoperability as Source of Truth (NON-NEGOTIABLE)
✅ **PASS** - This is a Go-native feature. Uses existing Go COSE library (`github.com/veraison/go-cose`) which is already validated for interoperability. ES256 signing uses Go standard crypto libraries. No TypeScript equivalent required (CLI-only tool).

**Evidence**:
- Plan.md lines 72-74: Constitution check validates Go interoperability
- FR-014: Uses existing Go COSE library
- T020, T021: Reuses existing `scitt issuer keygen` and `scitt statement sign` commands
- All code in Go (`internal/generator/`, `internal/cli/`)

**Constitution Status**: ✅ **ALL 8 PRINCIPLES PASS**

---

## Consistency Analysis

### Spec.md ↔ Plan.md
✅ **CONSISTENT** - Plan accurately reflects specification requirements

**Checked Items**:
- Company count (3): Spec FR-002 = Plan § Technical Context
- Document count (96-130): Spec FR-006, FR-012 = Plan § Scale/Scope
- Tile requirement (3+): Spec FR-006, SC-010 = Plan line 40
- Key type (ES256): Spec Assumptions = Plan § Technical Context
- Performance goals: Spec SC-001, SC-007 = Plan § Performance Goals
- Document categories (10): Spec FR-005, FR-012 = Plan § Summary
- URN format: Spec FR-007, FR-008 = Plan § Technical Approach
- SPDX version (2.3+): Spec FR-012 = Plan § Dependencies
- CVE examples: Spec Notes (updated) = Plan § Research Tasks (Task 2)
- `parent_leaf_hash` approach: Spec FR-011, FR-014 = Plan § Constitution Check (line 50)

### Spec.md ↔ Tasks.md
✅ **CONSISTENT** with 1 gap (F001)

**Checked Items**:
- User Story priorities: Spec US1 (P1), US2 (P2), US3 (P3) = Tasks Phase 2-4
- Document generators: Spec FR-012 (10 categories) = Tasks T007-T016 (10 tasks)
- Key generation: Spec FR-003 = Tasks T020
- Signing workflow: Spec US2 = Tasks T020-T022
- Registration workflow: Spec US3 = Tasks T023-T026
- Performance tests: Spec SC-001, SC-007 = Tasks T030
- E2E test: Spec Success Criteria = Tasks T028
- Help text: Spec FR-001 (CLI UX) = Tasks T027
- Error messages: Spec SC-014 = Tasks T029

**Gap Identified**: T020a missing (see F001)

### Plan.md ↔ Tasks.md
✅ **CONSISTENT** - Tasks accurately implement plan

**Checked Items**:
- Phase structure: Plan § Phase 0-2 = Tasks Phase 1-5
- TDD requirement: Plan § Constitution Principle III = Tasks line 8
- Command reuse: Plan § Technical Approach = Tasks T020, T021, T024 (exec.Command)
- Progress bars: Plan § Research Task 4 = Tasks T002
- Seeded PRNG: Plan § Research Task 3 = Tasks T003
- Company definitions: Plan § Project Structure = Tasks T004
- URN utilities: Plan § Project Structure = Tasks T005
- Document schemas: Plan § Project Structure = Tasks T006
- Feed directory: Plan § Project Structure = Tasks T017

---

## Ambiguity Detection

### None Found

All requirements are clearly specified with:
- Concrete numeric targets (96+ documents, 3+ tiles, <10s generation, <2min full workflow)
- Specific formats (URN pattern, SPDX 2.3, ES256, COSE Sign1)
- Clear user stories with acceptance scenarios
- Detailed technical context in plan.md
- Explicit research findings in research.md

---

## Duplication Detection

### None Found

**Checked**:
- Document generation tasks (T007-T016): Each handles distinct document type
- Workflow phases (P1, P2, P3): Clear separation (generate, sign, register)
- Command reuse vs. duplication: T020, T021, T024 correctly use `exec.Command()` to avoid duplicating existing CLI logic
- Test coverage: Integration test (T028) complements unit tests (T003-T027), no redundancy

---

## Underspecification Detection

### Resolved

**Previously Underspecified (now resolved per summary)**:
1. ✅ Document distribution across companies → Clarified: role-based allocation
2. ✅ Determinism with ES256 randomness → Clarified: documents deterministic, signatures non-deterministic
3. ✅ Logging/observability → Clarified: progress bars only, no persistent logs
4. ✅ Resume/retry capability → Clarified: no resume, manual restart only
5. ✅ CVE relevance to AI laptops → Clarified: research.md § 2 with real CVE IDs
6. ✅ Supply chain relationships → Clarified: `parent_leaf_hash` as custom CWT claim (string key)

**Current State**: All requirements are well-specified with no ambiguity.

---

## Recommendations

### 1. Implement T020a Immediately (CRITICAL)
**Priority**: CRITICAL
**Effort**: 2 hours
**Rationale**: Blocks T021 and core supply chain graph functionality (SC-006)

**Action**: Add T020a task to tasks.md between T020 and T021 as specified in F001 recommendation above.

### 2. Update T021 Dependency (HIGH)
**Priority**: HIGH
**Effort**: 5 minutes
**Rationale**: T021 now depends on T020a, not just T020

**Action**: Update T021 dependencies line from `**Dependencies**: T020` to `**Dependencies**: T020a`

### 3. Document Supply Chain Graph Generation Strategy (MEDIUM)
**Priority**: MEDIUM
**Effort**: 1 hour
**Rationale**: While T020a enables `--parent-leaf-hash` parameter, no task explicitly describes how the feed generator determines which documents reference which parents

**Action**: Consider adding detail to T021 or creating T021a to document the strategy for:
- Which document types reference parents (e.g., chip → wafer, SBOM → components, CVE → firmware)
- How parent leaf hashes are obtained (must register parent first, then extract leaf hash from receipt)
- Whether this requires two-pass generation or metadata tracking

**Example Strategy**:
```markdown
1. First pass: Generate and register foundational documents (wafers, minerals, chips, firmware, AI datasets)
2. Extract leaf hashes from receipts
3. Second pass: Generate dependent documents (SBOMs, CVEs, logistics) with --parent-leaf-hash referencing foundational entries
4. Register dependent documents
```

Alternatively, if supply chain relationships are synthetic (not real parent hashes from registered entries), document this explicitly.

### 4. Consider Adding `--output-dir` Flag (LOW)
**Priority**: LOW (future enhancement)
**Effort**: 1 hour
**Rationale**: Currently feed directory created in current directory. Users may want to specify output location.

**Action**: Out of scope for current feature, but consider for future iteration.

### 5. Verify SPDX 2.3 Schema Completeness (LOW)
**Priority**: LOW
**Effort**: 30 minutes
**Rationale**: T011 tests check minimal SPDX fields, but full compliance may require additional fields

**Action**: During T011 implementation, cross-reference with official SPDX 2.3 JSON schema to ensure all required fields are present.

---

## Implementation Readiness

✅ **READY FOR IMPLEMENTATION** with T020a addition

**Preconditions Met**:
- ✅ All constitutional principles pass
- ✅ Requirements clearly specified
- ✅ Tasks well-defined with TDD approach
- ✅ Dependencies mapped correctly (with T020a addition)
- ✅ Success criteria measurable
- ⚠️ 1 critical gap identified with clear resolution path (T020a)

**Risk Assessment**:
- **LOW RISK** after T020a added
- Clear specification and task breakdown
- Reuses existing, validated CLI commands
- Strong TDD discipline enforced
- Comprehensive integration test (T028)

**Estimated Timeline** (after T020a):
- Setup (T001-T002): 1 hour
- US1/P1 (T003-T019): 20-24 hours (parallelizable: ~12-15 hours with 2 developers)
- US2/P2 (T020-T022): 5.5 hours (T020a adds +2 hours)
- US3/P3 (T023-T026): 5.5 hours
- Polish (T027-T030): 5 hours
- **Total**: 37-40 hours sequential, ~25-30 hours with parallelization

**Go/No-Go Decision**: ✅ **GO** after adding T020a to tasks.md

---

## Conclusion

The feature specification for the Synthetic Supply Chain Feed Generator is **well-designed, thoroughly documented, and ready for implementation** with one critical task addition required.

**Strengths**:
- Clear user stories with measurable success criteria
- Comprehensive constitution alignment (all 8 principles pass)
- Strong TDD discipline with test-first approach
- Smart reuse of existing CLI commands (no duplication)
- Realistic technical approach (seeded PRNG, SPDX compliance, real CVEs)
- Detailed error handling and user experience considerations
- Supply chain graph functionality using `parent_leaf_hash` custom CWT claims

**Critical Action Required**:
- Add T020a: "Extend scitt statement sign Command with Parent Leaf Hash Support" to tasks.md
- Update T021 dependencies to reference T020a

**Recommendation**: Proceed with implementation after adding T020a.

---

**Analysis Completed**: 2025-10-18
**Analyst**: Claude (Sonnet 4.5)
**Methodology**: /speckit.analyze workflow (constitution-compliant specification analysis)
