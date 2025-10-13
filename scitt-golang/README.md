# SCITT Transparency Service - Go Implementation

Go implementation of the IETF SCITT (Supply Chain Integrity, Transparency, and Trust) transparency service, maintaining 100% API parity with the TypeScript implementation.

## Overview

This is part of a dual-language monorepo providing:
- **RFC 9052/9053**: COSE (CBOR Object Signing and Encryption) operations
- **RFC 6962**: Certificate Transparency-style Merkle trees
- **C2SP tlog-tiles**: Efficient tile-based Merkle tree storage
- **IETF SCITT**: Transparency service for supply chain artifacts


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

### Diagnose CBOR Files

Analyze CBOR files with extended diagnostic notation, recognizing COSE Keys and COSE Sign1 structures:

```bash
# Diagnose a CBOR file (output to stdout)
./scitt diagnose demo/pub.cbor

# Save diagnostic report to file
./scitt diagnose demo/pub.cbor --output report.md

# Example output for COSE Key:
# - Structure type detection (COSE Key, COSE Sign1, or generic CBOR)
# - Pretty-printed key parameters (kty, alg, crv, x, y)
# - Extended diagnostic notation in CBOR-diag format
# - Hex dump of raw CBOR bytes
```

### Generate Issuer Keys

```bash
# Generate a new ES256 key pair (COSE format)
./scitt issuer key generate

# This creates:
# - private_key.cbor (EC2 private key with ES256 algorithm)
# - public_key.cbor  (EC2 public key with ES256 algorithm)

# Generate with custom paths
./scitt issuer key generate \
  --private-key ./demo/priv.cbor \
  --public-key ./demo/pub.cbor
```

### Create Transparency Service

```bash
# Create a new transparency service
./scitt service definition create \
  --receipt-issuer https://transparency.example \
  --receipt-signing-key ./demo/priv.cbor \
  --receipt-verification-key ./demo/pub.cbor \
  --tile-storage ./demo/tiles \
  --metadata-storage ./demo/scitt.db \
  --definition ./demo/scitt.yaml

# This creates:
# - ./demo/scitt.yaml (configuration file)
# - ./demo/tiles (tile storage directory)
# - ./demo/scitt.db (SQLite database)
```

### Start the Transparency Service

```bash
# Start server using configuration file
./scitt service start --definition ./demo/scitt.yaml

# Or override config settings
./scitt service start --definition ./demo/scitt.yaml --host 127.0.0.1 --port 9000
```

### Sign Statements

```bash
./scitt statement sign \
  --content ./demo/test.parquet \
  --content-type application/vnd.apache.parquet \
  --content-location https://example.com/test.parquet \
  --issuer "https://example.com" \
  --subject "urn:example:dataset:2025-10-11" \
  --signing-key ./demo/priv.cbor \
  --signed-statement ./demo/statement.cbor
```

### Verify Statements

```bash
./scitt statement verify \
  --artifact ./demo/test.parquet \
  --signed-statement ./demo/statement.cbor \
  --verification-key ./demo/pub.cbor
```

## Contributing

This implementation maintains 100% API parity with the TypeScript implementation in `../scitt-typescript/`. Changes should be coordinated across both implementations.

## License

See repository root LICENSE file.
