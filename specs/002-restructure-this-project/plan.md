# Implementation Plan: Dual-Language Project Restructure

**Branch**: `002-restructure-this-project` | **Date**: 2025-10-12 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-restructure-this-project/spec.md`

## Summary

Restructure the transparency service project into a monorepo with parallel TypeScript and Go implementations, maintaining 100% interoperability through shared test vectors and consistent API contracts. The restructure creates `scitt-typescript/` and `scitt-golang/` top-level directories with library, CLI, and server components in each, orchestrated by a shared `tests/interop/` directory. Both implementations will maintain strict lockstep versioning, identical API contracts stored in `docs/openapi.yaml`, and CLI installation via GitHub releases.

**Technical Approach**: Migrate existing TypeScript code directly to new structure, implement Go version using Go tlog as canonical reference (Constitution Principle VIII), ensure tile APIs use C2SP-compliant format, and validate via cross-implementation test suite that generates vectors from Go and verifies in TypeScript.

## Technical Context

**Language/Version**:
- TypeScript with Bun (latest stable)
- Go 1.21+ (using golang.org/x/mod/sumdb/tlog)

**Primary Dependencies**:
- TypeScript: Bun runtime, existing COSE/CWT libraries
- Go: golang.org/x/mod/sumdb/tlog, standard library crypto
- Shared: OpenAPI 3.1 for API contracts

**Storage**:
- SQLite (embedded database for both implementations)
- MinIO (S3-compatible object storage for tile storage)
- No direct S3 or Azure Blob Storage integration

**Testing**:
- TypeScript: Bun test framework
- Go: go test with standard testing package
- Cross-implementation: Shared test vectors in tests/interop/

**Target Platform**:
- macOS (Darwin arm64, x86_64)
- Linux (x86_64, arm64)
- No Windows support

**Project Type**: Monorepo with dual-language implementations

**Performance Goals**:
- Statement registration: <200ms p95 latency
- Receipt generation: <100ms p95 latency
- Tile retrieval: <50ms p95 latency
- API throughput: 1000 requests/second per implementation
- Merkle tree operations must match Go tlog performance characteristics

**Constraints**:
- 100% interoperability between implementations (Constitution Principle VIII)
- Strict lockstep versioning (both at 1.0.0, both at 1.0.1)
- Go implementation is canonical reference for cryptography and Merkle trees
- Direct migration (breaking change for TypeScript import paths)
- GitHub Actions for CI/CD
- CLI installation via GitHub releases in docs/ directory

**Scale/Scope**:
- 2 complete language implementations
- 24 functional requirements (FR-001 through FR-024)
- 5 prioritized user stories (2 P1, 2 P2, 1 P3)
- 10 measurable success criteria
- Cross-implementation test coverage for all artifact types

**Media Types** (HTTP Responses):
- Statements: `application/cose`
- Receipts: `application/cose` (COSE Sign1 with inclusion proof)
- Tile data: `application/octet-stream` (C2SP binary format)
- Checkpoint: `application/cose` (COSE Sign1)
- Service configuration: `application/cbor` (per SCRAPI spec)
- Service keys: `application/cbor` (COSE Key Set)
- Error responses: `application/concise-problem-details+cbor`

**API Contract Storage**:
- Single source of truth: `docs/openapi.yaml`
- Must be compatible with both Go and TypeScript implementations
- Includes all SCRAPI endpoints with media type specifications
- Tile API endpoints documented with C2SP format requirements

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: Transparency by Design ✅ PASS
- **Requirement**: All operations must be traceable and explainable
- **Compliance**: Cross-implementation test suite provides full traceability of interoperability; each artifact creation logged
- **Evidence**: FR-009, FR-010 mandate test validation for every user story

### Principle II: Verifiable Audit Trails ✅ PASS
- **Requirement**: State changes must create immutable, verifiable audit records
- **Compliance**: Receipts provide cryptographic proof of statement registration; Merkle tree ensures tamper-evidence
- **Evidence**: FR-006 (receipt generation/verification), existing transparency log functionality preserved

### Principle III: Test-First Development (NON-NEGOTIABLE) ✅ PASS
- **Requirement**: Tests must be written before implementation
- **Compliance**: Phase 1 generates API contracts before implementation; cross-implementation tests define expected behavior
- **Evidence**: Planning phase includes contract generation (Phase 1) before task execution (Phase 2 - separate command)

### Principle IV: API-First Architecture ✅ PASS
- **Requirement**: APIs must be well-defined and versioned before implementation
- **Compliance**: OpenAPI specification stored in docs/openapi.yaml defines all endpoints; both implementations conform to same contract
- **Evidence**: FR-007 (SCRAPI endpoints), FR-019 (.well-known consistency), user input mandates shared OpenAPI definition

### Principle V: Observability and Monitoring ✅ PASS
- **Requirement**: Services must be fully observable in production
- **Compliance**: Existing observability features preserved in both implementations; structured logging maintained
- **Evidence**: Current implementation has observability; migration preserves this (Assumption #10 - API remains compatible)

### Principle VI: Data Integrity and Versioning ✅ PASS
- **Requirement**: Data integrity guaranteed through versioning and validation
- **Compliance**: Strict lockstep versioning (FR-021); artifacts use versioned schemas; interoperability tests validate data integrity
- **Evidence**: FR-013 (100% interoperability), FR-018 (IETF standards compliance)

### Principle VII: Simplicity and Maintainability ⚠️ JUSTIFIED COMPLEXITY
- **Requirement**: Favor simple solutions over clever complexity
- **Concern**: Dual-language implementation increases maintenance burden
- **Justification**:
  - Dual implementation enables language choice based on team expertise and infrastructure
  - Parallel structures (FR-012) reduce cognitive load
  - Shared test vectors (Assumption #2) ensure implementations don't diverge
  - Monorepo (FR-024) simplifies coordinated releases
  - Complexity is inherent to the requirement, not introduced by design choice
- **Mitigation**: Strict parallel organization, shared API contracts, automated interoperability testing

### Principle VIII: Go Interoperability as Source of Truth (NON-NEGOTIABLE) ✅ PASS
- **Requirement**: Go tlog and cryptography are canonical; TypeScript conforms to Go
- **Compliance**: Go implementation is reference (Assumption #1); test vectors generated from Go (Assumption #2); 100% interoperability required (FR-013)
- **Evidence**: FR-013, FR-020 (cross-tests block releases), existing interop tests provide foundation (Assumption #5)

**GATE RESULT**: ✅ PASS WITH JUSTIFIED COMPLEXITY

One complexity violation (Principle VII - dual implementation) is justified by user requirement and mitigated by strict organization patterns.

## Project Structure

### Documentation (this feature)

```
specs/002-restructure-this-project/
├── spec.md              # Feature specification (completed)
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (to be generated)
├── data-model.md        # Phase 1 output (to be generated)
├── quickstart.md        # Phase 1 output (to be generated)
├── contracts/           # Phase 1 output (to be generated)
│   └── openapi.yaml     # Shared API contract for both implementations
├── checklists/          # Quality validation
│   └── requirements.md  # Spec quality checklist (completed)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root) - Post-Restructure

```
transparency-service/                    # Monorepo root
├── README.md                            # Language choice navigation
├── .github/
│   └── workflows/
│       ├── ci-typescript.yml            # TypeScript CI pipeline
│       ├── ci-golang.yml                # Go CI pipeline
│       ├── ci-interop.yml               # Cross-implementation tests
│       └── release.yml                  # Coordinated release workflow
│
├── docs/                                # Shared documentation
│   ├── openapi.yaml                     # Single source of truth for API contract
│   ├── INTEROP-TEST-RESULTS.md          # Cross-implementation test report
│   ├── rfc-6962-analysis.md             # Merkle tree implementation notes
│   └── install/                         # CLI installation scripts
│       ├── install-typescript.sh        # Install scitt-ts CLI
│       └── install-golang.sh            # Install scitt CLI
│
├── scitt-typescript/                    # TypeScript implementation
│   ├── README.md                        # TypeScript-specific quick start
│   ├── package.json                     # Bun dependencies
│   ├── tsconfig.json
│   ├── src/
│   │   ├── lib/                         # Library API
│   │   │   ├── keys/                    # Key generation
│   │   │   ├── identity/                # Issuer identity
│   │   │   ├── statement/               # Statement creation
│   │   │   ├── receipt/                 # Receipt generation/verification
│   │   │   ├── merkle/                  # Tile log (conforms to Go)
│   │   │   │   ├── tile-log.ts
│   │   │   │   └── proofs.ts
│   │   │   ├── cose/                    # COSE Sign1
│   │   │   ├── storage/                 # SQLite + MinIO adapters
│   │   │   └── index.ts                 # Library entry point
│   │   ├── cli/                         # CLI commands
│   │   │   ├── commands/
│   │   │   │   ├── keys.ts              # scitt-ts keys generate
│   │   │   │   ├── identity.ts          # scitt-ts identity create
│   │   │   │   ├── statement.ts         # scitt-ts statement sign
│   │   │   │   └── receipt.ts           # scitt-ts receipt verify
│   │   │   └── index.ts                 # CLI entry point
│   │   └── server/                      # HTTP server (SCRAPI)
│   │       ├── api/
│   │       │   ├── statements.ts        # POST /entries
│   │       │   ├── receipts.ts          # GET /entries/{entry-id}/receipt
│   │       │   ├── tiles.ts             # GET /tiles/{level}/{index}
│   │       │   └── wellknown.ts         # .well-known endpoints
│   │       ├── middleware/
│   │       │   ├── cors.ts
│   │       │   └── content-type.ts      # Enforce correct media types
│   │       └── index.ts                 # Server entry point
│   └── tests/
│       ├── unit/                        # TypeScript unit tests
│       ├── integration/                 # TypeScript integration tests
│       └── contract/                    # OpenAPI contract tests
│
├── scitt-golang/                        # Go implementation
│   ├── README.md                        # Go-specific quick start
│   ├── go.mod
│   ├── go.sum
│   ├── cmd/
│   │   ├── scitt/                       # CLI binary
│   │   │   └── main.go
│   │   └── scitt-server/                # Server binary
│   │       └── main.go
│   ├── pkg/                             # Library API (public)
│   │   ├── keys/                        # Key generation
│   │   ├── identity/                    # Issuer identity
│   │   ├── statement/                   # Statement creation
│   │   ├── receipt/                     # Receipt generation/verification
│   │   ├── merkle/                      # Tile log (uses golang.org/x/mod/sumdb/tlog)
│   │   │   ├── tilelog.go
│   │   │   └── proofs.go
│   │   ├── cose/                        # COSE Sign1
│   │   └── storage/                     # SQLite + MinIO adapters
│   ├── internal/                        # Internal packages
│   │   ├── server/                      # HTTP server implementation
│   │   │   ├── api/
│   │   │   │   ├── statements.go        # POST /entries
│   │   │   │   ├── receipts.go          # GET /entries/{entry-id}/receipt
│   │   │   │   ├── tiles.go             # GET /tiles/{level}/{index}
│   │   │   │   └── wellknown.go         # .well-known endpoints
│   │   │   └── middleware/
│   │   │       └── contenttype.go       # Enforce correct media types
│   │   └── cli/                         # CLI commands implementation
│   │       ├── keys.go                  # scitt keys generate
│   │       ├── identity.go              # scitt identity create
│   │       ├── statement.go             # scitt statement sign
│   │       └── receipt.go               # scitt receipt verify
│   └── tests/
│       ├── unit/                        # Go unit tests
│       ├── integration/                 # Go integration tests
│       └── contract/                    # OpenAPI contract tests
│
└── tests/                               # Top-level cross-implementation tests
    └── interop/                         # Interoperability test suite
        ├── fixtures/                    # Test vectors (generated from Go)
        │   ├── keys/
        │   ├── statements/
        │   ├── receipts/
        │   └── tiles/
        ├── test-vectors/                # Go-generated test vectors (existing)
        │   └── tlog-size-*.json
        ├── go-interop.test.ts           # TypeScript validates Go vectors
        ├── ts-interop_test.go           # Go validates TypeScript artifacts
        ├── tile-compat.test.ts          # Tile format compatibility
        ├── tile-compat_test.go
        ├── api-compat.test.ts           # API response compatibility
        ├── api-compat_test.go
        └── README.md                    # Interop test documentation
```

**Structure Decision**: Monorepo with dual top-level implementation directories chosen to enable:
1. Atomic commits across both implementations (FR-024)
2. Shared CI/CD pipeline that blocks releases on interop test failures (FR-020)
3. Single source of truth for API contracts in docs/openapi.yaml
4. Independent language-specific tests within each implementation
5. Orchestrated cross-implementation validation in tests/interop/

The parallel structure (library/, cli/, server/, tests/) within each implementation reduces cognitive load and enables developers to navigate between implementations easily (SC-010).

## Complexity Tracking

*One violation requires justification per Constitution Check*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Dual-language implementation (Principle VII) | User requirement for language choice based on team expertise, infrastructure, and organizational needs; enables broader ecosystem adoption | Single language implementation rejected because: (1) User explicitly requested dual implementation, (2) Different organizations have different language preferences and constraints, (3) Go implementation required as canonical reference per Constitution Principle VIII, (4) TypeScript implementation already exists with users who depend on it |

**Mitigation Strategy**:
- Strict parallel organization (FR-012) ensures structural consistency
- Shared API contracts in docs/openapi.yaml prevent divergence
- Automated cross-implementation testing (FR-009, FR-010) validates interoperability continuously
- Monorepo structure (FR-024) enables atomic commits and coordinated releases
- Strict lockstep versioning (FR-021) prevents version drift
- Go as canonical reference (Principle VIII, Assumption #1) eliminates ambiguity in case of disagreement

## Phase 0: Research & Decision Log

*To be generated: research.md*

### Research Topics

1. **GitHub Actions Monorepo CI Strategy**
   - How to structure workflows for dual-language monorepo
   - Conditional execution based on changed paths
   - Coordinated release workflow that blocks on interop test failures
   - Artifact publishing to GitHub releases

2. **OpenAPI 3.1 Dual-Language Code Generation**
   - Tools for generating TypeScript types from OpenAPI
   - Tools for generating Go types from OpenAPI
   - Ensuring media type specifications are honored
   - Contract testing strategies for both languages

3. **C2SP Tile Format Specification**
   - Detailed format requirements for tile storage
   - Compatibility between Go tlog and TypeScript implementation
   - Media type for tile data (`application/x-tlog-tile`)
   - Tile API endpoint structure

4. **SQLite + MinIO Integration Patterns**
   - Best practices for embedded SQLite in both languages
   - MinIO SDK usage for TypeScript (Bun compatibility)
   - MinIO SDK usage for Go
   - Connection pooling and error handling

5. **CLI Installation via GitHub Releases**
   - Structure of install scripts in docs/install/
   - Platform detection (macOS vs Linux, arm64 vs x86_64)
   - Binary naming conventions
   - Verification of downloaded binaries

6. **SCRAPI Endpoint Media Types**
   - Correct Content-Type for COSE statements
   - Correct Content-Type for CBOR receipts
   - Tile endpoint media types
   - .well-known endpoint response formats

7. **Cross-Implementation Test Vector Generation**
   - Strategy for generating test vectors from Go (canonical)
   - Format for storing test vectors (JSON, CBOR, or binary)
   - Version compatibility testing strategy
   - Automated test vector regeneration on Go changes

### Test Vector Generation Strategy

**Purpose**: Ensure 100% interoperability by validating TypeScript implementation against canonical Go reference.

**Approach**: Three-tier testing strategy
1. **Static Fixtures** (committed to repo):
   - Location: `/tests/interop/fixtures/{category}/`
   - Generated by: Go test vector generator (`scitt-golang/tests/vectors/generate.go`)
   - Format: JSON for human readability and version control
   - Regeneration: On Go library changes that affect cryptographic operations
   - Categories: keys/, statements/, receipts/, tiles/, checkpoints/
   - Versioning: Test vectors include schema version for compatibility tracking

2. **Dynamic Cross-Tests** (runtime generation):
   - Location: `/tests/interop/cross-*.test.ts` and `/tests/interop/cross-*_test.go`
   - Purpose: Validate live artifact exchange between implementations
   - Examples: Generate key in TS → sign in Go → verify both ways
   - Execution: Every CI run on both implementations

3. **CI Regeneration Strategy**:
   - Trigger: Changes to Go cryptographic libraries (`pkg/keys/`, `pkg/cose/`, `pkg/merkle/`)
   - Action: Regenerate static fixtures, commit if changed
   - Validation: TypeScript tests must pass with new vectors before merge
   - Blocking: Both implementations blocked if cross-tests fail

**Rationale**: Static fixtures provide stable regression tests, dynamic tests validate real-time interop, CI regeneration keeps vectors synchronized with canonical Go implementation.

8. **Import Path Migration Strategy**
   - Breaking change: TypeScript import paths change from `./src/lib/*` to `scitt-typescript/lib/*`
   - Direct migration approach: No compatibility shims (per user feedback)
   - Migration guide: Document import path updates required

   **Note**: Project is currently experimental with no prior releases. This restructure will be the foundation for the first production release. Migration communication strategy will be developed when transitioning from experimental to production (post-1.0.0 release planning).

## Phase 1: Design Artifacts

*To be generated:*
- `data-model.md` - Entity schemas, validation rules, state transitions
- `contracts/openapi.yaml` - Complete API contract for both implementations
- `quickstart.md` - User-facing quick start guide with language choice

### Design Requirements

**data-model.md must include**:
- Cryptographic Key schema (algorithm, format, serialization)
- Issuer Identity structure (.well-known hosting requirements)
- Statement format (SCITT specification compliance)
- Signed Statement format (COSE Sign1 structure)
- Receipt format (inclusion proof, consistency proof)
- Tile format (C2SP compliance, level/index addressing)
- Configuration schema (environment variables, config file format)

**contracts/openapi.yaml must include**:
- POST /entries (register statement) - media type: `application/cose`
- GET /entries/{entry-id}/receipt - media type: `application/cbor`
- GET /tiles/{level}/{index} - media type: `application/x-tlog-tile`
- GET /.well-known/scitt-configuration - media type: `application/json`
- GET /.well-known/jwks.json - media type: `application/jwk-set+json`
- Error response schemas (consistent status codes per FR-016)
- All endpoints tagged with implementation requirements

**quickstart.md must include**:
- Language choice section (Choose TypeScript or Go)
- Installation instructions per platform (macOS, Linux)
- Quick start for TypeScript (5-minute goal per SC-001)
- Quick start for Go (5-minute goal per SC-001)
- CLI command examples (keys, identity, statement, receipt)
- Server deployment example (both implementations)
- Library integration example (both implementations)

## Next Steps

This plan is complete through Phase 1 design specification. The next actions are:

1. **Execute Phase 0**: Generate research.md by researching topics listed above
2. **Execute Phase 1**: Generate data-model.md, contracts/openapi.yaml, quickstart.md
3. **Update Agent Context**: Run `.specify/scripts/bash/update-agent-context.sh claude`
4. **Execute Phase 2**: Run `/speckit.tasks` command to generate implementation tasks
5. **Begin Implementation**: Execute tasks in task order with test-first approach

**Branch**: `002-restructure-this-project`
**Plan Path**: `/Users/orie/Desktop/Work/tv/transparency-service/specs/002-restructure-this-project/plan.md`

---

*Planning complete. Awaiting Phase 0 research execution.*
