# Data Model: Dual-Language Project Restructure

**Feature**: 002-restructure-this-project
**Date**: 2025-10-12

This document defines all data entities, schemas, validation rules, and state transitions for the dual-language transparency service implementation.

## Entity Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Transparency Service                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐ │
│  │ Cryptographic│    │   Issuer     │    │  Statement   │ │
│  │     Key      │───▶│   Identity   │───▶│ (COSE Sign1) │ │
│  └──────────────┘    └──────────────┘    └──────┬───────┘ │
│                                                   │          │
│                                                   ▼          │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐ │
│  │  Checkpoint  │    │  Merkle Tree │◀───│   Receipt    │ │
│  │ (COSE Sign1) │◀───│    (Tiles)   │    │ (COSE Proof) │ │
│  └──────────────┘    └──────────────┘    └──────────────┘ │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## 1. Cryptographic Key

**Purpose**: ES256 (ECDSA P-256) signing keys for statements, receipts, and checkpoints

**Schema** (COSE Key format per RFC 8152):
```typescript
interface CoseKey {
  kty: "EC";           // Key type: Elliptic Curve
  crv: "P-256";        // Curve: P-256 (secp256r1)
  kid: string;         // Key identifier (unique)
  x: string;           // Public key x-coordinate (base64url)
  y: string;           // Public key y-coordinate (base64url)
  d?: string;          // Private key (base64url, optional for public keys)
  use?: "sig";         // Key usage
  alg?: "ES256";       // Algorithm
}
```

**Validation Rules**:
- `kty` MUST be "EC"
- `crv` MUST be "P-256"
- `kid` MUST be unique per key
- `x` and `y` MUST be valid base64url-encoded coordinates
- `d` MUST be present for private keys, absent for public keys
- Key size MUST be 256 bits

**Storage**:
- Private keys: Secure key storage (environment variables, key management service)
- Public keys: `/.well-known/scitt-keys` endpoint
- Format: CBOR-encoded COSE Key Set

**Lifecycle**:
```
[Generated] ─▶ [Active] ─▶ [Rotated] ─▶ [Archived]
                  │
                  └─▶ [Revoked] (security incident)
```

## 2. Issuer Identity

**Purpose**: Cryptographic identity for entities that create signed statements

**Schema**:
```typescript
interface IssuerIdentity {
  id: string;                    // Issuer identifier (DID, URL, etc.)
  publicKeys: CoseKey[];         // Active public keys
  metadata: {
    name?: string;               // Human-readable name
    description?: string;
    created: string;             // RFC3339 timestamp
    updated?: string;            // RFC3339 timestamp
  };
  wellKnownUri?: string;        // .well-known hosting location
}
```

**Validation Rules**:
- `id` MUST be a valid URI (did:web:, https://, etc.)
- `publicKeys` MUST contain at least one active key
- `created` MUST be valid RFC3339 timestamp
- If `wellKnownUri` provided, MUST be accessible via HTTPS

**Storage**:
- SQLite table: `issuers`
- MinIO bucket: `scitt-artifacts/issuers/{issuer-id}.cbor`
- `.well-known` hosting: User's web server

**Hosting Format** (.well-known/issuer.json):
```json
{
  "id": "did:web:example.com",
  "keys": [
    {
      "kty": "EC",
      "crv": "P-256",
      "kid": "issuer-key-2025",
      "x": "<base64url>",
      "y": "<base64url>"
    }
  ],
  "created": "2025-10-12T00:00:00Z"
}
```

## 3. Statement

**Purpose**: Transparency log entry following SCITT specification

**Schema** (COSE Sign1 per RFC 8152):
```typescript
interface Statement {
  protected: {
    alg: -7;                     // ES256
    kid: string;                 // Key identifier
    content_type: string;        // Payload content type
  };
  unprotected: {};
  payload: Uint8Array;           // Statement payload (arbitrary data)
  signature: Uint8Array;         // ECDSA P-256 signature
}
```

**Payload Structure** (application-specific):
```typescript
interface StatementPayload {
  iss: string;                   // Issuer identifier
  sub?: string;                  // Subject
  iat: number;                   // Issued at (Unix timestamp)
  exp?: number;                  // Expiration (Unix timestamp)
  content: any;                  // Application-specific content
}
```

**Validation Rules**:
- Protected header MUST contain `alg: -7` (ES256)
- Protected header MUST contain valid `kid`
- Signature MUST verify against issuer's public key
- Payload MUST be valid CBOR or JSON
- Statement size MUST NOT exceed 1MB

**Entry Identifier** (per SCRAPI):
```
entryId = SHA-256(COSE_Sign1_bytes)
```

**Storage**:
- SQLite table: `statements`
- MinIO: Entry tiles (`/tile/entries/{index}`)
- Format: COSE Sign1 binary

## 4. Receipt

**Purpose**: Cryptographic proof of statement registration using IETF COSE Merkle Proofs

**Schema** (COSE Sign1 with inclusion proof):
```typescript
interface Receipt {
  protected: {
    alg: -7;                     // ES256
    kid: string;                 // Service key identifier
    vds: "RFC9162_SHA256";       // Verifiable Data Structure
    tree_size: number;           // Tree size at registration
  };
  unprotected: {};
  payload: {
    leaf_index: number;          // Index in Merkle tree
    inclusion_path: string[];    // Array of base64-encoded hashes
  };
  signature: Uint8Array;         // Service signature
}
```

**Validation Rules**:
- `vds` MUST be "RFC9162_SHA256"
- `tree_size` MUST be positive integer
- `leaf_index` MUST be less than `tree_size`
- `inclusion_path` length MUST equal `floor(log2(tree_size))`
- All hashes in path MUST be 32 bytes (SHA-256)
- Signature MUST verify against service key

**Verification Algorithm** (RFC 6962):
```
1. leaf_hash = SHA-256(0x00 || statement)
2. current = leaf_hash
3. For each sibling in inclusion_path:
     if (index & 1):
       current = SHA-256(0x01 || sibling || current)
     else:
       current = SHA-256(0x01 || current || sibling)
     index = index >> 1
4. computed_root = current
5. Verify computed_root matches checkpoint root
6. Verify COSE signature
```

**Storage**:
- Embedded in statement response (SCRAPI)
- SQLite table: `receipts` (optional caching)
- Format: CBOR-encoded COSE Sign1

## 5. Merkle Tree (RFC 6962)

**Purpose**: Tamper-evident log structure using C2SP tile format

**Structure**:
```
Height: Variable (grows with log size)
Tile Height: 8 (256 leaves per tile)
Hash Algorithm: SHA-256
Leaf Hash: SHA-256(0x00 || data)
Node Hash: SHA-256(0x01 || left || right)
```

**Tile Schema**:
```typescript
interface Tile {
  level: number;              // 0 = leaves, higher = intermediate nodes
  index: number;              // Position at this level
  width: number;              // Number of hashes (1-256)
  hashes: Uint8Array[];       // Array of 32-byte SHA-256 hashes
}
```

**Addressing**:
```
Full Tile:     /tile/{level}/{index}
Partial Tile:  /tile/{level}/{index}.p/{width}
Entry Tile:    /tile/entries/{index}[.p/{width}]
```

**Storage**:
- MinIO bucket: `scitt-tiles/`
- Path format: `{level}/{index}` or `{level}/{index}.p.{width}`
- Size: Full tile = 8,192 bytes (256 × 32)
- Immutable: Yes (long-lived caching)

**State Transitions**:
```
[Empty] ─▶ [Partial] ─▶ [Full] ─▶ [Immutable]
            (1-255)     (256)      (never changes)
```

## 6. Checkpoint

**Purpose**: Signed proof of Merkle tree state at specific size (C2SP adapted)

**Schema** (COSE Sign1):
```typescript
interface Checkpoint {
  protected: {
    alg: -7;                     // ES256
    kid: string;                 // Service key identifier
  };
  unprotected: {};
  payload: {
    origin: string;              // Service origin (e.g., "example.com/log")
    tree_size: number;           // Current tree size
    root_hash: string;           // Base64-encoded root hash
    timestamp: string;           // RFC3339 timestamp
  };
  signature: Uint8Array;         // Service signature
}
```

**Validation Rules**:
- `origin` MUST match service configuration
- `tree_size` MUST be positive integer
- `root_hash` MUST be valid base64-encoded 32-byte hash
- `timestamp` MUST be valid RFC3339 timestamp
- `timestamp` MUST NOT be in the future
- Signature MUST verify against service key

**Response Format** (GET /checkpoint):
```
Content-Type: application/cose
Body: CBOR-encoded COSE_Sign1
```

**Storage**:
- SQLite table: `checkpoints`
- In-memory cache for latest checkpoint
- Historical checkpoints stored for consistency proofs

**Update Frequency**:
- New checkpoint generated on each statement registration
- Latest checkpoint returned by GET /checkpoint
- Short-term caching (e.g., 5 seconds)

## 7. Service Configuration

**Purpose**: SCRAPI service metadata and capabilities

**Schema** (per SCRAPI spec):
```typescript
interface ServiceConfiguration {
  serviceId: string;                  // Service identifier URL
  treeAlgorithm: "RFC9162_SHA256";   // Merkle tree algorithm
  signatureAlgorithm: "ES256";       // Signature algorithm
  entryId: "sha-256";                // Entry ID hash algorithm
  supportedPayloadFormats: string[]; // e.g., ["application/json", "application/cbor"]
  registrationPolicies: Policy[];    // Registration requirements
  extensions?: {
    c2sp_tiles: boolean;             // C2SP tile support
    checkpoint_endpoint: string;     // "/checkpoint"
  };
}
```

**Response Format** (GET /.well-known/scitt-configuration):
```
Content-Type: application/cbor
Body: CBOR-encoded configuration
```

**Validation Rules**:
- `serviceId` MUST be valid HTTPS URL
- `treeAlgorithm` MUST be "RFC9162_SHA256"
- `signatureAlgorithm` MUST be "ES256"
- `entryId` MUST be "sha-256"

**Storage**:
- Static configuration file
- Loaded at service startup
- Cached indefinitely

## 8. Cross-Implementation Test Vector

**Purpose**: Validate interoperability between TypeScript and Go implementations

**Schema**:
```typescript
interface TestVector {
  version: string;               // Test vector version (semver)
  specs: {
    scrapi: string;             // SCRAPI spec version
    c2sp: string;               // C2SP spec reference
    cose_proofs: string;        // COSE Merkle Proofs version
  };
  generatedFrom: "go";          // Source implementation
  algorithm: "ES256";           // Cryptographic algorithm
  hashAlgorithm: "SHA-256";     // Hash algorithm
  testCases: TestCase[];
}

interface TestCase {
  name: string;                 // Test case name
  category: string;             // e.g., "scrapi", "c2sp-tiles", "cose-proofs"
  input: Record<string, any>;   // Test inputs
  output: Record<string, any>;  // Expected outputs
}
```

**Storage**:
- Location: `tests/interop/fixtures/{category}/`
- Format: JSON (human-readable)
- Version control: Git (committed with code)

**Categories**:
- `keys/` - ES256 key generation
- `scrapi/` - SCRAPI endpoint responses
- `c2sp/` - C2SP tile generation
- `cose-proofs/` - COSE Merkle proof structures
- `checkpoints/` - Checkpoint generation

## Database Schema (SQLite)

```sql
-- Registered statements
CREATE TABLE statements (
  entry_id TEXT PRIMARY KEY,           -- SHA-256 of COSE Sign1
  cose_sign1 BLOB NOT NULL,            -- Full COSE Sign1 statement
  tree_index INTEGER NOT NULL UNIQUE,  -- Position in Merkle tree
  registered_at INTEGER NOT NULL,      -- Unix timestamp
  issuer_id TEXT,                      -- Issuer identifier
  payload_hash BLOB NOT NULL           -- SHA-256 of payload
);

CREATE INDEX idx_statements_tree ON statements(tree_index);
CREATE INDEX idx_statements_issuer ON statements(issuer_id);
CREATE INDEX idx_statements_registered ON statements(registered_at);

-- Checkpoints (tree state snapshots)
CREATE TABLE checkpoints (
  tree_size INTEGER PRIMARY KEY,       -- Tree size at checkpoint
  root_hash BLOB NOT NULL,             -- Merkle root (32 bytes)
  cose_sign1 BLOB NOT NULL,            -- Signed checkpoint
  created_at INTEGER NOT NULL          -- Unix timestamp
);

CREATE INDEX idx_checkpoints_created ON checkpoints(created_at DESC);

-- Issuers (for /issuers/{issuerId})
CREATE TABLE issuers (
  issuer_id TEXT PRIMARY KEY,          -- Issuer identifier
  metadata BLOB NOT NULL,              -- CBOR-encoded metadata
  created_at INTEGER NOT NULL,         -- Unix timestamp
  updated_at INTEGER                   -- Unix timestamp
);

-- Service keys (for /.well-known/scitt-keys)
CREATE TABLE service_keys (
  kid TEXT PRIMARY KEY,                -- Key identifier
  cose_key BLOB NOT NULL,              -- CBOR-encoded COSE Key
  status TEXT NOT NULL,                -- "active", "rotated", "revoked"
  created_at INTEGER NOT NULL,
  rotated_at INTEGER
);

CREATE INDEX idx_keys_status ON service_keys(status);
```

## MinIO Bucket Structure

```
scitt-tiles/                    # Tile storage (C2SP format)
  0/                            # Level 0 (leaves)
    000                         # First tile (8,192 bytes)
    001                         # Second tile
    123                         # Tile 123
    123.p.100                   # Partial tile (3,200 bytes)
  1/                            # Level 1
    000                         # First tile at level 1
  entries/                      # Entry tiles (original data)
    000                         # First 256 entries
    001                         # Next 256 entries
    000.p.100                   # Partial entry tile

scitt-artifacts/                # Other artifacts
  issuers/
    {issuer-id}.cbor            # Issuer metadata
  statements/
    {entry-id}.cose             # Archived statements (optional)
```

## Configuration Schema

**Environment Variables**:
```bash
# Service Configuration
SCITT_SERVICE_ID=https://transparency.example.com
SCITT_ORIGIN=example.com/log

# Storage
SCITT_DB_PATH=/data/transparency.db
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false

# Keys
SCITT_SERVICE_KEY_PATH=/secrets/service-key.json
SCITT_KEY_ID=service-key-2025

# Server
SCITT_PORT=8080
SCITT_HOST=0.0.0.0
```

**Configuration File** (config.json):
```json
{
  "service": {
    "id": "https://transparency.example.com",
    "origin": "example.com/log"
  },
  "storage": {
    "database": "/data/transparency.db",
    "minio": {
      "endpoint": "localhost:9000",
      "buckets": {
        "tiles": "scitt-tiles",
        "artifacts": "scitt-artifacts"
      }
    }
  },
  "keys": {
    "servicePath": "/secrets/service-key.json",
    "kid": "service-key-2025"
  },
  "server": {
    "port": 8080,
    "host": "0.0.0.0",
    "cors": {
      "enabled": true,
      "origins": ["*"]
    }
  }
}
```

## 9. Error Response Formats

**Purpose**: Consistent error reporting across implementations (FR-016)

### API Errors (HTTP/Server)

**Media Type**: `application/concise-problem-details+cbor`
**Format**: RFC 7807 Problem Details in CBOR encoding

**Schema**:
```typescript
interface ProblemDetails {
  type: string;        // URI identifying the problem type
  title: string;       // Human-readable summary
  status: number;      // HTTP status code
  detail?: string;     // Human-readable explanation
  instance?: string;   // URI reference to specific occurrence
}
```

**Example**:
```json
{
  "type": "https://transparency.example.com/errors/invalid-statement",
  "title": "Invalid Statement",
  "status": 400,
  "detail": "COSE Sign1 signature verification failed",
  "instance": "/entries/sha256-abc123"
}
```

**Status Codes**:
- 400: Invalid request (malformed COSE, invalid signature, schema violation)
- 303: Registration running (async processing, Location header provided)
- 404: Resource not found (entry, issuer, statement)
- 415: Unsupported media type
- 429: Rate limit exceeded (Retry-After header provided)
- 500: Internal server error

**Response Headers**:
```
Content-Type: application/concise-problem-details+cbor
Location: <uri> (for 303 responses)
Retry-After: <seconds> (for 429 responses)
```

### CLI Errors (Command-Line Interface)

**Output Target**: stderr
**Exit Codes**:
- 0: Success
- 1: General error (validation failure, file not found)
- 2: Usage error (invalid arguments, missing flags)
- 3: Authentication/authorization error
- 4: Network/connectivity error

**Format** (without `--debug` flag):
```
Error: <brief message>
Details: <explanation>
See: <documentation-url>
```

**Example**:
```
Error: Invalid COSE Sign1 structure
Details: Protected header missing required 'kid' field
See: https://docs.example.com/errors/invalid-cose
```

**Format** (with `--debug` flag):
```
Error: <brief message>
Details: <explanation>
Stack trace:
  at <function> (<file>:<line>:<col>)
  at <function> (<file>:<line>:<col>)
See: <documentation-url>
```

**Consistency Requirements**:
- Error messages MUST be identical across TypeScript and Go implementations
- Exit codes MUST match for equivalent error conditions
- `--debug` flag behavior MUST be consistent

### Library Errors (Programmatic)

**TypeScript**:
```typescript
// Error hierarchy
class ScittError extends Error {
  code: string;         // Error code (e.g., "INVALID_SIGNATURE")
  cause?: Error;        // Underlying error
  context?: Record<string, any>; // Additional context
}

// Specific error types
class ValidationError extends ScittError {}
class CryptoError extends ScittError {}
class StorageError extends ScittError {}
class NetworkError extends ScittError {}
```

**Usage**:
```typescript
try {
  const statement = await signStatement(payload, key);
} catch (error) {
  if (error instanceof ValidationError) {
    console.error(`Validation failed: ${error.message}`);
    console.error(`Code: ${error.code}`);
    console.error(`Context:`, error.context);
  }
}
```

**Go**:
```go
// Error types (using error wrapping)
type ScittError struct {
    Code    string                 // Error code
    Message string                 // Error message
    Err     error                  // Underlying error
    Context map[string]interface{} // Additional context
}

func (e *ScittError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *ScittError) Unwrap() error {
    return e.Err
}

// Specific error constructors
func NewValidationError(msg string, err error) *ScittError
func NewCryptoError(msg string, err error) *ScittError
func NewStorageError(msg string, err error) *ScittError
func NewNetworkError(msg string, err error) *ScittError
```

**Usage**:
```go
statement, err := SignStatement(payload, key)
if err != nil {
    var scittErr *ScittError
    if errors.As(err, &scittErr) {
        log.Printf("Error: %s (code: %s)", scittErr.Message, scittErr.Code)
        log.Printf("Context: %v", scittErr.Context)
    }
    return err
}
```

**Error Code Consistency**:

| Code | Description | HTTP Status | CLI Exit |
|------|-------------|-------------|----------|
| `INVALID_SIGNATURE` | Signature verification failed | 400 | 1 |
| `INVALID_COSE` | Malformed COSE structure | 400 | 1 |
| `INVALID_KEY` | Invalid or malformed key | 400 | 1 |
| `KEY_NOT_FOUND` | Key not found | 404 | 1 |
| `ENTRY_NOT_FOUND` | Entry not found | 404 | 1 |
| `ISSUER_NOT_FOUND` | Issuer not found | 404 | 1 |
| `UNSUPPORTED_MEDIA_TYPE` | Wrong Content-Type | 415 | 2 |
| `RATE_LIMIT_EXCEEDED` | Too many requests | 429 | 4 |
| `STORAGE_ERROR` | Database/storage failure | 500 | 1 |
| `NETWORK_ERROR` | Network connectivity issue | 500 | 4 |

**Validation**: Error codes and messages MUST match between TypeScript and Go implementations (validated by interop tests)

---

## Validation Summary

| Entity | Key Validation | Storage |
|--------|---------------|---------|
| Cryptographic Key | ES256, P-256 curve, unique kid | KMS, environment |
| Issuer Identity | Valid URI, at least one key | SQLite + MinIO |
| Statement | COSE Sign1, valid signature, <1MB | SQLite + Entry tiles |
| Receipt | COSE proof, valid inclusion path | Embedded in response |
| Merkle Tree | RFC 6962, SHA-256, tile height 8 | MinIO tiles |
| Checkpoint | COSE Sign1, valid timestamp | SQLite + cache |
| Service Config | SCRAPI-compliant, CBOR format | Static file |
| Test Vector | JSON format, Go-generated | Git repository |
| Error Response | RFC 7807, consistent codes | HTTP/CLI/Library |

---

**Next**: Generate OpenAPI contract (contracts/openapi.yaml)
