# Data Model: IETF Standards-Based Transparency Service

**Feature**: IETF Standards-Based Transparency Service
**Date**: 2025-10-11
**Prerequisites**: research.md, spec.md

## Overview

This document defines the data model for the transparency service, including SQLite schemas for metadata and object storage structures for content. The design follows SCITT/COSE standards and C2SP tlog-tiles specification.

## Storage Architecture

### Two-Tier Storage Strategy

**SQLite (Metadata & Indexes)**:
- Statement metadata for efficient querying
- Tile coordinate mapping
- Receipt pointers
- Tree state and checkpoints

**Object Storage (Content)**:
- Merkle tree tiles (hash tiles + entry tiles)
- Receipts (COSE Merkle proofs)
- Checkpoints (signed notes)
- All using C2SP tlog-tiles naming convention

---

## SQLite Schema

### Version: 1.0.0

```sql
-- Schema versioning and migrations
CREATE TABLE schema_version (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO schema_version (version) VALUES ('1.0.0');

-- Statements table: Metadata for registered signed statements
CREATE TABLE statements (
  entry_id INTEGER PRIMARY KEY AUTOINCREMENT,
  statement_hash TEXT UNIQUE NOT NULL,  -- SHA-256 hash of COSE Sign1 statement

  -- Protected header fields (indexed for queries)
  iss TEXT NOT NULL,                     -- Issuer URL
  sub TEXT,                              -- Subject (optional)
  cty TEXT,                              -- Content type (optional)
  typ TEXT,                              -- Type (optional)

  -- Hash envelope parameters
  payload_hash_alg INTEGER NOT NULL,     -- Label 258: Hash algorithm (e.g., -16 for SHA-256)
  payload_hash TEXT NOT NULL,            -- Hash of original artifact
  preimage_content_type TEXT,            -- Label 259: Content type of original bytes
  payload_location TEXT,                 -- Label 260: Optional payload location URL

  -- Registration metadata
  registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  tree_size_at_registration INTEGER NOT NULL,  -- Tree size when registered

  -- Object storage reference
  entry_tile_key TEXT NOT NULL,          -- e.g., "tile/entries/x000/x001/234"
  entry_tile_offset INTEGER NOT NULL,    -- Position within tile (0-255)

  -- Indexes for efficient querying
  FOREIGN KEY (tree_size_at_registration) REFERENCES tree_state(tree_size)
);

-- Indexes for query performance
CREATE INDEX idx_statements_iss ON statements(iss);
CREATE INDEX idx_statements_sub ON statements(sub);
CREATE INDEX idx_statements_cty ON statements(cty);
CREATE INDEX idx_statements_typ ON statements(typ);
CREATE INDEX idx_statements_registered_at ON statements(registered_at);
CREATE INDEX idx_statements_hash ON statements(statement_hash);

-- Receipts table: Pointers to receipt objects in storage
CREATE TABLE receipts (
  entry_id INTEGER PRIMARY KEY,
  receipt_hash TEXT UNIQUE NOT NULL,     -- SHA-256 hash of receipt
  storage_key TEXT UNIQUE NOT NULL,      -- Object storage key: "receipts/{entry-id}"
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  -- Receipt metadata
  tree_size INTEGER NOT NULL,            -- Tree size at time of receipt issuance
  leaf_index INTEGER NOT NULL,           -- Entry's position in log (0-indexed)

  FOREIGN KEY (entry_id) REFERENCES statements(entry_id)
);

-- Tiles table: Merkle tree tile metadata
CREATE TABLE tiles (
  tile_id INTEGER PRIMARY KEY AUTOINCREMENT,
  level INTEGER NOT NULL,                -- Tile level (0 = leaves, higher = tree)
  tile_index INTEGER NOT NULL,           -- Tile index at this level

  -- Storage reference (C2SP tlog-tiles naming)
  storage_key TEXT UNIQUE NOT NULL,      -- e.g., "tile/0/x000/x001/234"

  -- Tile characteristics
  is_partial BOOLEAN DEFAULT FALSE,      -- TRUE if partial tile (< 256 hashes)
  width INTEGER,                         -- Width if partial (1-255), NULL if full (256)

  -- Tile content hash for integrity
  tile_hash TEXT NOT NULL,               -- SHA-256 of tile content (8,192 bytes or less)

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  UNIQUE(level, tile_index)
);

-- Index for tile lookups
CREATE INDEX idx_tiles_level_index ON tiles(level, tile_index);

-- Tree state: Current Merkle tree state
CREATE TABLE tree_state (
  tree_size INTEGER PRIMARY KEY,         -- Number of entries in log
  root_hash TEXT NOT NULL,               -- Current Merkle root hash
  checkpoint_storage_key TEXT NOT NULL,  -- "checkpoint" in object storage
  checkpoint_signed_note TEXT NOT NULL,  -- Signed note content (for quick access)
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Always maintain current tree size
CREATE TABLE current_tree_size (
  id INTEGER PRIMARY KEY CHECK (id = 1), -- Singleton table
  tree_size INTEGER NOT NULL DEFAULT 0,
  last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO current_tree_size (id, tree_size) VALUES (1, 0);

-- Service configuration: Transparency service settings
CREATE TABLE service_config (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Initialize service config
INSERT INTO service_config (key, value) VALUES
  ('service_url', 'http://localhost:3000'),
  ('tile_height', '8'),                   -- Tile height parameter (not used in C2SP, but for future)
  ('checkpoint_frequency', '1000'),       -- Checkpoint every N entries
  ('hash_algorithm', '-16'),              -- SHA-256 (COSE algorithm ID)
  ('signature_algorithm', '-7');          -- ES256 (COSE algorithm ID)

-- Service keys: Transparency service signing keys
CREATE TABLE service_keys (
  kid TEXT PRIMARY KEY,                   -- Key ID
  public_key_jwk TEXT NOT NULL,          -- Public key (JWK format for compatibility)
  private_key_pem TEXT NOT NULL,         -- Private key (PEM, encrypted at rest ideally)
  algorithm TEXT NOT NULL,               -- e.g., "ES256", "Ed25519"
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  active BOOLEAN DEFAULT TRUE
);
```

---

## Object Storage Structure

### C2SP tlog-tiles Naming Convention

All objects stored in MinIO/S3-compatible storage using standard tile log paths:

#### Merkle Tree Hash Tiles

**Path**: `tile/<L>/<N>[.p/<W>]`

- `<L>`: Level (decimal integer 0-63)
  - 0 = leaf hashes
  - Higher levels = hashes of tiles from level below
- `<N>`: Tile index as zero-padded 3-digit path elements
  - Format: `xDDD/xDDD/DDD` where D is digit
  - Example: index 1234067 → `x001/x234/067`
- `[.p/<W>]`: Partial tile suffix (optional)
  - Present only for rightmost partial tiles
  - `<W>` is width in hashes (1-255)

**Content**: Binary blob
- Full tiles: Exactly 256 × 32 bytes = 8,192 bytes (SHA-256 hashes)
- Partial tiles: W × 32 bytes (1-255 hashes)

**Examples**:
- `tile/0/000` - Full tile at level 0, index 0 (first 256 leaf hashes)
- `tile/0/x001/x234/067` - Full tile at level 0, index 1234067
- `tile/0/x001/x234/067.p/128` - Partial tile with 128 hashes
- `tile/1/000` - Level 1 tile (hashes of 256 level-0 tiles)

#### Entry Tiles (Signed Statements)

**Path**: `tile/entries/<N>[.p/<W>]`

- `<N>`: Tile index (same format as hash tiles)
- `[.p/<W>]`: Partial tile suffix

**Content**: Concatenated CBOR-encoded COSE Sign1 statements
- Each entry is variable length (CBOR)
- Tile boundaries determined by entry count, not byte size
- Full entry tiles: 256 entries
- Partial entry tiles: 1-255 entries

**Examples**:
- `tile/entries/000` - Entries 0-255
- `tile/entries/x001/x234/067` - Entries 1234067 * 256 to (1234067 + 1) * 256 - 1
- `tile/entries/x001/x234/067.p/128` - Partial tile with 128 entries

#### Receipts

**Path**: `receipts/<entry-id>`

- `<entry-id>`: Entry ID from statements table (decimal integer)

**Content**: CBOR-encoded COSE structure with Merkle proof
```
COSE_Sign1 {
  protected: {
    394: [  // "receipts" header (array of receipts)
      COSE_Receipt {
        protected: {
          395: "RFC9162_SHA256",  // VDS algorithm
          396: {                   // VDP (inclusion proof)
            tree_size: <integer>,
            leaf_index: <integer>,
            inclusion_path: [<hash>, <hash>, ...]
          }
        },
        payload: null,  // Detached
        signature: <bytes>  // Transparency service signature
      }
    ]
  },
  payload: null,
  signature: <bytes>
}
```

**Examples**:
- `receipts/0` - Receipt for entry 0
- `receipts/1234067` - Receipt for entry 1234067

#### Checkpoint (Signed Tree Head)

**Path**: `checkpoint`

**Content**: Signed note (text format per C2SP spec)

```
<origin>
<tree_size>
<root_hash>

— <signature_name> <signature_base64>
```

**Example**:
```
transparency.example.com/log
1234567
abc123def456...
timestamp 1633024800

— transparency.example.com wXyz123abc...
```

**Requirements**:
- At least one Ed25519 signature for public logs
- Tree size must be decimal integer
- Root hash is base64-encoded
- Signature verifiable with keys from `/.well-known/scitt-keys`

---

## Entity Relationships

```
statements (entry_id, statement_hash, iss, sub, ...)
    ↓ 1:1
receipts (entry_id, receipt_hash, storage_key, tree_size, leaf_index)
    ↓ references
Object Storage: receipts/<entry-id>

statements (entry_tile_key, entry_tile_offset)
    ↓ references
Object Storage: tile/entries/<N>[.p/<W>]

tree_state (tree_size, root_hash, checkpoint_storage_key)
    ↓ references
Object Storage: checkpoint

tiles (level, tile_index, storage_key, is_partial, width)
    ↓ references
Object Storage: tile/<L>/<N>[.p/<W>]
```

---

## Data Validation Rules

### Statement Registration

1. **Hash Envelope Validation**:
   - Label 258 (payload_hash_alg) MUST be present and supported (e.g., -16 for SHA-256)
   - Label 259 (preimage_content_type) SHOULD be present
   - Label 260 (payload_location) is OPTIONAL
   - Payload MUST be the hash of the original artifact

2. **Protected Header Validation**:
   - `iss` (issuer) MUST be a valid URL
   - `sub` (subject) is OPTIONAL
   - `cty` (content type) is OPTIONAL
   - `typ` (type) is OPTIONAL

3. **Signature Validation**:
   - COSE Sign1 signature MUST verify with issuer's public key
   - Public key retrieved from `{iss}/.well-known/...` (issuer-defined path)

4. **Uniqueness**:
   - `statement_hash` MUST be unique (no duplicate registrations)

### Tile Creation

1. **Full Tiles**:
   - MUST contain exactly 256 hashes (8,192 bytes)
   - Immutable once created
   - Never modified or deleted

2. **Partial Tiles**:
   - Only rightmost tiles can be partial
   - Width MUST be 1-255
   - Become full tiles as tree grows (new tile created)

3. **Level Constraints**:
   - Level 0: Leaf hashes (SHA-256 of CBOR-encoded entry)
   - Level L+1: Hash of full tiles from level L
   - Partial tiles MUST NOT be hashed into higher levels

### Receipt Generation

1. **Inclusion Proof**:
   - MUST include tree_size, leaf_index, and inclusion_path
   - Path MUST be verifiable against tiles
   - Root hash MUST match checkpoint

2. **Signature**:
   - Receipt MUST be signed by transparency service
   - Signature verifiable with keys from `/.well-known/scitt-keys`

### Checkpoint

1. **Signed Note Format**:
   - MUST follow C2SP signed note specification
   - MUST include at least one Ed25519 signature
   - Tree size MUST match current_tree_size in SQLite

2. **Update Frequency**:
   - Generated every N entries (configurable, default 1000)
   - Always generated when queried (fresh checkpoint on demand)

---

## Query Patterns

### Common Queries

**Find statements by issuer**:
```sql
SELECT * FROM statements WHERE iss = 'https://example.com/issuer';
```

**Find statements by subject**:
```sql
SELECT * FROM statements WHERE sub = 'urn:example:artifact:123';
```

**Find statements by content type**:
```sql
SELECT * FROM statements WHERE cty = 'application/vnd.apache.parquet';
```

**Find statements in date range**:
```sql
SELECT * FROM statements
WHERE registered_at BETWEEN '2025-01-01' AND '2025-12-31';
```

**Combined query (iss + cty + date range)**:
```sql
SELECT * FROM statements
WHERE iss = 'https://example.com/issuer'
  AND cty = 'application/vnd.apache.parquet'
  AND registered_at BETWEEN '2025-01-01' AND '2025-12-31'
ORDER BY registered_at DESC;
```

**Get receipt for entry**:
```sql
SELECT receipts.storage_key, receipts.receipt_hash
FROM receipts
WHERE receipts.entry_id = ?;
```

**Get current tree size**:
```sql
SELECT tree_size FROM current_tree_size WHERE id = 1;
```

**Get tile for level and index**:
```sql
SELECT storage_key, is_partial, width
FROM tiles
WHERE level = ? AND tile_index = ?;
```

---

## Migration Strategy

### Schema Versioning

Schema changes tracked in `schema_version` table. Migrations applied sequentially:

```sql
-- Example migration to 1.1.0 (adding new index)
INSERT INTO schema_version (version) VALUES ('1.1.0');
CREATE INDEX idx_statements_payload_hash ON statements(payload_hash);
```

### Data Portability

**SQLite Backup**:
- Single file backup via `VACUUM INTO 'backup.db'`
- Periodic backups before schema migrations

**Object Storage Migration**:
- All tiles, receipts, checkpoints using standard C2SP paths
- MinIO → S3: Simple bucket copy (aws s3 sync or rclone)
- S3 → Azure Blob: Rclone or AzCopy
- No data transformation required (binary blobs)

---

## Next Steps

With data model defined, we can now:
1. Generate API contracts (OpenAPI spec for SCRAPI endpoints)
2. Create quickstart guide with example data flows
3. Update agent context with data model details
