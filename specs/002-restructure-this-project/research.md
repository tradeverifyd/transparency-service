# Research & Technical Decisions

**Feature**: Dual-Language Project Restructure
**Branch**: `002-restructure-this-project`
**Date**: 2025-10-12

This document records all technical research and decisions made during Phase 0 planning.

**Project Status**: Experimental - no prior releases

## 1. Complete API Specification

### Decision: Full SCITT SCRAPI + C2SP Tile Log Endpoints

**Specification References**:
- **SCITT SCRAPI**: https://ietf-wg-scitt.github.io/draft-ietf-scitt-scrapi/draft-ietf-scitt-scrapi.txt
- **C2SP Tile Logs**: https://github.com/C2SP/C2SP/blob/main/tlog-tiles.md
- **COSE Merkle Proofs**: https://www.ietf.org/archive/id/draft-ietf-cose-merkle-tree-proofs-17.txt

**Complete Endpoint List**:

### SCRAPI Mandatory Endpoints

```
GET    /.well-known/scitt-configuration
       Response: 200 application/cbor
       Purpose: Service configuration and capabilities

GET    /.well-known/scitt-keys
       Response: 200 application/cbor
       Purpose: Service signing keys (COSE Key format)

POST   /entries
       Request: application/cose (signed statement)
       Response: 201 application/cose (receipt)
                 303 application/concise-problem-details+cbor (registration running)
                 400 application/concise-problem-details+cbor (invalid request)
       Purpose: Register signed statement

GET    /entries/{entryId}
       Response: 200 application/cose (statement with receipt)
                 302 Location header (registration running)
                 404 application/concise-problem-details+cbor
                 429 application/concise-problem-details+cbor (rate limit)
       Purpose: Query registration status and retrieve statement
```

### SCRAPI Optional Endpoints

```
POST   /receipt-exchange
       Request: application/cose (receipt to exchange)
       Response: 200 application/cose (new receipt format)
       Purpose: Exchange receipt for different format

GET    /signed-statements/{statementId}
       Response: 200 application/cose
                 404 application/concise-problem-details+cbor
       Purpose: Resolve signed statement by content identifier

GET    /issuers/{issuerId}
       Response: 200 application/cbor
                 404 application/concise-problem-details+cbor
       Purpose: Resolve issuer information
```

### C2SP Tile Log Endpoints (adapted for COSE)

```
GET    /checkpoint
       C2SP Spec: text/plain (Ed25519 signature)
       Our Implementation: application/cose (COSE Sign1 with ES256)
       Purpose: Get signed checkpoint of current tree state

GET    /tile/{level}/{index}
       Response: 200 application/octet-stream
       Purpose: Retrieve full tile (256 hashes × 32 bytes = 8,192 bytes)
       Caching: Immutable, long-lived

GET    /tile/{level}/{index}.p/{width}
       Response: 200 application/octet-stream
       Purpose: Retrieve partial tile (1-255 hashes)
       Parameters: width = 1-255
       Caching: Immutable, long-lived

GET    /tile/entries/{index}
       Response: 200 application/octet-stream
       Purpose: Retrieve full tile of log entries (256 entries)
       Recommended: gzip compression, length-prefixed entries
       Caching: Immutable, long-lived

GET    /tile/entries/{index}.p/{width}
       Response: 200 application/octet-stream
       Purpose: Retrieve partial tile of log entries (1-255 entries)
       Recommended: gzip compression, length-prefixed entries
       Caching: Immutable, long-lived
```

## 2. Media Type Mapping

### Complete Media Type Table

| Endpoint | Method | Request | Response | Specification |
|----------|--------|---------|----------|---------------|
| /.well-known/scitt-configuration | GET | - | application/cbor | SCRAPI |
| /.well-known/scitt-keys | GET | - | application/cbor | SCRAPI |
| /entries | POST | application/cose | application/cose (201)<br>application/concise-problem-details+cbor (303/400) | SCRAPI |
| /entries/{entryId} | GET | - | application/cose (200)<br>application/concise-problem-details+cbor (404/429) | SCRAPI |
| /receipt-exchange | POST | application/cose | application/cose | SCRAPI |
| /signed-statements/{statementId} | GET | - | application/cose | SCRAPI |
| /issuers/{issuerId} | GET | - | application/cbor | SCRAPI |
| /checkpoint | GET | - | application/cose | C2SP (adapted) |
| /tile/{L}/{N} | GET | - | application/octet-stream | C2SP |
| /tile/{L}/{N}.p/{W} | GET | - | application/octet-stream | C2SP |
| /tile/entries/{N} | GET | - | application/octet-stream | C2SP |
| /tile/entries/{N}.p/{W} | GET | - | application/octet-stream | C2SP |

**Note**: SCRAPI uses `application/cbor` for configuration/keys and `application/concise-problem-details+cbor` for errors (not JSON).

## 3. SCRAPI Configuration Response Structure

### /.well-known/scitt-configuration

```cbor
{
  "serviceId": "https://example.com/scitt",
  "treeAlgorithm": "RFC9162_SHA256",
  "signatureAlgorithm": "ES256",
  "entryId": "hash-algorithm",  // e.g., "sha-256"
  "supportedPayloadFormats": ["application/json", "application/cbor"],
  "registrationPolicies": [...],
  "extensions": {
    "c2sp-tiles": true,
    "checkpoint-endpoint": "/checkpoint"
  }
}
```

### /.well-known/scitt-keys

```cbor
{
  "keys": [
    {
      "kty": "EC",
      "crv": "P-256",
      "kid": "service-key-2025",
      "x": "<base64>",
      "y": "<base64>",
      "use": "sig",
      "alg": "ES256"
    }
  ]
}
```

## 4. Cryptographic Standards

### Decision: ES256 with RFC 6962 Merkle Trees

**Algorithm Specifications**:
```
Signature: ES256
  - Algorithm: ECDSA
  - Curve: P-256 (secp256r1)
  - Hash: SHA-256
  - COSE Algorithm ID: -7

Merkle Tree: RFC 6962
  - Hash: SHA-256
  - Leaf: SHA-256(0x00 || data)
  - Node: SHA-256(0x01 || left || right)
  - VDS: RFC9162_SHA256

Tile Format: C2SP
  - Height: 8 (256 leaves/tile)
  - Tile size: 8,192 bytes (256 × 32)
```

**Key Deviations from C2SP**:
- Checkpoint signatures: COSE Sign1 + ES256 (not Ed25519 text format)
- Service keys: COSE Key format (not raw Ed25519)
- All signatures: COSE Sign1 structure

## 5. C2SP Checkpoint Format (adapted for COSE)

### Checkpoint Structure

**C2SP Original** (text format with Ed25519):
```
example.com/log
12345
Xbjhshxs+5vV4VVX...
— example.com LOGID Signed...
```

**Our Implementation** (COSE Sign1 with ES256):
```
GET /checkpoint
Response: application/cose

COSE_Sign1 {
  protected: {
    alg: -7,  // ES256
    kid: "service-key-2025"
  },
  payload: {
    "origin": "example.com/log",
    "tree_size": 12345,
    "root_hash": "<base64-sha256>",
    "timestamp": "2025-10-12T00:00:00Z"
  },
  signature: <ECDSA P-256 signature bytes>
}
```

## 6. Tile Path Format

### C2SP Tile Addressing

```
Level (L): 0-63 (no leading zeros)
  - 0 = leaf level
  - Higher levels = intermediate nodes

Index (N): Zero-padded 3-digit groups
  Examples:
    000
    001
    123
    001/234
    001/234/567

Width (W): 1-255 (for partial tiles)
  - Omitted for full tiles (256 hashes)
  - .p/<W> suffix for partial tiles

Full Tile Examples:
  /tile/0/000          - First tile at leaf level (8,192 bytes)
  /tile/0/001          - Second tile at leaf level
  /tile/1/000          - First tile at level 1

Partial Tile Examples:
  /tile/0/000.p/100    - 100 hashes at leaf level (3,200 bytes)
  /tile/0/001.p/50     - 50 hashes (1,600 bytes)

Entry Tile Examples:
  /tile/entries/000    - First 256 log entries
  /tile/entries/001    - Next 256 log entries
  /tile/entries/000.p/100  - First 100 entries only
```

## 7. Storage Architecture

### Decision: SQLite + MinIO

**SQLite Schema**:
```sql
-- Registered statements
CREATE TABLE statements (
  entry_id TEXT PRIMARY KEY,
  cose_sign1 BLOB NOT NULL,
  tree_index INTEGER NOT NULL,
  registered_at INTEGER NOT NULL,
  issuer_id TEXT
);

-- Checkpoints
CREATE TABLE checkpoints (
  tree_size INTEGER PRIMARY KEY,
  root_hash BLOB NOT NULL,
  cose_sign1 BLOB NOT NULL,
  created_at INTEGER NOT NULL
);

-- Issuers (for /issuers/{issuerId})
CREATE TABLE issuers (
  issuer_id TEXT PRIMARY KEY,
  metadata BLOB NOT NULL,  -- CBOR
  created_at INTEGER NOT NULL
);

CREATE INDEX idx_statements_tree ON statements(tree_index);
CREATE INDEX idx_statements_issuer ON statements(issuer_id);
```

**MinIO Buckets**:
```
scitt-tiles/
  0/000                # Tile level 0, index 0
  0/001                # Tile level 0, index 1
  0/000.p.100          # Partial tile
  1/000                # Tile level 1, index 0
  entries/000          # Entry tile 0
  entries/001          # Entry tile 1

scitt-receipts/        # Receipt artifacts
scitt-statements/      # Statement artifacts
```

**Connection Details**:
- SQLite: `transparency.db` (embedded, ACID transactions)
- MinIO: S3-compatible, localhost:9000 (dev), no direct S3/Azure

## 8. COSE Merkle Proof Structure

### IETF COSE Merkle Tree Proofs Format

**Receipt Structure** (draft-ietf-cose-merkle-tree-proofs-17):
```
COSE_Sign1 {
  protected: {
    alg: -7,  // ES256
    kid: "service-key-2025",
    vds: "RFC9162_SHA256",  // Verifiable Data Structure
    tree_size: 12345
  },
  payload: {
    "leaf_index": 42,
    "inclusion_path": [
      "<base64-hash-1>",
      "<base64-hash-2>",
      ...
    ]
  },
  signature: <ECDSA P-256 signature>
}
```

**Verification**:
1. Extract leaf_index and inclusion_path from payload
2. Compute root using RFC 6962 algorithm
3. Verify COSE signature
4. Compare computed root with checkpoint root

## 9. Error Response Format

### SCRAPI Error Structure

**Media Type**: `application/concise-problem-details+cbor`

```cbor
{
  "type": "https://example.com/errors/invalid-statement",
  "title": "Invalid Statement",
  "status": 400,
  "detail": "COSE Sign1 signature verification failed",
  "instance": "/entries/abc123"
}
```

**Common Error Types**:
- 400: Invalid request (malformed COSE, invalid signature)
- 303: Registration running (Location header to GET /entries/{entryId})
- 404: Resource not found
- 429: Rate limit exceeded

## 10. CI/CD and Installation

### GitHub Actions

```
.github/workflows/
  ci-typescript.yml      # scitt-typescript/** changes
  ci-golang.yml          # scitt-golang/** changes
  ci-interop.yml         # Both + tests/interop/** changes
  ci-scrapi-contract.yml # Validate SCRAPI compliance
  ci-c2sp-contract.yml   # Validate C2SP tile format
  release.yml            # Lockstep versioning
```

**Contract Testing**:
- SCRAPI compliance tests against specification
- C2SP tile format validation
- COSE structure validation
- Cross-implementation interoperability

### Installation Scripts

```bash
# docs/install/install-typescript.sh
# docs/install/install-golang.sh

# Platform support: macOS and Linux only
# Architectures: x86_64, arm64
# Downloads from GitHub releases
```

## 11. OpenAPI Contract

### Decision: docs/openapi.yaml for SCRAPI + C2SP

**Structure**:
```yaml
openapi: 3.1.0
info:
  title: SCITT Transparency Service
  version: 1.0.0
  description: |
    IETF SCITT SCRAPI + C2SP Tile Log Extensions
    - SCRAPI: draft-ietf-scitt-scrapi
    - C2SP: tlog-tiles.md
    - COSE Proofs: draft-ietf-cose-merkle-tree-proofs-17

servers:
  - url: https://transparency.example.com

paths:
  /.well-known/scitt-configuration:
    get:
      responses:
        '200':
          content:
            application/cbor:
              schema:
                $ref: '#/components/schemas/ServiceConfiguration'

  /.well-known/scitt-keys:
    get:
      responses:
        '200':
          content:
            application/cbor:
              schema:
                $ref: '#/components/schemas/ServiceKeys'

  /entries:
    post:
      requestBody:
        content:
          application/cose:
            schema:
              type: string
              format: binary
      responses:
        '201':
          content:
            application/cose:
              schema:
                type: string
                format: binary
        '303':
          headers:
            Location:
              schema:
                type: string
          content:
            application/concise-problem-details+cbor:
              schema:
                $ref: '#/components/schemas/ProblemDetails'

  # ... all other endpoints
```

## 12. Test Vectors

### Go-Generated Cross-Implementation Vectors

```
tests/interop/fixtures/
  scrapi/
    configuration.json      # /.well-known/scitt-configuration
    keys.json              # /.well-known/scitt-keys
    register-statement.json # POST /entries
    query-entry.json       # GET /entries/{entryId}
    receipt-exchange.json  # POST /receipt-exchange
  c2sp/
    checkpoint.json        # GET /checkpoint
    full-tiles.json        # GET /tile/{L}/{N}
    partial-tiles.json     # GET /tile/{L}/{N}.p/{W}
    entry-tiles.json       # GET /tile/entries/{N}
  cose-proofs/
    inclusion-proof.json   # COSE Merkle inclusion proofs
    consistency-proof.json # Merkle consistency proofs
  keys/
    es256-keygen.json      # ES256 key generation
```

## Summary

**Endpoint Coverage**:
- ✅ All SCRAPI mandatory endpoints
- ✅ All SCRAPI optional endpoints
- ✅ All C2SP tile log endpoints (full + partial)
- ✅ C2SP checkpoint (adapted to COSE Sign1)

**Specification Compliance**:
- ✅ IETF SCITT SCRAPI (draft-ietf-scitt-scrapi)
- ✅ C2SP Tile Logs (with COSE instead of Ed25519)
- ✅ IETF COSE Merkle Tree Proofs (draft-17)
- ✅ RFC 6962 Merkle trees (SHA-256)
- ✅ RFC 8152 COSE (ES256)

**Key Adaptations**:
- C2SP checkpoints: COSE Sign1 + ES256 (not Ed25519)
- Media types: CBOR for config/keys (per SCRAPI)
- Error format: application/concise-problem-details+cbor

**Next Phase**: Generate data-model.md, contracts/openapi.yaml, and quickstart.md (Phase 1)
