# Implementation Plan: Synthetic Supply Chain Feed Generator

**Branch**: `004-add-a-cli` | **Date**: 2025-10-18 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-add-a-cli/spec.md`

**Note**: This plan follows the constitution-mandated workflow with Test-First Development (Principle III) and Go as source of truth for cryptography (Principle VIII).

## Summary

Add `./scitt feed generate` command to the Go CLI that produces synthetic supply chain data for an AI-capable laptop from 3 semiconductor companies (foundry, IDM, fabless). The command generates 90-110 documents across 10 categories (wafers, minerals, chips, firmware, SBOMs, memory, AI datasets/models, CVEs, logistics), creates ES256 key pairs for each company, provides interactive workflows for signing and registration, and ensures at least 3 complete tiles (96+ entries) are created in transparency storage.

**Technical Approach**: Extend existing Go CLI (`cmd/scitt/main.go`) with new `feed` subcommand using Cobra framework. Implement deterministic document generation using timestamp-seeded PRNG, leverage existing COSE libraries for ES256 signing, and integrate with existing SCITT service infrastructure for registration.

## Technical Context

**Language/Version**: Go 1.24 (existing project standard)
**Primary Dependencies**:
- `github.com/spf13/cobra` (existing CLI framework)
- `github.com/veraison/go-cose` (existing COSE/ES256 signing)
- Standard library: `encoding/json`, `crypto/rand`, `crypto/sha256`, `math/rand` (seeded)

**Storage**: File system (JSON documents, CBOR keys/statements/receipts)
**Testing**: Go standard `testing` package with TDD methodology
**Target Platform**: macOS/Linux/Windows CLI (cross-platform Go binary)
**Project Type**: Single (CLI extension within existing scitt-golang)
**Performance Goals**:
- Generate 100+ documents in <10 seconds
- Complete full workflow (generate + sign + register) in <2 minutes

**Constraints**:
- Documents must be deterministic (timestamp-seeded randomness)
- ES256 signatures non-deterministic (random nonces per ECDSA spec)
- Progress feedback via progress bars only (no persistent logs per clarification)
- No resume/retry capability (fail-fast, user re-runs entire command)

**Scale/Scope**:
- 90-110 documents per feed directory
- 3 company identities with ES256 key pairs
- 10 document categories with role-based distribution
- 3-4 tiles generated per complete registration

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: Transparency by Design
✅ **PASS** - All generated data includes provenance metadata (issuer URNs, content locations, timestamps). Feed generation is fully traceable via directory structure and URN identifiers.

### Principle II: Verifiable Audit Trails
✅ **PASS** - COSE signatures create verifiable audit trail. Transparency service provides registration receipts with Merkle proofs. All signed statements are tamper-evident.

### Principle III: Test-First Development (NON-NEGOTIABLE)
✅ **PASS** - TDD workflow will be followed:
1. Write tests for document generation (verify counts, schema, URNs)
2. Write tests for key generation (ES256/P-256, CBOR format)
3. Write tests for signing workflow (COSE Sign1 structure)
4. Write tests for registration workflow (service integration)
5. All tests must fail before implementation begins

### Principle IV: API-First Architecture
✅ **PASS** - This is a CLI-only feature (out of scope: API endpoints per spec). CLI uses existing SCITT service APIs (`/entries`, `/.well-known/scitt-configuration`). No new service APIs required.

### Principle V: Observability and Monitoring
✅ **PASS** - Progress feedback provided via progress bars (per clarification). Error messages are actionable with remediation steps (per FR/SC-014). CLI uses structured error handling.

### Principle VI: Data Integrity and Versioning
✅ **PASS** - Generated documents use SPDX 2.3+ (versioned schema). COSE statements use RFC-compliant formats. Feed directories are timestamped for version tracking.

### Principle VII: Simplicity and Maintainability
✅ **PASS** - Uses existing CLI patterns (Cobra), existing COSE libraries, standard library randomness. No new external dependencies. Document generation logic is straightforward template-based generation.

### Principle VIII: Go Interoperability as Source of Truth (NON-NEGOTIABLE)
✅ **PASS** - This is a Go-native feature. Uses existing Go COSE library (`github.com/veraison/go-cose`) which is already validated for interoperability. ES256 signing uses Go standard crypto libraries. No TypeScript equivalent required (CLI-only tool).

**Constitution Status**: ✅ ALL GATES PASS - Ready for Phase 0

## Project Structure

### Documentation (this feature)

```
specs/004-add-a-cli/
├── spec.md              # Feature specification with clarifications
├── plan.md              # This file
├── research.md          # Phase 0 output (document schemas, SPDX structure, PRNG patterns)
├── data-model.md        # Phase 1 output (feed directory structure, document entities)
├── quickstart.md        # Phase 1 output (user guide for feed generation workflow)
└── contracts/           # N/A for CLI-only feature
```

### Source Code (repository root: scitt-golang/)

```
cmd/scitt/
└── main.go              # Add feed subcommand registration

internal/cli/
├── feed.go              # New: feed subcommand implementation
├── feed_generate.go     # New: document generation logic
├── feed_sign.go         # New: signing workflow
├── feed_register.go     # New: registration workflow
├── feed_test.go         # New: TDD tests for feed command
└── (existing files)

internal/generator/      # New package
├── companies.go         # Company identity definitions (foundry, IDM, fabless)
├── documents.go         # Document generation (wafer, mineral, chip, etc.)
├── schema.go            # JSON schemas for each document type
├── urns.go              # URN generation utilities
├── seeded_rand.go       # Deterministic PRNG wrapper
├── generator_test.go    # TDD tests
└── fixtures_test.go     # Test fixtures for validation

pkg/cose/
└── (existing files)     # Use existing ES256 signing capabilities

tests/integration/
└── feed_e2e_test.go     # New: End-to-end test for generate→sign→register workflow
```

**Structure Decision**: Extends existing `scitt-golang` single-project structure. New `internal/generator` package isolates document generation logic. CLI commands in `internal/cli` follow existing patterns (`service.go`, `statement.go`, `issuer.go`).

## Complexity Tracking

*No constitution violations requiring justification*

All constitutional gates pass. This feature:
- Extends existing CLI naturally (no architectural changes)
- Uses established Go patterns and libraries
- Maintains test-first discipline
- Follows simplicity principles (no clever abstractions)
- Leverages existing COSE/signing infrastructure

No complexity justification needed.

---

## Phase 0: Outline & Research

**Prerequisites**: Constitution Check passed ✅

### Research Tasks

1. **SPDX 2.3 Schema Structure**
   - Task: Research SPDX 2.3 JSON schema for SBOM/HBOM documents
   - Why: FR-012 requires SPDX format; need canonical schema
   - Output: Document structure in research.md with example

2. **Real CVE/CWE Examples**
   - Task: Validate real CVE IDs mentioned in spec (CVE-2024-0519, CVE-2024-3660, CVE-2024-27351)
   - Why: Ensure realism; avoid hallucinated CVE IDs
   - Output: Confirmed CVE details and additional realistic examples for AI laptops

3. **Deterministic PRNG Patterns**
   - Task: Research Go `math/rand` seeded generation for deterministic document IDs/content
   - Why: FR-025 requires timestamp-seeded determinism
   - Output: Code pattern for seeded random generation

4. **Progress Bar Libraries**
   - Task: Evaluate Go progress bar libraries (e.g., `github.com/schollz/progressbar`, `github.com/cheggaaa/pb`)
   - Why: Clarification specifies progress bars only (no logs)
   - Output: Recommended library with rationale

5. **COSE ES256 Best Practices**
   - Task: Review Go COSE library ES256 usage patterns in existing codebase
   - Why: Ensure consistency with existing signing patterns
   - Output: Code examples from `internal/cli/statement.go` or similar

### Research Workflow

Will dispatch 5 research agents in parallel to resolve unknowns:

```
Agent 1: SPDX Schema → research.md § SPDX Structure
Agent 2: CVE Validation → research.md § Real CVE Examples
Agent 3: Seeded PRNG → research.md § Deterministic Generation
Agent 4: Progress Bars → research.md § Progress Feedback
Agent 5: COSE Patterns → research.md § ES256 Signing
```

**Expected Output**: `research.md` with 5 sections answering all "NEEDS CLARIFICATION" items

---

## Phase 1: Design & Contracts

**Prerequisites**: `research.md` complete

### Deliverables

1. **data-model.md**: Document entity structures
   - Feed Directory (fields: timestamp, path, seed)
   - Company Identity (fields: name, role, URN, keys_path)
   - Supply Chain Document (fields: URN, type, content, metadata)
   - Document relationships (graph structure)
   - File system layout

2. **quickstart.md**: User guide
   - Installation/setup
   - Basic usage: `./scitt feed generate`
   - Interactive prompts walkthrough
   - Example output
   - Troubleshooting common errors

3. **contracts/**: N/A (CLI-only, no new service APIs)

### Agent Context Update

After Phase 1 artifacts complete:
```bash
.specify/scripts/bash/update-agent-context.sh claude
```

This will update `.specify/memory/claude-context.md` with new technologies/patterns from this plan.

---

## Phase 2: Implementation (Not Created by /speckit.plan)

**Note**: Phase 2 (tasks.md generation) is handled by `/speckit.tasks` command, not this plan command.

Implementation will follow TDD discipline per Constitution Principle III:
1. Tests first (fail)
2. Minimal implementation (pass)
3. Refactor
4. Repeat

Expected task count: ~15-20 implementation tasks covering:
- Document generation (8 tasks: one per major document category)
- Key generation (1 task)
- Signing workflow (2 tasks: signing logic + interactive prompts)
- Registration workflow (2 tasks: API integration + progress bars)
- CLI wiring (1 task: Cobra subcommand setup)
- Integration tests (1 task: end-to-end workflow)
- Documentation (1 task: help text, examples)

---

## Success Criteria

This plan is complete when:

- ✅ Constitution Check passes (all 8 principles)
- ✅ Technical Context fully specified (no NEEDS CLARIFICATION)
- ✅ Phase 0: research.md created with all research tasks resolved
- ✅ Phase 1: data-model.md created with entity definitions
- ✅ Phase 1: quickstart.md created with user guide
- ✅ Phase 1: Agent context updated

**Planning Complete**: All deliverables created. Ready for `/speckit.tasks` to generate implementation tasks.
