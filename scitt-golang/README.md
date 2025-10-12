# SCITT Transparency Service - Go Implementation

Go implementation of the IETF SCITT (Supply Chain Integrity, Transparency, and Trust) transparency service, maintaining 100% API parity with the TypeScript implementation.

## Overview

This is part of a dual-language monorepo providing:
- **RFC 9052/9053**: COSE (CBOR Object Signing and Encryption) operations
- **RFC 6962**: Certificate Transparency-style Merkle trees
- **C2SP tlog-tiles**: Efficient tile-based Merkle tree storage
- **IETF SCITT**: Transparency service for supply chain artifacts

## Status

### Completed ✅

#### COSE Package (`pkg/cose/`)
- ES256 key generation (ECDSA P-256)
- JWK/PEM import/export with RFC 7638 thumbprints
- COSE Sign1 operations (RFC 9052)
- ES256 signer/verifier with IEEE P1363 format
- Hash envelope support (draft-ietf-cose-hash-envelope)
- CWT Claims (RFC 8392, RFC 9597)
- **Tests**: 24 test suites passing

#### Database Package (`pkg/database/`)
- SQLite schema with WAL mode
- Schema versioning and migrations
- Tables: statements, receipts, tiles, tree_state, service_config, service_keys
- Log state management (tree size, checkpoints)
- Statement metadata operations (insert, query, blob storage)
- **Tests**: 17 test suites passing

#### Merkle Package (`pkg/merkle/`)
- C2SP tile naming utilities
  - Tile path generation/parsing
  - Entry tile paths
  - Hybrid index encoding (3-digit/base-256/decimal)
- Storage abstraction interface
- In-memory storage implementation
- Tile-log foundation (RFC 6962 structure)
- RFC 6962 Merkle proofs
  - Inclusion proof generation/verification
  - Consistency proof generation/verification
  - Tree root computation
- Checkpoint operations (Signed Tree Heads)
  - Create signed checkpoints
  - Verify checkpoint signatures
  - Encode/decode to signed note format
- **Tests**: 22 test suites passing (tile naming + proofs + checkpoints)

#### Storage Package (`pkg/storage/`)
- Storage abstraction interface
- In-memory storage (thread-safe, for testing)
- Local filesystem storage (production-ready)
  - Atomic writes via temp file + rename
  - Nested directory support
  - Cross-platform path handling
  - Streaming reader interface
- **Tests**: 10 test suites passing

**Total: 73 test suites, 220+ individual tests passing**

#### CLI Tool (`cmd/scitt/`)
- Cobra-based CLI framework
- Commands:
  - `init` - Initialize a new transparency service
  - `serve` - Start HTTP server (placeholder for T024)
  - `statement sign/verify/hash` - Manage COSE Sign1 statements
  - `receipt verify/info` - Manage receipts (placeholder for T027)
- YAML configuration file support
- **Tests**: 3 test suites passing

### In Progress 🔄

- Full tile-log integration with `golang.org/x/mod/sumdb/tlog`
- HTTP server implementation (T024)

### Planned 📋

- MinIO/S3 storage implementation
- Full integration tests

## Dependencies

```go
require (
	github.com/fxamacker/cbor/v2 v2.x.x     // CBOR encoding
	github.com/mattn/go-sqlite3 v1.14.32    // SQLite driver
	github.com/spf13/cobra v1.10.1          // CLI framework
	golang.org/x/mod v0.29.0                 // sumdb/tlog for RFC 6962
	gopkg.in/yaml.v3 v3.0.1                 // YAML configuration
)
```

## Project Structure

```
scitt-golang/
├── cmd/
│   ├── scitt/        # CLI tool (✅ Complete)
│   └── scitt-server/ # HTTP server (📋 Planned - T024)
├── pkg/
│   ├── cose/         # COSE operations (✅ Complete)
│   ├── database/     # SQLite operations (✅ Complete)
│   ├── merkle/       # Merkle tree operations (✅ Complete)
│   └── storage/      # Storage abstraction (✅ Complete)
├── internal/
│   ├── cli/          # CLI commands (✅ Complete)
│   └── config/       # Configuration (✅ Complete)
├── tests/
│   ├── unit/         # ✅ Package-level tests passing
│   ├── contract/     # 📋 Planned
│   └── integration/  # 📋 Planned
└── go.mod
```

## Building

```bash
# Build all packages
go build ./...

# Build CLI tool
go build -o scitt ./cmd/scitt

# Run tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run specific package tests
go test -v ./pkg/cose
go test -v ./pkg/database
go test -v ./pkg/merkle
```

## CLI Usage

### Initialize a New Transparency Service

```bash
# Initialize service with origin URL
scitt init --origin https://transparency.example.com

# This creates:
# - service-key.pem (private key)
# - service-key.jwk (public key)
# - scitt.db (SQLite database)
# - ./storage/ (tile storage directory)
# - scitt.yaml (configuration file)
```

### Sign and Verify Statements

```bash
# Sign a payload
scitt statement sign \
  --input payload.json \
  --key private-key.pem \
  --output statement.cbor \
  --issuer "https://example.com" \
  --subject "my-artifact"

# Verify a statement
scitt statement verify \
  --input statement.cbor \
  --key public-key.jwk

# Compute statement hash
scitt statement hash --input statement.cbor
```

### Configuration File

Example `scitt.yaml`:

```yaml
origin: https://transparency.example.com

database:
  path: scitt.db
  enable_wal: true

storage:
  type: local
  path: ./storage

keys:
  private: service-key.pem
  public: service-key.jwk

server:
  host: 0.0.0.0
  port: 8080
  cors:
    enabled: true
    allowed_origins:
      - "*"
```

## API Usage Examples

### COSE Sign1

```go
import "github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"

// Generate key pair
keyPair, _ := cose.GenerateES256KeyPair()

// Create signer
signer, _ := cose.NewES256Signer(keyPair.Private)

// Sign data
headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
    Alg: cose.AlgorithmES256,
    Kid: "my-key-1",
})
payload := []byte("Hello, World!")
coseSign1, _ := cose.CreateCoseSign1(headers, payload, signer, cose.CoseSign1Options{})

// Verify
verifier, _ := cose.NewES256Verifier(keyPair.Public)
valid, _ := cose.VerifyCoseSign1(coseSign1, verifier, nil)
```

### Hash Envelope

```go
// Sign large artifact by hash
artifact := []byte("large file contents")
opts := cose.HashEnvelopeOptions{
    ContentType:   "application/octet-stream",
    Location:      "https://example.com/artifact.bin",
    HashAlgorithm: cose.HashAlgorithmSHA256,
}

coseSign1, _ := cose.SignHashEnvelope(artifact, opts, signer, nil, false)

// Verify hash envelope
result, _ := cose.VerifyHashEnvelope(coseSign1, artifact, verifier)
// result.SignatureValid && result.HashValid
```

### Database Operations

```go
import "github.com/tradeverifyd/transparency-service/scitt-golang/pkg/database"

// Open database
db, _ := database.OpenDatabase(database.DatabaseOptions{
    Path:      "./scitt.db",
    EnableWAL: true,
})
defer database.CloseDatabase(db)

// Insert statement
statement := database.Statement{
    StatementHash:          "abc123",
    Iss:                    "https://issuer.example.com",
    PayloadHashAlg:         -16,
    PayloadHash:            "hash",
    TreeSizeAtRegistration: 1,
    EntryTileKey:           "0/0",
    EntryTileOffset:        0,
}
entryID, _ := database.InsertStatement(db, statement)

// Query statements
statements, _ := database.FindStatementsByIssuer(db, "https://issuer.example.com")
```

### Tile Naming

```go
import "github.com/tradeverifyd/transparency-service/scitt-golang/pkg/merkle"

// Generate tile path
path := merkle.TileIndexToPath(0, 1234, nil)
// "tile/0/x004/210"

// Parse tile path
parsed, _ := merkle.ParseTilePath(path)
// parsed.Level = 0, parsed.Index = 1234

// Entry coordinates
entryID := int64(1000)
tileIndex := merkle.EntryIDToTileIndex(entryID)  // 3
tileOffset := merkle.EntryIDToTileOffset(entryID) // 232
```

### Merkle Proofs

```go
import "github.com/tradeverifyd/transparency-service/scitt-golang/pkg/merkle"

// Initialize storage and tile log
store := storage.NewMemoryStorage()
tl := merkle.NewTileLog(store)
tl.Load()

// Append leaves
leaf1 := sha256.Sum256([]byte("data1"))
leaf2 := sha256.Sum256([]byte("data2"))
tl.Append(leaf1)
tl.Append(leaf2)

// Generate inclusion proof
proof, _ := merkle.GenerateInclusionProof(store, 0, 2)

// Verify inclusion proof
root, _ := merkle.ComputeTreeRoot(store, 2)
valid := merkle.VerifyInclusionProof(leaf1, proof, root)

// Generate consistency proof
oldRoot, _ := merkle.ComputeTreeRoot(store, 1)
newRoot, _ := merkle.ComputeTreeRoot(store, 2)
consistencyProof, _ := merkle.GenerateConsistencyProof(store, 1, 2)

// Verify consistency proof
valid = merkle.VerifyConsistencyProof(consistencyProof, oldRoot, newRoot)
```

### Checkpoints (Signed Tree Heads)

```go
import "github.com/tradeverifyd/transparency-service/scitt-golang/pkg/merkle"

// Initialize storage and tile log
store := storage.NewMemoryStorage()
tl := merkle.NewTileLog(store)
tl.Load()

// Append some leaves
for i := 0; i < 10; i++ {
    leaf := sha256.Sum256([]byte{byte(i)})
    tl.Append(leaf)
}

// Create checkpoint
keyPair, _ := cose.GenerateES256KeyPair()
root, _ := merkle.ComputeTreeRoot(store, 10)
checkpoint, _ := merkle.CreateCheckpoint(
    10,                                 // tree size
    root,                               // root hash
    keyPair.Private,                    // signing key
    "https://transparency.example.com", // origin
)

// Encode to signed note format
encoded := merkle.EncodeCheckpoint(checkpoint)
// Output format:
// https://transparency.example.com
// 10
// <base64-root-hash>
// <timestamp>
//
// — https://transparency.example.com <base64-signature>

// Decode and verify
decoded, _ := merkle.DecodeCheckpoint(encoded)
valid, _ := merkle.VerifyCheckpoint(decoded, keyPair.Public)
```

## Architecture

### COSE Operations
- Implements RFC 9052 (COSE Sign1)
- Supports ES256 (ECDSA P-256 + SHA-256)
- Hash envelope for large artifacts
- CWT Claims for transparency service metadata

### Database Layer
- SQLite with WAL mode for performance
- Schema versioning for migrations
- Optimized indexes for common queries
- Foreign key constraints enforced

### Merkle Tree
- C2SP tlog-tiles format for efficient storage
- RFC 6962 compliant Merkle tree operations
- Tile-based storage (256 hashes per tile)
- Integration with golang.org/x/mod/sumdb/tlog

### Storage Abstraction
- Interface for multiple backends (MinIO, filesystem, memory)
- Thread-safe in-memory implementation for testing
- Designed for S3-compatible object storage

## Standards Compliance

- ✅ RFC 9052: CBOR Object Signing and Encryption (COSE)
- ✅ RFC 9053: COSE Algorithms
- ✅ RFC 8392: CBOR Web Token (CWT)
- ✅ RFC 9597: CWT Claims in COSE Headers
- ✅ RFC 6962: Certificate Transparency
- ✅ RFC 7638: JSON Web Key (JWK) Thumbprint
- 🔄 C2SP tlog-tiles specification
- 🔄 draft-ietf-cose-hash-envelope
- 🔄 IETF SCITT SCRAPI

## Contributing

This implementation maintains 100% API parity with the TypeScript implementation in `../scitt-typescript/`. Changes should be coordinated across both implementations.

## License

See repository root LICENSE file.
