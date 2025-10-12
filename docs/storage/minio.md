# MinIO Storage Architecture

**Feature**: 002-restructure-this-project
**Date**: 2025-10-12

This document describes the MinIO object storage architecture for the transparency service, including bucket structure, object key formats, lifecycle policies, and setup instructions.

## Overview

The transparency service uses MinIO as an S3-compatible object storage backend for storing:

1. **Merkle tree tiles** (C2SP tile log format)
2. **Entry tiles** (original COSE Sign1 statements)
3. **Issuer metadata** (optional caching)
4. **Archived statements** (optional long-term storage)

**Design Principles**:
- Immutable objects (tiles never change once full)
- Flat namespace with structured keys
- HTTP caching-friendly (long-lived Cache-Control headers)
- No direct S3/Azure dependencies (constitution requirement)

---

## Bucket Structure

### Primary Buckets

```
scitt-tiles/                    # Merkle tree tiles (C2SP format)
  0/                            # Level 0 (leaf hashes)
  1/                            # Level 1 (intermediate nodes)
  entries/                      # Entry tiles (original statements)

scitt-artifacts/                # Optional artifacts
  issuers/                      # Issuer metadata cache
  statements/                   # Statement archives
```

### Bucket Configuration

| Bucket | Purpose | Versioning | Lifecycle | Cache-Control |
|--------|---------|------------|-----------|---------------|
| `scitt-tiles` | Merkle tree storage | Disabled | Never expire | `max-age=31536000, immutable` |
| `scitt-artifacts` | Issuer/statement cache | Disabled | 90 days (optional) | `max-age=3600` |

---

## Object Key Formats

### 1. Merkle Tree Tiles (Level 0-N)

**Full Tile**:
```
{level}/{index}
```

**Partial Tile**:
```
{level}/{index}.p.{width}
```

**Examples**:
```
scitt-tiles/0/000              # Level 0, Tile 0 (full, 256 hashes)
scitt-tiles/0/001              # Level 0, Tile 1
scitt-tiles/0/123.p.100        # Level 0, Tile 123 (partial, 100 hashes)
scitt-tiles/1/000              # Level 1, Tile 0
scitt-tiles/1/x001/x234/067    # Level 1, Tile index 1,234,067 (sharded)
```

**Sharding Strategy** (for large indexes):
- Indexes ≥ 1,000: Shard into `x{thousands}/x{tens}/units` format
- Example: Index 1,234,067 → `x001/x234/067`
- Prevents directory listing performance issues

**Object Size**:
- Full tile: 8,192 bytes (256 hashes × 32 bytes)
- Partial tile: width × 32 bytes (e.g., 100 hashes = 3,200 bytes)

**Content-Type**: `application/octet-stream`

**Immutability**:
- Full tiles (width=256): IMMUTABLE, never modified
- Partial tiles (width<256): May be updated until full, then immutable

### 2. Entry Tiles (Original Statements)

**Full Entry Tile**:
```
entries/{index}
```

**Partial Entry Tile**:
```
entries/{index}.p.{width}
```

**Examples**:
```
scitt-tiles/entries/000        # First 256 statements (entries 0-255)
scitt-tiles/entries/001        # Next 256 statements (entries 256-511)
scitt-tiles/entries/123.p.100  # Partial tile with 100 statements
```

**Object Structure**:
```
Entry Tile Format:
┌─────────────────────────────────┐
│ Statement 0 (COSE Sign1)        │  Variable size
├─────────────────────────────────┤
│ Statement 1 (COSE Sign1)        │  Variable size
├─────────────────────────────────┤
│ ...                             │
├─────────────────────────────────┤
│ Statement 255 (COSE Sign1)      │  Variable size
└─────────────────────────────────┘

Each statement:
- CBOR-encoded COSE Sign1 structure
- Size: Typically 500-2000 bytes
- No length prefix (concatenated CBOR stream)
```

**Content-Type**: `application/cbor-seq` (CBOR sequence)

**Object Size**: Variable (depends on statement sizes, typically 128KB - 512KB per tile)

### 3. Issuer Metadata (Optional)

**Key Format**:
```
issuers/{issuer-id}.cbor
```

**Examples**:
```
scitt-artifacts/issuers/did:web:example.com.cbor
scitt-artifacts/issuers/https:--transparency.example.com.cbor
```

**Key Encoding**:
- Replace `:` with `-` (did:web → did-web)
- Replace `/` with `--` (https://example.com → https---example.com)
- Append `.cbor` extension

**Content-Type**: `application/cbor`

**Object Structure**:
```typescript
{
  id: string;                    // Issuer identifier
  publicKeys: CoseKey[];         // Active public keys
  metadata: {
    name?: string;
    description?: string;
    created: string;             // RFC3339
    updated?: string;            // RFC3339
  };
}
```

### 4. Statement Archives (Optional)

**Key Format**:
```
statements/{entry-id}.cose
```

**Examples**:
```
scitt-artifacts/statements/sha256-abc123def456.cose
```

**Content-Type**: `application/cose`

**Purpose**: Long-term archival of statements outside entry tiles

---

## Storage Interface Implementation

### TypeScript (Bun)

**Location**: `scitt-typescript/src/lib/storage/minio.ts`

**Implementation Requirements**:

```typescript
import { S3Client, PutObjectCommand, GetObjectCommand } from "@aws-sdk/client-s3";
import type { Storage } from "./interface.ts";

export class MinIOStorage implements Storage {
  readonly type = "minio" as const;
  private readonly client: S3Client;
  private readonly bucket: string;

  constructor(options: {
    endpoint: string;
    accessKey: string;
    secretKey: string;
    bucket: string;
    useSSL?: boolean;
  }) {
    this.client = new S3Client({
      endpoint: options.endpoint,
      credentials: {
        accessKeyId: options.accessKey,
        secretAccessKey: options.secretKey,
      },
      region: "us-east-1", // MinIO requires region
      forcePathStyle: true, // Required for MinIO
    });
    this.bucket = options.bucket;
  }

  async put(key: string, data: Uint8Array): Promise<void> {
    await this.client.send(
      new PutObjectCommand({
        Bucket: this.bucket,
        Key: key,
        Body: data,
        ContentType: this.getContentType(key),
        CacheControl: this.getCacheControl(key),
      })
    );
  }

  async get(key: string): Promise<Uint8Array | null> {
    try {
      const response = await this.client.send(
        new GetObjectCommand({
          Bucket: this.bucket,
          Key: key,
        })
      );
      const body = await response.Body?.transformToByteArray();
      return body ? new Uint8Array(body) : null;
    } catch (error) {
      if ((error as any).name === "NoSuchKey") {
        return null;
      }
      throw error;
    }
  }

  private getContentType(key: string): string {
    if (key.endsWith(".cbor")) return "application/cbor";
    if (key.endsWith(".cose")) return "application/cose";
    if (key.startsWith("entries/")) return "application/cbor-seq";
    return "application/octet-stream";
  }

  private getCacheControl(key: string): string {
    // Full tiles are immutable
    if (/^\d+\/\d+$/.test(key) && !key.includes(".p.")) {
      return "max-age=31536000, immutable";
    }
    // Partial tiles may change
    if (key.includes(".p.")) {
      return "max-age=60";
    }
    // Default caching
    return "max-age=3600";
  }
}
```

### Go

**Location**: `scitt-golang/pkg/storage/minio.go`

**Implementation Requirements**:

```go
package storage

import (
    "bytes"
    "context"
    "io"
    "strings"

    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
    client *minio.Client
    bucket string
}

func NewMinIOStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinIOStorage, error) {
    client, err := minio.New(endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
        Secure: useSSL,
    })
    if err != nil {
        return nil, err
    }

    return &MinIOStorage{
        client: client,
        bucket: bucket,
    }, nil
}

func (s *MinIOStorage) Put(ctx context.Context, key string, data []byte) error {
    contentType := s.getContentType(key)
    cacheControl := s.getCacheControl(key)

    _, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
        minio.PutObjectOptions{
            ContentType:  contentType,
            CacheControl: cacheControl,
        })
    return err
}

func (s *MinIOStorage) Get(ctx context.Context, key string) ([]byte, error) {
    obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
    if err != nil {
        return nil, err
    }
    defer obj.Close()

    return io.ReadAll(obj)
}

func (s *MinIOStorage) getContentType(key string) string {
    if strings.HasSuffix(key, ".cbor") {
        return "application/cbor"
    }
    if strings.HasSuffix(key, ".cose") {
        return "application/cose"
    }
    if strings.HasPrefix(key, "entries/") {
        return "application/cbor-seq"
    }
    return "application/octet-stream"
}

func (s *MinIOStorage) getCacheControl(key string) string {
    // Full tiles are immutable (no .p. in key)
    if !strings.Contains(key, ".p.") && strings.Count(key, "/") == 1 {
        return "max-age=31536000, immutable"
    }
    // Partial tiles
    if strings.Contains(key, ".p.") {
        return "max-age=60"
    }
    return "max-age=3600"
}
```

---

## Setup Instructions

### Local Development (Docker Compose)

**File**: `docker-compose.yml`

```yaml
version: '3.8'

services:
  minio:
    image: minio/minio:latest
    container_name: scitt-minio
    ports:
      - "9000:9000"      # API
      - "9001:9001"      # Console
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    command: server /data --console-address ":9001"
    volumes:
      - minio-data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 10s
      timeout: 5s
      retries: 3

  # Initialize buckets
  minio-init:
    image: minio/mc:latest
    depends_on:
      minio:
        condition: service_healthy
    entrypoint: >
      /bin/sh -c "
      mc alias set local http://minio:9000 minioadmin minioadmin &&
      mc mb local/scitt-tiles --ignore-existing &&
      mc mb local/scitt-artifacts --ignore-existing &&
      echo 'Buckets created successfully'
      "

volumes:
  minio-data:
```

**Start MinIO**:
```bash
docker-compose up -d
```

**Access MinIO Console**:
- URL: http://localhost:9001
- Username: `minioadmin`
- Password: `minioadmin`

### Production Setup

**1. Install MinIO Server**:

```bash
# Linux
wget https://dl.min.io/server/minio/release/linux-amd64/minio
chmod +x minio
sudo mv minio /usr/local/bin/

# macOS
brew install minio/stable/minio
```

**2. Create Service Configuration** (`/etc/systemd/system/minio.service`):

```ini
[Unit]
Description=MinIO
Documentation=https://min.io/docs/minio/linux/index.html
Wants=network-online.target
After=network-online.target

[Service]
Type=notify
WorkingDirectory=/usr/local
User=minio-user
Group=minio-user

Environment="MINIO_ROOT_USER=admin"
Environment="MINIO_ROOT_PASSWORD=your-secure-password"
Environment="MINIO_VOLUMES=/mnt/minio/data"

ExecStart=/usr/local/bin/minio server $MINIO_VOLUMES --console-address ":9001"

Restart=always
LimitNOFILE=65536
TasksMax=infinity

[Install]
WantedBy=multi-user.target
```

**3. Create Buckets**:

```bash
# Install MinIO Client
wget https://dl.min.io/client/mc/release/linux-amd64/mc
chmod +x mc
sudo mv mc /usr/local/bin/

# Configure alias
mc alias set production https://minio.example.com admin your-secure-password

# Create buckets
mc mb production/scitt-tiles
mc mb production/scitt-artifacts

# Set public read policy for tiles (optional, for CDN)
mc anonymous set download production/scitt-tiles

# Verify
mc ls production/
```

**4. Enable TLS**:

```bash
# Place certificates in /certs directory
mkdir -p /mnt/minio/certs
cp public.crt /mnt/minio/certs/
cp private.key /mnt/minio/certs/

# MinIO automatically detects and uses certificates
```

---

## Configuration

### Environment Variables

```bash
# MinIO Connection
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false

# Buckets
MINIO_TILE_BUCKET=scitt-tiles
MINIO_ARTIFACT_BUCKET=scitt-artifacts

# Regions (for AWS S3 compatibility)
MINIO_REGION=us-east-1
```

### Configuration File (`config.json`)

```json
{
  "storage": {
    "type": "minio",
    "minio": {
      "endpoint": "localhost:9000",
      "accessKey": "minioadmin",
      "secretKey": "minioadmin",
      "useSSL": false,
      "buckets": {
        "tiles": "scitt-tiles",
        "artifacts": "scitt-artifacts"
      }
    }
  }
}
```

---

## Performance Optimization

### 1. Bucket Policies

**Set lifecycle policy for partial tiles** (auto-delete after 1 hour):

```bash
cat > lifecycle.json <<EOF
{
  "Rules": [
    {
      "ID": "DeletePartialTiles",
      "Status": "Enabled",
      "Filter": {
        "Prefix": "0/",
        "Tag": {
          "Key": "type",
          "Value": "partial"
        }
      },
      "Expiration": {
        "Days": 1
      }
    }
  ]
}
EOF

mc ilm import production/scitt-tiles < lifecycle.json
```

### 2. CDN Integration

**CloudFlare Workers Example**:

```javascript
export default {
  async fetch(request, env) {
    const url = new URL(request.url);

    // Route /tile/* to MinIO
    if (url.pathname.startsWith('/tile/')) {
      const minioUrl = `https://minio.example.com/scitt-tiles${url.pathname}`;
      return fetch(minioUrl, {
        cf: {
          cacheEverything: true,
          cacheTtl: 31536000, // 1 year for immutable tiles
        },
      });
    }

    return new Response('Not Found', { status: 404 });
  },
};
```

### 3. Replication (High Availability)

```bash
# Configure site replication between two MinIO instances
mc admin replicate add primary https://minio-1.example.com \
                         secondary https://minio-2.example.com \
                         --access-key admin --secret-key password
```

---

## Testing

### Local Storage Mock (for CI/CD)

**TypeScript**:
```typescript
import { LocalStorage } from "./local.ts";

// Use LocalStorage for testing
const storage = new LocalStorage(".test-storage");
```

**Go**:
```go
storage := NewLocalStorage(".test-storage")
```

### Integration Test

**TypeScript**:
```typescript
import { test, expect } from "bun:test";
import { MinIOStorage } from "../src/lib/storage/minio.ts";

test("MinIO storage put/get", async () => {
  const storage = new MinIOStorage({
    endpoint: "localhost:9000",
    accessKey: "minioadmin",
    secretKey: "minioadmin",
    bucket: "scitt-tiles",
    useSSL: false,
  });

  const key = "test/tile";
  const data = new Uint8Array([1, 2, 3, 4]);

  await storage.put(key, data);
  const retrieved = await storage.get(key);

  expect(retrieved).toEqual(data);

  await storage.delete(key);
});
```

---

## Troubleshooting

### Common Issues

**1. Connection Refused**:
```
Error: connect ECONNREFUSED 127.0.0.1:9000
```
**Solution**: Ensure MinIO is running and accessible at the configured endpoint.

**2. Access Denied**:
```
Error: AccessDenied: Access Denied
```
**Solution**: Verify access key and secret key are correct. Check bucket policies.

**3. Bucket Not Found**:
```
Error: NoSuchBucket: The specified bucket does not exist
```
**Solution**: Create buckets using `mc mb` or the MinIO console.

**4. TLS Certificate Error**:
```
Error: self signed certificate
```
**Solution**: For self-signed certs, set `NODE_TLS_REJECT_UNAUTHORIZED=0` (dev only) or add cert to trust store.

---

## Security Considerations

1. **Access Keys**:
   - Never commit access keys to version control
   - Use environment variables or secrets management (Vault, AWS Secrets Manager)
   - Rotate keys regularly

2. **Network Security**:
   - Enable TLS in production (`useSSL: true`)
   - Use firewall rules to restrict access to MinIO ports
   - Consider VPC/private network for MinIO-to-service communication

3. **Bucket Policies**:
   - Set read-only public access for tiles (if serving via CDN)
   - Keep artifacts bucket private
   - Use IAM policies for fine-grained access control

4. **Data Integrity**:
   - Enable versioning for critical buckets (optional)
   - Use checksums (MinIO provides automatic MD5/SHA256)
   - Implement backup strategy for long-term data retention

---

## Appendix: Key Format Examples

| Purpose | Key | Size | Immutable |
|---------|-----|------|-----------|
| Full tile L0 | `0/000` | 8,192 bytes | Yes |
| Partial tile L0 | `0/000.p.100` | 3,200 bytes | No |
| Full tile L1 | `1/000` | 8,192 bytes | Yes |
| Entry tile | `entries/000` | Variable | Yes |
| Partial entry | `entries/000.p.50` | Variable | No |
| Issuer metadata | `issuers/did-web-example.com.cbor` | ~2KB | No |
| Statement archive | `statements/sha256-abc123.cose` | ~1KB | Yes |

---

**Next**: Implement MinIO storage class in TypeScript and Go
