-- SCITT Transparency Service Database Schema
-- SQLite schema for dual-language implementation
-- Based on: specs/002-restructure-this-project/data-model.md

-- Registered statements
CREATE TABLE IF NOT EXISTS statements (
  entry_id TEXT PRIMARY KEY,           -- SHA-256 of COSE Sign1
  cose_sign1 BLOB NOT NULL,            -- Full COSE Sign1 statement
  tree_index INTEGER NOT NULL UNIQUE,  -- Position in Merkle tree
  registered_at INTEGER NOT NULL,      -- Unix timestamp
  issuer_id TEXT,                      -- Issuer identifier
  payload_hash BLOB NOT NULL           -- SHA-256 of payload
);

CREATE INDEX IF NOT EXISTS idx_statements_tree ON statements(tree_index);
CREATE INDEX IF NOT EXISTS idx_statements_issuer ON statements(issuer_id);
CREATE INDEX IF NOT EXISTS idx_statements_registered ON statements(registered_at);

-- Checkpoints (tree state snapshots)
CREATE TABLE IF NOT EXISTS checkpoints (
  tree_size INTEGER PRIMARY KEY,       -- Tree size at checkpoint
  root_hash BLOB NOT NULL,             -- Merkle root (32 bytes)
  cose_sign1 BLOB NOT NULL,            -- Signed checkpoint
  created_at INTEGER NOT NULL          -- Unix timestamp
);

CREATE INDEX IF NOT EXISTS idx_checkpoints_created ON checkpoints(created_at DESC);

-- Issuers (for /issuers/{issuerId})
CREATE TABLE IF NOT EXISTS issuers (
  issuer_id TEXT PRIMARY KEY,          -- Issuer identifier
  metadata BLOB NOT NULL,              -- CBOR-encoded metadata
  created_at INTEGER NOT NULL,         -- Unix timestamp
  updated_at INTEGER                   -- Unix timestamp
);

-- Service keys (for /.well-known/scitt-keys)
CREATE TABLE IF NOT EXISTS service_keys (
  kid TEXT PRIMARY KEY,                -- Key identifier
  cose_key BLOB NOT NULL,              -- CBOR-encoded COSE Key
  status TEXT NOT NULL,                -- "active", "rotated", "revoked"
  created_at INTEGER NOT NULL,
  rotated_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_keys_status ON service_keys(status);
