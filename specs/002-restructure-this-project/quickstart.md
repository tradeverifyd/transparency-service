# Quick Start Guide: SCITT Transparency Service

**Feature**: Dual-Language Project Restructure
**Goal**: Get started with the transparency service in under 5 minutes

## Choose Your Language

This transparency service is available in two implementations with 100% interoperability:

| Implementation | Use When | Installation Time |
|----------------|----------|-------------------|
| **TypeScript** | You prefer TypeScript/JavaScript, use Bun runtime, need Node.js integration | ~2 minutes |
| **Go** | You prefer Go, need compiled binaries, want minimal dependencies | ~2 minutes |

**Both implementations**:
- Support identical CLI commands
- Generate interoperable artifacts (keys, statements, receipts)
- Implement IETF SCITT SCRAPI + C2SP Tile Logs
- Use ES256 (ECDSA P-256 + SHA-256) cryptography

---

## TypeScript Quick Start

### Prerequisites
- **Bun** (latest stable): https://bun.sh
- **macOS** or **Linux** (Windows not supported)

### 1. Install CLI

```bash
# Install scitt-ts CLI
curl -fsSL https://raw.githubusercontent.com/org/transparency-service/main/docs/install/install-typescript.sh | sh

# Verify installation
scitt-ts --version
```

### 2. Generate Keys

```bash
# Generate ES256 key pair
scitt-ts keys generate --output ./my-key.json

# Output: my-key.json (COSE Key format)
```

### 3. Create Issuer Identity

```bash
# Create issuer identity
scitt-ts identity create \
  --key ./my-key.json \
  --id "did:web:example.com" \
  --output ./issuer-identity/

# Output: issuer-identity/
#   ├── issuer.json     (identity metadata)
#   └── jwks.json       (public keys for .well-known)
```

**Host identity** at `https://example.com/.well-known/issuer.json`:
```bash
# Copy to your web server
cp ./issuer-identity/issuer.json /var/www/.well-known/
```

### 4. Create and Sign Statement

```bash
# Create a statement
echo '{"message": "Hello, SCITT!"}' > statement-payload.json

# Sign the statement
scitt-ts statement sign \
  --key ./my-key.json \
  --payload ./statement-payload.json \
  --output ./signed-statement.cose

# Output: signed-statement.cose (COSE Sign1 format)
```

### 5. Run Local Server

```bash
# Start transparency service (development mode)
cd scitt-typescript
bun run src/server/index.ts

# Server running at http://localhost:8080
```

### 6. Register Statement

```bash
# Register statement to transparency log
curl -X POST http://localhost:8080/entries \
  -H "Content-Type: application/cose" \
  --data-binary @signed-statement.cose \
  -o receipt.cose

# Output: receipt.cose (COSE Sign1 receipt with inclusion proof)
```

### 7. Verify Receipt

```bash
# Verify the receipt
scitt-ts receipt verify \
  --statement ./signed-statement.cose \
  --receipt ./receipt.cose \
  --checkpoint http://localhost:8080/checkpoint

# Output: ✓ Receipt verified successfully
```

### Library Usage (TypeScript)

```typescript
import { generateKeys, signStatement, verifyReceipt } from 'scitt-typescript/lib';

// Generate keys
const keys = await generateKeys('ES256');

// Sign statement
const payload = { message: "Hello, SCITT!" };
const statement = await signStatement(payload, keys.private);

// Verify receipt (after registration)
const isValid = await verifyReceipt(statement, receipt, checkpoint);
console.log('Valid:', isValid);
```

---

## Go Quick Start

### Prerequisites
- **Go 1.21+**: https://go.dev/dl/
- **macOS** or **Linux** (Windows not supported)

### 1. Install CLI

```bash
# Install scitt CLI
curl -fsSL https://raw.githubusercontent.com/org/transparency-service/main/docs/install/install-golang.sh | sh

# Verify installation
scitt --version
```

### 2. Generate Keys

```bash
# Generate ES256 key pair
scitt keys generate --output ./my-key.json

# Output: my-key.json (COSE Key format)
```

### 3. Create Issuer Identity

```bash
# Create issuer identity
scitt identity create \
  --key ./my-key.json \
  --id "did:web:example.com" \
  --output ./issuer-identity/

# Output: issuer-identity/
#   ├── issuer.json     (identity metadata)
#   └── jwks.json       (public keys for .well-known)
```

**Host identity** at `https://example.com/.well-known/issuer.json`:
```bash
# Copy to your web server
cp ./issuer-identity/issuer.json /var/www/.well-known/
```

### 4. Create and Sign Statement

```bash
# Create a statement
echo '{"message": "Hello, SCITT!"}' > statement-payload.json

# Sign the statement
scitt statement sign \
  --key ./my-key.json \
  --payload ./statement-payload.json \
  --output ./signed-statement.cose

# Output: signed-statement.cose (COSE Sign1 format)
```

### 5. Run Local Server

```bash
# Start transparency service (development mode)
cd scitt-golang
go run ./cmd/scitt-server

# Server running at http://localhost:8080
```

### 6. Register Statement

```bash
# Register statement to transparency log
curl -X POST http://localhost:8080/entries \
  -H "Content-Type: application/cose" \
  --data-binary @signed-statement.cose \
  -o receipt.cose

# Output: receipt.cose (COSE Sign1 receipt with inclusion proof)
```

### 7. Verify Receipt

```bash
# Verify the receipt
scitt receipt verify \
  --statement ./signed-statement.cose \
  --receipt ./receipt.cose \
  --checkpoint http://localhost:8080/checkpoint

# Output: ✓ Receipt verified successfully
```

### Library Usage (Go)

```go
package main

import (
    "github.com/org/transparency-service/scitt-golang/pkg/keys"
    "github.com/org/transparency-service/scitt-golang/pkg/statement"
    "github.com/org/transparency-service/scitt-golang/pkg/receipt"
)

func main() {
    // Generate keys
    privateKey, publicKey, err := keys.Generate("ES256")

    // Sign statement
    payload := []byte(`{"message": "Hello, SCITT!"}`)
    stmt, err := statement.Sign(payload, privateKey)

    // Verify receipt (after registration)
    valid, err := receipt.Verify(stmt, rcpt, checkpoint)
    fmt.Println("Valid:", valid)
}
```

---

## Production Deployment

### TypeScript Deployment

```bash
# Build production bundle
cd scitt-typescript
bun build src/server/index.ts --target=bun --outdir=dist

# Run with environment variables
export SCITT_SERVICE_ID=https://transparency.example.com
export SCITT_SERVICE_KEY_PATH=/secrets/service-key.json
export SCITT_DB_PATH=/data/transparency.db
export MINIO_ENDPOINT=minio.example.com:9000
export MINIO_ACCESS_KEY=<your-key>
export MINIO_SECRET_KEY=<your-secret>

bun dist/index.js
```

**Docker** (TypeScript):
```dockerfile
FROM oven/bun:latest
WORKDIR /app
COPY scitt-typescript /app
RUN bun install
CMD ["bun", "src/server/index.ts"]
```

### Go Deployment

```bash
# Build production binary
cd scitt-golang
go build -o scitt-server ./cmd/scitt-server

# Run with environment variables
export SCITT_SERVICE_ID=https://transparency.example.com
export SCITT_SERVICE_KEY_PATH=/secrets/service-key.json
export SCITT_DB_PATH=/data/transparency.db
export MINIO_ENDPOINT=minio.example.com:9000
export MINIO_ACCESS_KEY=<your-key>
export MINIO_SECRET_KEY=<your-secret>

./scitt-server
```

**Docker** (Go):
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY scitt-golang /app
RUN go build -o scitt-server ./cmd/scitt-server

FROM alpine:latest
COPY --from=builder /app/scitt-server /usr/local/bin/
CMD ["scitt-server"]
```

---

## Common CLI Commands

Both implementations support identical CLI commands:

### Key Management
```bash
# Generate new key
{scitt|scitt-ts} keys generate --output key.json

# Show public key
{scitt|scitt-ts} keys public --key key.json
```

### Identity Management
```bash
# Create identity
{scitt|scitt-ts} identity create \
  --key key.json \
  --id did:web:example.com \
  --output ./identity/

# Verify identity
{scitt|scitt-ts} identity verify --url https://example.com/.well-known/issuer.json
```

### Statement Operations
```bash
# Sign statement
{scitt|scitt-ts} statement sign \
  --key key.json \
  --payload data.json \
  --output statement.cose

# Verify statement
{scitt|scitt-ts} statement verify \
  --statement statement.cose \
  --key key.json
```

### Receipt Operations
```bash
# Verify receipt
{scitt|scitt-ts} receipt verify \
  --statement statement.cose \
  --receipt receipt.cose \
  --checkpoint http://localhost:8080/checkpoint
```

---

## API Examples

### Register Statement (curl)
```bash
curl -X POST https://transparency.example.com/entries \
  -H "Content-Type: application/cose" \
  --data-binary @statement.cose
```

### Query Entry
```bash
curl https://transparency.example.com/entries/sha256-abc123...
```

### Get Checkpoint
```bash
curl https://transparency.example.com/checkpoint
```

### Get Service Configuration
```bash
curl https://transparency.example.com/.well-known/scitt-configuration
```

### Get Tile
```bash
curl https://transparency.example.com/tile/0/000 --output tile-0-0.bin
```

---

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SCITT_SERVICE_ID` | Service identifier (URL) | `http://localhost:8080` |
| `SCITT_ORIGIN` | Service origin for checkpoints | `localhost/log` |
| `SCITT_SERVICE_KEY_PATH` | Path to service key file | `./service-key.json` |
| `SCITT_KEY_ID` | Service key identifier | `service-key-2025` |
| `SCITT_DB_PATH` | SQLite database path | `./transparency.db` |
| `SCITT_PORT` | Server port | `8080` |
| `SCITT_HOST` | Server host | `0.0.0.0` |
| `MINIO_ENDPOINT` | MinIO endpoint | `localhost:9000` |
| `MINIO_ACCESS_KEY` | MinIO access key | `minioadmin` |
| `MINIO_SECRET_KEY` | MinIO secret key | `minioadmin` |
| `MINIO_USE_SSL` | Use SSL for MinIO | `false` |

### Configuration File (config.json)

```json
{
  "service": {
    "id": "https://transparency.example.com",
    "origin": "example.com/log"
  },
  "storage": {
    "database": "/data/transparency.db",
    "minio": {
      "endpoint": "minio.example.com:9000",
      "useSSL": true,
      "accessKey": "<from-env>",
      "secretKey": "<from-env>",
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
    "host": "0.0.0.0"
  }
}
```

---

## Interoperability

**Key Feature**: Artifacts created by one implementation work with the other!

### Cross-Implementation Example

```bash
# Generate key in TypeScript
scitt-ts keys generate --output key.json

# Sign statement in TypeScript
scitt-ts statement sign --key key.json --payload data.json --output stmt.cose

# Register to Go server
curl -X POST http://go-server:8080/entries \
  --data-binary @stmt.cose

# Verify receipt in TypeScript
scitt-ts receipt verify --statement stmt.cose --receipt receipt.cose
```

---

## Next Steps

- **API Reference**: See `docs/openapi.yaml` for complete API documentation
- **Data Model**: See `specs/002-restructure-this-project/data-model.md`
- **Development**: See `README.md` in `scitt-typescript/` or `scitt-golang/`
- **Standards**:
  - SCITT SCRAPI: https://ietf-wg-scitt.github.io/draft-ietf-scitt-scrapi/
  - C2SP Tiles: https://github.com/C2SP/C2SP/blob/main/tlog-tiles.md
  - COSE Merkle Proofs: https://www.ietf.org/archive/id/draft-ietf-cose-merkle-tree-proofs-17.txt

---

**Congratulations!** You've successfully set up the SCITT Transparency Service.

For questions or issues, see: https://github.com/org/transparency-service/issues
