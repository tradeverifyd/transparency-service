# Test Fixture Generation Tools

## Overview

This directory contains tools for generating test fixtures used by the cross-implementation integration test suite.

## Available Tools

### generate_keypair.go

Generates ES256 keypair fixtures in the format expected by `test-fixtures.schema.json`.

**Usage:**
```bash
go run tools/generate_keypair.go \
  -id keypair_alice \
  -desc "Test keypair for Alice (signer)" \
  -output fixtures/keys
```

**Output:**
- PEM-encoded ECDSA P-256 private key
- JWK-encoded public key (RFC 7517)
- RFC 7638 JWK thumbprint (hex-encoded)

### Existing RFC Test Vector Generators

The repository already contains comprehensive RFC test vector generators:

#### Go tlog Merkle Tree Vectors

**Location:** `go-tlog-generator/`

**Purpose:** Generates RFC 6962 Merkle tree test vectors using Go's canonical `golang.org/x/mod/sumdb/tlog` implementation.

**Usage:**
```bash
cd go-tlog-generator
go run main.go
```

**Output:** `../test-vectors/tlog-size-{2,4,7,8,16}.json`
- Tree leaves (hex-encoded)
- Expected root hashes
- Inclusion proofs for all leaf positions

**Validation:** Used by `go-interop.test.ts` to validate TypeScript implementation against Go canonical behavior.

#### Go COSE Sign1 Vectors

**Location:** `go-cose-generator/`

**Purpose:** Generates RFC 9052 COSE Sign1 test vectors using Go COSE libraries.

**Usage:**
```bash
cd go-cose-generator
go run main.go
```

**Output:** `../cose-vectors/*.json`
- COSE Sign1 structures with various protected headers
- CWT claims combinations
- Signature test vectors

**Validation:** Used by `go-cose-interop.test.ts` to validate TypeScript COSE implementation.

## RFC Compliance

All test vectors are generated from Go implementations to serve as the canonical reference (per Constitution Principle VIII):

- **RFC 6962** (Merkle Trees): `golang.org/x/mod/sumdb/tlog`
- **RFC 9052** (COSE): Go COSE libraries
- **RFC 8392** (CWT): CWT claim encoding
- **RFC 7638** (JWK Thumbprint): SHA-256 of canonical JWK JSON
- **C2SP tlog-tiles**: Tile naming conventions from Go tlog

## Test Fixture Schema Compliance

All generated fixtures follow the schemas defined in:
- `../../specs/003-create-an-integration/contracts/test-fixtures.schema.json`
- `../../specs/003-create-an-integration/contracts/test-report.schema.json`

With required conventions:
- **snake_case** for all JSON field names
- **hex encoding** (lowercase) for all binary identifiers
- **ISO 8601** timestamps

## Regenerating All Fixtures

To regenerate all test fixtures:

```bash
# Generate keypairs
./generate_all_keypairs.sh

# Generate RFC test vectors (using existing generators)
cd go-tlog-generator && go run main.go && cd ..
cd go-cose-generator && go run main.go && cd ..

# Verify fixture validity
go test -v ../lib/fixtures_test.go
```

## Adding New Fixtures

1. Create generation tool in this directory
2. Follow `test-fixtures.schema.json` schema
3. Use snake_case and hex encoding
4. Document usage in this README
5. Add validation test in `../lib/`
