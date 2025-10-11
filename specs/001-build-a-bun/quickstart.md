# Quickstart: IETF Standards-Based Transparency Service

**Feature**: IETF Standards-Based Transparency Service
**Date**: 2025-10-11
**Prerequisites**: Bun installed, MinIO or S3-compatible storage accessible

## Overview

This quickstart demonstrates the complete transparency service workflow:
1. **Service Operator**: Initialize and start transparency service
2. **Issuer**: Generate identity, sign artifacts, register statements
3. **Verifier**: Verify statements and receipts
4. **Auditor**: Query log and verify consistency

---

## Prerequisites

**Install Bun**:
```bash
curl -fsSL https://bun.sh/install | bash
```

**Object Storage**: MinIO (local), S3, or Azure Blob
```bash
# Option 1: Local MinIO (Docker)
docker run -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"

# Option 2: Use existing S3/Azure credentials
```

---

## Step 1: Initialize Transparency Service

The transparency service operator sets up the log and generates service signing keys.

```bash
# Initialize transparency service
$ transparency init \
  --database ./transparency.db \
  --storage minio \
  --storage-endpoint http://localhost:9000 \
  --storage-access-key minioadmin \
  --storage-secret-key minioadmin \
  --storage-bucket transparency-log \
  --service-url http://localhost:3000

✓ Created SQLite database: ./transparency.db
✓ Generated service signing key (ES256): service-key-1
✓ Initialized Merkle tree (tile height: 8)
✓ Created MinIO bucket: transparency-log
✓ Service ready

Service public keys saved to: ./.transparency/service-keys.json
```

**What happened**:
- Created SQLite database with schema
- Generated ES256 key pair for signing receipts
- Initialized empty Merkle tree
- Configured object storage connection
- Created `.well-known/scitt-keys` content for hosting

---

## Step 2: Start Transparency Service

```bash
# Start HTTP service
$ transparency serve \
  --database ./transparency.db \
  --port 3000 \
  --host 0.0.0.0

🚀 Transparency service started
📍 URL: http://localhost:3000
🔑 Keys: http://localhost:3000/.well-known/scitt-keys
📊 Config: http://localhost:3000/.well-known/scitt-configuration
✅ Health: http://localhost:3000/health

Press Ctrl+C to stop
```

**Verify service is running**:
```bash
$ curl http://localhost:3000/health

{
  "status": "healthy",
  "components": {
    "database": { "status": "up" },
    "object_storage": { "status": "up" },
    "service": { "status": "up" }
  }
}
```

---

## Step 3: Generate Issuer Identity

An issuer (software publisher, manufacturer) creates an identity for signing statements.

```bash
# Create issuer identity
$ issuer init \
  --issuer-url https://example.com/issuer \
  --output ./issuer-identity

✓ Generated ES256 key pair
✓ Created issuer identity configuration
✓ Public key ready for hosting

Files created:
  ./issuer-identity/private-key.pem     (KEEP SECRET!)
  ./issuer-identity/public-key.json     (host at issuer URL)
  ./issuer-identity/.well-known/        (ready to deploy)
  ./issuer-identity/config.json         (issuer configuration)
```

**Host public key** (for verification):
```bash
# Option 1: Serve locally for testing
$ cd issuer-identity && bun serve --port 8080

# Option 2: Deploy .well-known/ to https://example.com/issuer/.well-known/
```

**Public key structure** (`public-key.json`):
```json
{
  "keys": [
    {
      "kty": "EC",
      "crv": "P-256",
      "x": "...",
      "y": "...",
      "kid": "issuer-key-1",
      "alg": "ES256"
    }
  ]
}
```

---

## Step 4: Sign Artifact Statement

Issuer signs a large artifact (e.g., parquet file) using hash envelope.

```bash
# Create hash envelope statement
$ transparency statement sign \
  --artifact ./data.parquet \
  --issuer-identity ./issuer-identity/config.json \
  --issuer-key ./issuer-identity/private-key.pem \
  --subject "urn:example:dataset:2025-10-11" \
  --content-type "application/vnd.apache.parquet" \
  --output ./statement.cose

📊 Computing hash of artifact (streaming)...
✓ Hash: sha256:abc123def456...
✓ Created hash envelope (COSE Sign1)
✓ Signed statement with ES256
💾 Signed statement saved: ./statement.cose (2.1 KB)

Statement details:
  Issuer: https://example.com/issuer
  Subject: urn:example:dataset:2025-10-11
  Content Type: application/vnd.apache.parquet
  Artifact Hash: sha256:abc123def456...
  Artifact Size: 1.2 GB
```

**What happened**:
- Streamed artifact file to compute SHA-256 hash (no full file in memory)
- Created COSE Hash Envelope with labels 258, 259, 260
- Signed with issuer's private key (COSE Sign1, ES256)
- Output is CBOR-encoded statement (small, regardless of artifact size)

---

## Step 5: Register Statement

Submit signed statement to transparency service for registration.

```bash
# Register statement
$ transparency statement register \
  --statement ./statement.cose \
  --service-url http://localhost:3000 \
  --output ./receipt.cose

📤 Submitting statement to transparency service...
⏳ Registration in progress (async)...
⏱️  Polling status...
✓ Statement registered!
✓ Receipt received

Receipt details:
  Entry ID: 0
  Tree Size: 1
  Leaf Index: 0
  Receipt saved: ./receipt.cose
```

**What happened**:
- POST /entries with COSE Sign1 statement
- Service validated signature (fetched public key from issuer URL)
- Added statement to Merkle tree (entry tile created)
- Generated receipt with inclusion proof
- Stored receipt in object storage: `receipts/0`

---

## Step 6: Verify Statement (Verifier)

Verifier receives artifact + transparent statement and verifies authenticity.

```bash
# Verify artifact hash matches statement
$ transparency statement verify \
  --artifact ./data.parquet \
  --statement ./statement.cose

✓ Computing artifact hash...
✓ Hash matches statement payload
✓ Signature valid (issuer: https://example.com/issuer)
✓ Statement is authentic

Statement verified:
  Issuer: https://example.com/issuer (key: issuer-key-1)
  Subject: urn:example:dataset:2025-10-11
  Artifact Hash: sha256:abc123def456... ✓ MATCH
```

**Offline capability**: This verification works without network if issuer's public key is cached.

---

## Step 7: Verify Receipt (Complete Verification)

Verify the complete transparent statement: artifact + statement + receipt.

```bash
# Verify receipt and Merkle inclusion proof
$ transparency receipt verify \
  --artifact ./data.parquet \
  --statement ./statement.cose \
  --receipt ./receipt.cose \
  --service-url http://localhost:3000

✓ Artifact hash matches statement payload
✓ Statement signature valid
✓ Receipt signature valid (service key: service-key-1)
✓ Merkle inclusion proof verified
✓ Tree size: 1, Leaf index: 0
✓ Checkpoint: Tree size 1, Root hash: xyz789...

🎉 Transparent statement fully verified!

Verification summary:
  ✓ Artifact authenticity (issuer signed hash)
  ✓ Registration proof (receipt with Merkle proof)
  ✓ Service commitment (signed checkpoint)
  ✓ Offline verifiable: YES (with cached keys)
```

**What happened**:
- Verified artifact hash (streaming, no full file in memory)
- Verified COSE Sign1 signature on statement
- Fetched tiles from `/tile/{L}/{N}` to reconstruct Merkle path
- Verified inclusion proof against checkpoint
- Verified checkpoint signature with service public key

---

## Step 8: Query Log (Auditor/User)

Search log by metadata (issuer, subject, content type, date range).

```bash
# Query all statements from an issuer
$ transparency log query \
  --service-url http://localhost:3000 \
  --query '{"iss": "https://example.com/issuer"}'

Found 1 statement(s):

Entry 0:
  Issuer: https://example.com/issuer
  Subject: urn:example:dataset:2025-10-11
  Content Type: application/vnd.apache.parquet
  Registered: 2025-10-11T12:34:56Z
  Receipt: receipts/0
```

**Advanced query** (issuer + content type + date range):
```bash
$ transparency log query \
  --service-url http://localhost:3000 \
  --query '{
    "iss": "https://example.com/issuer",
    "cty": "application/vnd.apache.parquet",
    "registered_after": "2025-10-01",
    "registered_before": "2025-10-31"
  }'
```

---

## Step 9: Audit Log Consistency

Auditor verifies Merkle tree consistency between checkpoints.

```bash
# Get current checkpoint
$ curl http://localhost:3000/checkpoint

transparency.example.com/log
1
xyz789abc...
timestamp 1633024800

— transparency.example.com wABC123...

# Register more statements, then check consistency
$ transparency statement register \
  --statement ./statement2.cose \
  --service-url http://localhost:3000

# Get new checkpoint
$ curl http://localhost:3000/checkpoint

transparency.example.com/log
2
def456ghi...
timestamp 1633028400

— transparency.example.com wDEF456...

# Verify consistency (tree grew from size 1 → 2, append-only)
$ transparency log verify-consistency \
  --service-url http://localhost:3000 \
  --old-size 1 \
  --new-size 2

✓ Fetched consistency proof
✓ Consistency verified: Log is append-only
✓ No entries removed or modified
```

---

## Tile Log Access (Advanced)

Clients can fetch tiles directly for custom verification workflows.

**Get hash tile**:
```bash
$ curl http://localhost:3000/tile/0/000

<binary data: 8,192 bytes = 256 SHA-256 hashes>
```

**Get entry tile**:
```bash
$ curl http://localhost:3000/tile/entries/000

<binary data: concatenated CBOR-encoded COSE Sign1 statements>
```

**Partial tile** (rightmost, incomplete):
```bash
$ curl http://localhost:3000/tile/0/000.p/128

<binary data: 4,096 bytes = 128 SHA-256 hashes>
```

---

## CLI Command Reference

### Transparency Service

```bash
# Initialize service
transparency init --database <path> --storage <type> [options]

# Start HTTP service
transparency serve --database <path> --port <port>

# Query log
transparency log query --service-url <url> --query <json>

# Verify consistency
transparency log verify-consistency --service-url <url> --old-size <n> --new-size <m>
```

### Issuer Identity

```bash
# Generate identity
issuer init --issuer-url <url> --output <dir>
```

### Statement Lifecycle

```bash
# Sign statement
transparency statement sign \
  --artifact <file> \
  --issuer-identity <config> \
  --issuer-key <pem> \
  --subject <urn> \
  --content-type <mime> \
  --output <cose-file>

# Register statement
transparency statement register \
  --statement <cose-file> \
  --service-url <url> \
  --output <receipt-file>

# Verify statement
transparency statement verify \
  --artifact <file> \
  --statement <cose-file>
```

### Receipt Verification

```bash
# Verify receipt
transparency receipt verify \
  --artifact <file> \
  --statement <cose-file> \
  --receipt <cose-file> \
  --service-url <url>
```

---

## Configuration Files

### Service Configuration (`transparency.db` + object storage)

SQLite database stores metadata, object storage stores content.

### Issuer Identity Configuration (`config.json`)

```json
{
  "iss": "https://example.com/issuer",
  "kid": "issuer-key-1",
  "alg": "ES256",
  "private_key_path": "./private-key.pem",
  "public_key_path": "./public-key.json"
}
```

---

## Object Storage Layout

```
transparency-log/
├── tile/
│   ├── 0/                    # Level 0 (leaf hashes)
│   │   ├── 000               # Full tile (256 hashes)
│   │   └── 001.p/128         # Partial tile (128 hashes)
│   ├── 1/                    # Level 1 (hashes of level 0 tiles)
│   │   └── 000
│   └── entries/              # Entry tiles (signed statements)
│       ├── 000
│       └── 001.p/128
├── receipts/
│   ├── 0
│   ├── 1
│   └── 2
└── checkpoint                # Current signed tree head
```

---

## Next Steps

- **Production Deployment**: Configure TLS, authentication, monitoring
- **Key Management**: Use HSM for service and issuer private keys
- **Scaling**: Configure PostgreSQL instead of SQLite for high throughput
- **Federation**: Run multiple transparency services with cross-log consistency
- **Client Libraries**: Build language-specific SDKs wrapping CLI commands

---

## Troubleshooting

**Service won't start**:
- Check SQLite database permissions
- Verify object storage credentials
- Check port 3000 is available

**Statement registration fails**:
- Verify issuer public key is accessible at issuer URL
- Check COSE Sign1 signature is valid
- Ensure statement hash is unique (no duplicates)

**Verification fails**:
- Ensure artifact hasn't been modified (hash must match)
- Check issuer/service public keys are accessible
- Verify network connectivity for tile fetching

**Object storage errors**:
- Test MinIO/S3 connectivity independently
- Verify bucket exists and is writable
- Check access keys have correct permissions

---

## Standards Compliance

This implementation follows:
- **SCITT Architecture**: draft-ietf-scitt-architecture (latest editor's draft)
- **SCITT SCRAPI**: draft-ietf-scitt-scrapi (latest editor's draft)
- **COSE Hash Envelope**: draft-ietf-cose-hash-envelope (latest editor's draft)
- **COSE Merkle Tree Proofs**: draft-ietf-cose-merkle-tree-proofs (latest editor's draft)
- **C2SP tlog-tiles**: https://github.com/C2SP/C2SP/blob/main/tlog-tiles.md
