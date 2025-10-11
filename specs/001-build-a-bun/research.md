# Research: IETF Standards-Based Transparency Service

**Feature**: IETF Standards-Based Transparency Service
**Date**: 2025-10-11
**Status**: Complete

## Overview

This document consolidates research findings for implementing a transparency service complying with IETF SCITT/COSE specifications. All technical decisions below are informed by the user requirements, IETF editor's draft standards (latest versions), and the transmute-industries/cose reference implementation.

**Standards Sources** (Editor's Drafts - Latest Versions):
- SCITT SCRAPI: https://ietf-wg-scitt.github.io/draft-ietf-scitt-scrapi/draft-ietf-scitt-scrapi.txt
- SCITT Architecture: https://ietf-wg-scitt.github.io/draft-ietf-scitt-architecture/draft-ietf-scitt-architecture.txt
- COSE Hash Envelope: https://cose-wg.github.io/draft-ietf-cose-hash-envelope/draft-ietf-cose-hash-envelope.txt
- COSE Merkle Tree Proofs: https://cose-wg.github.io/draft-ietf-cose-merkle-tree-proofs/draft-ietf-cose-merkle-tree-proofs.txt
- Reference Implementation: github.com/transmute-industries/cose

## Key Technical Decisions

### 1. Runtime: Bun

**Decision**: Use Bun as the TypeScript runtime for library, CLI, and service

**Rationale**:
- Native TypeScript support (no separate compilation step)
- Built-in test runner (bun test) eliminates test framework dependency
- Fast startup time critical for CLI responsiveness
- Native SQLite support reduces dependencies
- Excellent streaming file I/O for hash computation of large files
- Single binary distribution simplifies deployment

**Alternatives Considered**:
- Node.js: More mature but slower, requires additional tooling
- Deno: Good TypeScript support but smaller ecosystem for S3/MinIO clients
- Go: Better performance but increases complexity (new language, no existing COSE library)

**Trade-offs**:
- Bun is newer with smaller community
- Acceptable because core functionality relies on standards (COSE/SCITT) not runtime-specific features
- Performance requirements (GB file hashing) favor Bun's I/O performance

---

### 2. Database: SQLite

**Decision**: Use SQLite for all metadata and Merkle tree state storage

**Rationale**:
- Zero-configuration deployment (file-based, no server)
- Sufficient performance for metadata queries (indexed on iss, sub, cty, typ, timestamp)
- ACID guarantees for Merkle tree state consistency
- Bun has native SQLite support
- Easy backup/migration (single file)
- Satisfies spec requirement for "SQL" storage

**Alternatives Considered**:
- PostgreSQL/MySQL: Over-engineered for single-node deployment
- Embedded key-value store (RocksDB): Less convenient for rich querying

**Schema Design Priorities**:
- Statements table: Full-text indexed protected headers for query capability
- Receipts table: Links to object storage with content-addressable keys
- Merkle tree tiles table: Efficient tile log format storage
- Migrations: Versioned schema with forward/backward migration support

---

### 3. Object Storage: MinIO-Compatible Abstraction

**Decision**: Implement storage interface supporting MinIO, Local, Azure Blob, S3

**Rationale**:
- Requirement explicitly calls for "mino compatible interface" and portability
- Statements and receipts stored separately from metadata (SQLite stores only pointers)
- Content-addressable storage (hash-based keys) enables deduplication
- Interface abstraction allows deployment flexibility without code changes

**Interface Design**:
```typescript
interface ObjectStorage {
  put(key: string, data: Uint8Array): Promise<void>
  get(key: string): Promise<Uint8Array>
  exists(key: string): Promise<boolean>
  list(prefix: string): AsyncIterableIterator<string>
}
```

**Implementations**:
1. **Local**: Filesystem-based for development/testing
2. **MinIO**: S3-compatible API (default production)
3. **Azure**: Azure Blob Storage SDK
4. **S3**: AWS S3 SDK

**Key Resolution**:
- Statements: `statements/{hash-of-statement}`
- Receipts: `receipts/{hash-of-receipt}`
- Enables portability and integrity verification

---

### 4. COSE Implementation: Port from transmute-industries/cose

**Decision**: Port Merkle tree proofs implementation from transmute-industries/cose to Bun/TypeScript

**Rationale**:
- Reference implementation exists at github.com/transmute-industries/cose/tree/main/src/drafts/draft-ietf-cose-merkle-tree-proofs
- COSE Sign1 and Hash Envelope operations available in same repo
- Porting ensures minimal dependencies (avoid pulling entire library)
- Bun's TypeScript allows direct port without major rewrites

**Components to Port**:
- COSE Sign1 signing and verification (ES256 focus)
- COSE Hash Envelope creation and validation
- Merkle tree tile log structure
- Inclusion and consistency proof generation/verification
- Signed tree head (checkpoint) management

**Cryptography Dependencies**:
- Use Web Crypto API (available in Bun) for ES256 operations
- Avoid external crypto libraries where possible
- SHA-256 hashing via Web Crypto API

---

### 5. Merkle Tree: Tile Log Format

**Decision**: Implement Merkle tree using tile log format with configurable parameters

**Rationale**:
- Industry standard for certificate transparency and transparency logs
- Efficient storage (tiles cached/reused)
- Scalable to billions of entries
- Supports efficient consistency and inclusion proofs
- Configurable tile height during initialization for performance tuning

**Tile Log Structure** (C2SP tlog-tiles specification):

Tile naming convention: `tile/<L>/<N>[.p/<W>]`
- `<L>`: Level (0 = leaves, higher = Merkle tree hashes)
- `<N>`: Tile index as zero-padded 3-digit path elements
  - Example: index 1234067 → `x001/x234/067`
- `[.p/<W>]`: Optional partial tile with width W (1-255 hashes)

Tile characteristics:
- Full tiles: Exactly 256 hashes (8,192 bytes)
- Partial tiles: Rightmost tiles with 1-255 hashes
- Immutable, cacheable static resources
- Level 0: Leaf hashes (statement hashes)
- Level L+1: Hashes of full tiles from level L

**Storage in Object Storage (MinIO)**:
- Merkle tree hash tiles: `tile/<L>/<N>[.p/<W>]`
- Entry tiles (statements): `tile/entries/<N>[.p/<W>]`
- Receipts: `receipts/{entry-id}`
- Checkpoint: `checkpoint` (signed note with tree size and root hash)

**SQLite Metadata**:
- Tile index mapping (level, index → object storage key)
- Statement metadata (iss, sub, cty, typ, timestamp) for querying
- Receipt pointers (entry-id → object storage key)
- Current tree size and checkpoint state

**Single SCRAPI Server**:
- Serves all tiles, statements, receipts from unified object storage
- Clients fetch tiles directly for proof verification
- No need for separate tile server infrastructure

**References**:
- C2SP tlog-tiles: https://github.com/C2SP/C2SP/blob/main/tlog-tiles.md
- Tile-based logs: https://transparency.dev/articles/tile-based-logs/
- Implementation: github.com/transmute-industries/cose

---

### 6. Identity Management: URL-Based with .well-known

**Decision**: Issuer identities are URLs, public keys served at .well-known paths

**Rationale**:
- SCITT architecture specifies URL-based issuer identifiers
- .well-known URLs are IETF standard for service discovery (RFC 8615)
- Decentralized (no central key registry)
- Enables key rotation via URL content updates

**Implementation**:
- Issuer URL in `iss` field of COSE Sign1 protected header
- Key ID (`kid`) references specific key at issuer URL
- Transparency service keys at `/.well-known/scitt-keys` (per SCRAPI spec)
- Format: COSE Key Set for interoperability

**Issuer Init Command Outputs**:
- Private key (PEM file, not shared)
- Public key (COSE Key format for hosting)
- Example .well-known directory structure for easy hosting
- Issuer URL configuration file

---

### 7. CLI Command Structure

**Decision**: Hierarchical command structure matching user workflows

**User-Specified Commands**:

1. **transparency init**: Initialize transparency service
   - Generates service identity (key pair for receipts)
   - Creates SQLite database schema
   - Configures object storage connection
   - Initializes Merkle tree with parameters (tile height)

2. **issuer init**: Generate issuer identity
   - Creates ES256 key pair
   - Generates issuer URL configuration
   - Produces .well-known hosting materials (COSE Key Set format)

3. **transparency statement sign**: Sign statement
   - Takes artifact file path and issuer identity
   - Computes streaming hash (SHA-256)
   - Creates COSE Hash Envelope
   - Signs with COSE Sign1 (issuer's private key)
   - Outputs signed statement (CBOR file)

4. **transparency statement register**: Register signed statement
   - Submits to transparency service URL
   - Handles both synchronous (201) and asynchronous (303) registration
   - Polls status if async (302 → 200)
   - Retrieves and saves receipt when ready

5. **transparency statement verify**: Verify signed statement
   - Resolves issuer public key from iss/kid fields
   - Verifies COSE Sign1 signature
   - Validates hash envelope structure

6. **transparency receipt verify**: Verify receipt
   - Takes artifact, signed statement, and receipt
   - Verifies artifact hash matches statement
   - Verifies statement signature (calls statement verify)
   - Verifies Merkle inclusion proof in receipt
   - Validates receipt signature (transparency service commitment)

7. **transparency log query**: Query log by metadata
   - Accepts JSON query config mapping to protected header fields
   - Supports: iss (issuer), sub (subject), cty (content type), typ (type)
   - Supports timestamp range queries
   - Returns matching statements with receipts
   - Executes SQL queries against indexed metadata

8. **transparency serve**: Start HTTP service
   - Launches SCITT SCRAPI-compliant HTTP server
   - Exposes endpoints: config, keys, register, receipts, health
   - Configured port and host binding

**Command Naming Convention**: `<entity> <noun> <verb>` pattern

---

### 8. HTTP Service: SCITT SCRAPI Compliance (Latest Editor's Draft)

**Decision**: Implement REST API following latest SCITT SCRAPI specification

**Mandatory Endpoints** (per latest SCRAPI spec):

1. **GET /.well-known/scitt-configuration**
   - Returns CBOR map of service-specific configuration elements
   - Content-Type: application/cbor
   - Includes: service_url, algorithms, registration_policies

2. **GET /.well-known/scitt-keys**
   - Returns COSE Key Set with public keys for verifying Receipts
   - Content-Type: application/cbor
   - Transparency service's public signing key(s)

3. **POST /entries**
   - Register signed statement (COSE Sign1 with hash envelope)
   - Content-Type: application/cose
   - Response codes:
     - **201 Created**: Synchronous registration with immediate Receipt in body
     - **303 See Other**: Asynchronous registration, Location header points to status URL

4. **GET /entries/{entry-id}**
   - Query registration status (for async) or retrieve signed statement
   - Response codes:
     - **302 Found**: Registration still in progress, Location header for polling
     - **200 OK**: Registration complete, Receipt in body (Content-Type: application/cose)

5. **GET /entries/{entry-id}/receipt**
   - Resolve receipt for registered statement
   - Content-Type: application/cose
   - Response: 200 (receipt ready) or 404 (not found)

6. **GET /health**
   - Health check reporting component status
   - Content-Type: application/json
   - Reports: database, object storage, service status

**Error Responses**: Concise Problem Details (CBOR format, not JSON)

**Important SCRAPI Updates**:
- Uses CBOR (not JSON) for configuration and keys endpoints
- COSE Key Set format (not JWK) for service keys
- Entry ID is content-addressable (hash of signed statement)
- Asynchronous flow: POST → 303 → GET (302 polling) → GET (200 with receipt)

**Logging**: Structured JSON logs with correlation IDs

---

### 9. Streaming Hash Computation

**Decision**: Implement streaming SHA-256 for large file hashing

**Rationale**:
- Requirement: Support artifacts from 1KB to 100GB+
- Cannot load entire file into memory
- Success criteria: 1GB file hashed in <30 seconds

**Implementation Approach**:
- Use Bun's streaming file I/O (`Bun.file(path).stream()`)
- Web Crypto API SubtleCrypto for SHA-256 (chunk-by-chunk)
- Read file in chunks (e.g., 1MB buffers)
- Update hash incrementally
- Never materialize full file in memory

**Performance Target**:
- 1MB chunks * 1024 = 1GB file
- Target: <30 seconds for 1GB = ~35MB/s throughput
- Bun's I/O performance should easily meet this

---

### 10. Offline Verification Capability

**Decision**: Design verification to work without network access

**Rationale**:
- Success criteria: 90%+ verification tasks offline
- Critical for supply chain scenarios (air-gapped environments)
- Verifiers should trust cryptographic proofs, not live service queries

**Offline Verification Flow**:
1. Verifier receives: artifact file, signed statement, receipt
2. Compute artifact hash (streaming, no network)
3. Verify hash matches hash envelope in statement (no network)
4. Verify COSE Sign1 signature (requires issuer public key)
   - If public key cached: no network
   - If not cached: fetch once from issuer URL, cache
5. Verify receipt Merkle inclusion proof (no network)
6. Verify receipt signature with service public key (cached from /.well-known/scitt-keys)

**Online Operations** (one-time setup):
- Fetch issuer public key (cache by iss+kid)
- Fetch transparency service public keys from /.well-known/scitt-keys (cache)
- Optional: Query service for revocation status

**Cache Strategy**:
- Store COSE keys in local config directory (~/.transparency/keys/)
- TTL configurable (default 24 hours)
- Manual cache refresh command available

---

## SCITT Data Structures (Latest Architecture Spec)

### Signed Statement (COSE_Sign1)
- **Protected Headers**: iss (issuer URL), sub (subject), optional cty (content type), typ (type)
- **Payload**: Hash envelope (hash of artifact, not full artifact)
- **Signature**: ES256 (ECDSA P-256 + SHA-256)

### Receipt
- **Structure**: COSE structure containing inclusion proof
- **Contents**:
  - Merkle tree inclusion proof (path from leaf to root)
  - Tree size at time of inclusion
  - Signed tree head (checkpoint)
- **Signature**: Signed by transparency service using keys from /.well-known/scitt-keys

### Transparent Statement
- **Definition**: Signed Statement + Receipt(s)
- **Purpose**: Complete proof package for offline verification
- **Format**: CBOR-encoded bundle

---

## Dependencies Summary

**Runtime Dependencies** (minimal):
- Bun runtime (includes TypeScript, test runner, SQLite)
- MinIO SDK for S3-compatible storage (@minio/minio-js or equivalent)
- Azure Storage SDK for Azure Blob (@azure/storage-blob)
- AWS SDK for S3 (@aws-sdk/client-s3)

**Development Dependencies**:
- TypeScript (bundled with Bun)
- Bun test framework (bundled)

**Zero External Crypto Libraries**:
- Web Crypto API sufficient for ES256 and SHA-256
- COSE implementation ported (no external COSE library dependency)

---

## Standards Compliance Checklist (Latest Editor's Drafts)

- ✅ **COSE Hash Envelope** (draft-ietf-cose-hash-envelope, editor's draft)
  - Hash algorithm header (label 258)
  - Preimage content type (label 259)
  - Optional payload location (label 260)

- ✅ **SCITT Architecture** (draft-ietf-scitt-architecture, editor's draft)
  - Signed statements (COSE Sign1 with protected headers)
  - Receipts with inclusion proofs
  - Transparent statements (statement + receipt)
  - Issuer/Relying Party/Auditor roles
  - Append-only verifiable data structure

- ✅ **SCITT SCRAPI** (draft-ietf-scitt-scrapi, editor's draft)
  - /.well-known/scitt-configuration (CBOR format)
  - /.well-known/scitt-keys (COSE Key Set format)
  - POST /entries (statement registration)
  - GET /entries/{entry-id} (status query / receipt resolution)
  - Asynchronous registration support (303 → 302 → 200)
  - Concise Problem Details for errors

- ✅ **ES256 Signature Algorithm**
  - ECDSA with P-256 curve and SHA-256
  - COSE algorithm identifier -7

- ✅ **URL-Based Issuer Identities**
  - Issuer URLs in `iss` protected header field
  - .well-known key resolution (COSE Key Set format)

- ✅ **Tile Log Merkle Tree**
  - Industry-standard format
  - Inclusion and consistency proofs
  - Ported from transmute-industries/cose implementation

---

## Risk Mitigation

**Identified Risks**:

1. **Bun Maturity**: Newer runtime may have undiscovered issues
   - Mitigation: Core logic is standards-based COSE/SCITT (runtime-agnostic)
   - Can port to Node.js if needed with minimal changes

2. **Porting COSE Library**: Errors in porting could break standards compliance
   - Mitigation: Extensive unit tests against test vectors from COSE specs
   - Contract tests validate SCITT SCRAPI compliance

3. **Object Storage Reliability**: Network failures during receipt storage
   - Mitigation: Atomic SQLite transactions track storage state
   - Retry logic for transient failures
   - Health check detects persistent failures

4. **SQLite Scalability**: Could hit limits with extremely high statement rates
   - Mitigation: Benchmarking during implementation
   - Success criteria: 100 concurrent registrations (testable)
   - Future migration path: Replace SQLite persistence layer (interface-based)

5. **Key Management**: Issuer private keys are user-managed (not service-managed)
   - Mitigation: Clear documentation and warnings in `issuer init`
   - Recommend hardware security modules (HSMs) for production
   - Service never handles issuer private keys (by design)

6. **CBOR vs JSON**: SCRAPI uses CBOR for config/keys, not JSON
   - Mitigation: Use CBOR libraries in Bun ecosystem
   - Clearly document content types in API responses
   - Provide CLI output formatting (CBOR → human-readable)

---

## Next Steps (Phase 1)

1. Generate **data-model.md**: Define entity schemas for SQLite and object storage, including CBOR structures
2. Generate **contracts/**: OpenAPI/AsyncAPI spec for SCITT SCRAPI endpoints (noting CBOR content types)
3. Generate **quickstart.md**: End-to-end quickstart guide with example CBOR/COSE workflows
4. Update agent context with Bun/TypeScript/COSE/SCITT/CBOR information

All research complete. No NEEDS CLARIFICATION items remain. Ready for Phase 1 design.
