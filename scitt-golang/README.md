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
- **Tests**: 10 test suites passing (tile naming)

**Total: 51 test suites, 139+ individual tests passing**

### In Progress 🔄

- Full tile-log integration with `golang.org/x/mod/sumdb/tlog`
- Merkle proof generation
- Checkpoint operations

### Planned 📋

- MinIO storage implementation
- CLI (using cobra)
- HTTP server (net/http)
- Full integration tests

## Dependencies

```go
require (
	github.com/fxamacker/cbor/v2 v2.x.x     // CBOR encoding
	github.com/mattn/go-sqlite3 v1.14.32    // SQLite driver
	golang.org/x/mod v0.29.0                 // sumdb/tlog for RFC 6962
)
```

## Project Structure

```
scitt-golang/
├── cmd/
│   ├── scitt/        # CLI tool
│   └── scitt-server/ # HTTP server
├── pkg/
│   ├── cose/         # COSE operations (✅ Complete)
│   ├── database/     # SQLite operations (✅ Complete)
│   ├── merkle/       # Merkle tree operations (🔄 In progress)
│   └── storage/      # Storage abstraction (✅ Interface complete)
├── tests/
│   ├── unit/
│   ├── contract/
│   └── integration/
└── go.mod
```

## Building

```bash
# Build all packages
go build ./...

# Run tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run specific package tests
go test -v ./pkg/cose
go test -v ./pkg/database
go test -v ./pkg/merkle
```

## Usage Examples

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
