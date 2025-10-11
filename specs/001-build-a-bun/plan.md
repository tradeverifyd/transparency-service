# Implementation Plan: IETF Standards-Based Transparency Service

**Branch**: `001-build-a-bun` | **Date**: 2025-10-11 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/001-build-a-bun/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Build a library and CLI implementing IETF-compliant transparency service using COSE Hash Envelopes for efficient registration of large artifacts (parquet files, datasets). The service maintains an append-only Merkle tree log backed by SQLite metadata storage and MinIO-compatible object storage for receipts. All signed statements use COSE Sign1 with ES256 signatures, issuer identities are URLs with .well-known key material hosting, and Merkle tree uses tile log format for efficient proofs. CLI commands support full lifecycle: transparency service initialization, issuer identity generation, statement signing/registration, verification workflows, and log querying with rich metadata search.

## Technical Context

**Language/Version**: Bun (latest stable) with TypeScript
**Primary Dependencies**:
  - Minimal external libraries (leverage Bun built-ins)
  - COSE library ported from github.com/transmute-industries/cose (Merkle tree proofs implementation)
  - SQLite driver (Bun native support)
  - MinIO SDK (S3-compatible client)

**Storage**:
  - SQLite for all metadata (transparency service state, statement metadata, Merkle tree tiles)
  - Object storage abstraction layer supporting:
    - MinIO (default)
    - Local filesystem (development/testing)
    - Azure Blob Storage (production option)
    - Amazon S3 (production option)
  - Statement content stored in object storage (not SQLite)

**Testing**: Bun test framework (built-in)

**Target Platform**:
  - CLI: Cross-platform (Linux, macOS, Windows via Bun runtime)
  - Service: Linux server primary, containerizable

**Project Type**: Single project (library + CLI + service)

**Performance Goals**:
  - Hash computation: 1GB file in <30 seconds (streaming)
  - Statement registration: <5 seconds for 10MB files
  - Concurrent registrations: 100+ without degradation
  - Verification: <2 seconds regardless of artifact size

**Constraints**:
  - Streaming hash computation (no full file in memory)
  - Object storage portability via abstraction interfaces
  - Offline verification capability (no service dependency when verifying)
  - Single-node operation (no distributed consensus)

**Scale/Scope**:
  - Artifacts: 1KB to 100GB+
  - Registration rate: 100+ concurrent operations
  - Log size: Unbounded (tile log format scales)
  - Query performance: Indexed metadata fields (iss, sub, cty, typ, timestamp ranges)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### ✅ I. Transparency by Design
- **Status**: PASS
- All operations logged with full traceability via Merkle tree
- API responses include provenance metadata (issuer, timestamp, tree position)
- Hash envelopes make data transformations explicit and verifiable
- No hidden behavior - all cryptographic operations are standardized COSE/SCITT

### ✅ II. Verifiable Audit Trails
- **Status**: PASS
- Append-only Merkle tree provides tamper-evident log
- All state changes recorded with timestamps and cryptographic commitments
- Receipts provide independent verification of registration
- Tile log format enables efficient auditing and consistency proofs
- Object storage + SQLite metadata survives failures independently

### ⚠️ III. Test-First Development (NON-NEGOTIABLE)
- **Status**: PENDING IMPLEMENTATION
- Tests MUST be written before code during implementation phase
- Acceptance scenarios from spec.md will drive test creation
- Red-Green-Refactor cycle will be enforced in tasks.md
- **Action**: Ensure tasks.md includes explicit test-writing tasks before implementation

### ✅ IV. API-First Architecture
- **Status**: PASS
- SCITT SCRAPI defines all service endpoints (OpenAPI will be generated)
- CLI commands are clients of the API (no special-case CLI-only logic)
- Library exposes programmatic interface independent of CLI
- REST API follows SCITT standards with documented deviations if any

### ✅ V. Observability and Monitoring
- **Status**: PASS
- Health check endpoint reports component status (DB, storage, service)
- Structured logging via Bun logging (JSON format)
- Log query endpoint enables operational visibility
- Merkle tree state (size, root hash) exposed for monitoring
- Performance tracked via success criteria metrics

### ✅ VI. Data Integrity and Versioning
- **Status**: PASS
- All schemas will be versioned (SQLite migrations tracked)
- Hash envelopes include algorithm identifiers (SHA-256 default)
- COSE structures are versioned via protected headers
- Receipts include format version information
- Content-addressable storage prevents corruption

### ✅ VII. Simplicity and Maintainability
- **Status**: PASS
- Minimal dependencies (Bun built-ins, ported COSE code, SQLite, MinIO SDK)
- Standard IETF specifications (no custom crypto)
- Object storage abstraction avoids vendor lock-in
- Single-node deployment (no complex distributed coordination)
- Clear separation: library (core), CLI (user interface), service (HTTP API)

### 🟡 Complexity Notes

The constitution emphasizes simplicity, but this feature has inherent complexity due to:

1. **Cryptographic Standards Compliance**: COSE/SCITT specifications are complex but necessary for interoperability
2. **Object Storage Abstraction**: Supporting multiple backends adds interfaces but enables portability (constitutional requirement)
3. **Tile Log Format**: Merkle tree tile structure is complex but industry-standard for scalable transparency logs

These complexities are **justified** because they are:
- Required by IETF standards (not invented complexity)
- Essential for stated requirements (object storage portability, large-scale logs)
- Well-documented established patterns (not experimental)

**Overall Constitution Compliance**: ✅ PASS with action item for Test-First enforcement during implementation

## Project Structure

### Documentation (this feature)

```
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```
src/
├── lib/                          # Core library (COSE, Merkle tree, cryptography)
│   ├── cose/                     # COSE Sign1, Hash Envelope implementations
│   │   ├── sign.ts               # Signing and verification
│   │   ├── hash-envelope.ts      # Hash envelope creation and validation
│   │   └── key-material.ts       # Key generation, URL-based resolution
│   ├── merkle/                   # Merkle tree tile log implementation
│   │   ├── tile-log.ts           # Tile-based Merkle tree
│   │   ├── proofs.ts             # Inclusion and consistency proofs
│   │   └── checkpoints.ts        # Signed tree head management
│   ├── storage/                  # Storage abstraction layer
│   │   ├── interface.ts          # Storage interface definition
│   │   ├── local.ts              # Local filesystem implementation
│   │   ├── minio.ts              # MinIO/S3-compatible implementation
│   │   ├── azure.ts              # Azure Blob Storage implementation
│   │   └── s3.ts                 # Amazon S3 implementation
│   └── database/                 # SQLite database layer
│       ├── schema.ts             # Database schema and migrations
│       ├── statements.ts         # Statement metadata queries
│       ├── receipts.ts           # Receipt storage and retrieval
│       └── log-state.ts          # Merkle tree state management
│
├── service/                      # HTTP service (SCITT SCRAPI endpoints)
│   ├── server.ts                 # HTTP server setup
│   ├── routes/                   # API route handlers
│   │   ├── config.ts             # Transparency configuration endpoint
│   │   ├── register.ts           # Statement registration endpoint
│   │   ├── receipts.ts           # Receipt resolution endpoint
│   │   └── health.ts             # Health check endpoint
│   ├── middleware/               # Request/response middleware
│   │   ├── logging.ts            # Structured logging
│   │   └── validation.ts         # Request validation
│   └── types/                    # SCITT SCRAPI types
│       └── scrapi.ts             # SCRAPI request/response types
│
├── cli/                          # CLI commands
│   ├── index.ts                  # CLI entry point
│   ├── commands/                 # Command implementations
│   │   ├── transparency/         # Transparency service commands
│   │   │   ├── init.ts           # Initialize transparency service
│   │   │   ├── serve.ts          # Start HTTP service
│   │   │   └── query.ts          # Query log by metadata
│   │   ├── issuer/               # Issuer identity commands
│   │   │   └── init.ts           # Generate issuer identity
│   │   ├── statement/            # Statement lifecycle commands
│   │   │   ├── sign.ts           # Sign hash envelope statement
│   │   │   ├── register.ts       # Register statement with service
│   │   │   └── verify.ts         # Verify statement signature
│   │   └── receipt/              # Receipt verification commands
│   │       └── verify.ts         # Verify receipt and inclusion proof
│   └── utils/                    # CLI utilities
│       ├── config.ts             # CLI configuration
│       └── output.ts             # Output formatting
│
└── types/                        # Shared TypeScript types
    ├── cose.ts                   # COSE structure types
    ├── scitt.ts                  # SCITT data types
    └── config.ts                 # Configuration types

tests/
├── contract/                     # Contract tests (SCITT SCRAPI compliance)
│   ├── registration.test.ts      # Statement registration contract
│   ├── receipts.test.ts          # Receipt resolution contract
│   └── config.test.ts            # Configuration endpoint contract
│
├── integration/                  # Integration tests (full workflows)
│   ├── issuer-workflow.test.ts   # Issuer: init → sign → register
│   ├── verifier-workflow.test.ts # Verifier: verify signature → receipt
│   ├── auditor-workflow.test.ts  # Auditor: consistency checks
│   └── storage-portability.test.ts # Object storage migration
│
└── unit/                         # Unit tests (individual components)
    ├── cose/                     # COSE operations tests
    ├── merkle/                   # Merkle tree tests
    ├── storage/                  # Storage abstraction tests
    └── database/                 # Database layer tests

package.json                      # Bun project configuration
tsconfig.json                     # TypeScript configuration
README.md                         # Project documentation
```

**Structure Decision**: Single project structure selected because:
- Library + CLI + Service are tightly coupled (all part of same transparency system)
- Shared types and interfaces benefit from monolithic structure
- Bun's performance makes single build target viable
- Simplicity principle favors fewer projects over premature separation
- All components will be versioned together (SCITT/COSE standards evolve together)

Clear layering:
1. **lib/**: Core cryptographic and storage primitives (no HTTP, no CLI)
2. **service/**: HTTP API layer consuming lib (no CLI dependencies)
3. **cli/**: User interface consuming both lib and service (thin command wrappers)

## Complexity Tracking

*No constitution violations to justify - all checks passed.*
