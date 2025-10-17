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

Analyze CBOR files with extended diagnostic notation, recognizing COSE Keys and COSE Sign1 structures. 
This helps explore keys, transparency statements and receipts.

```bash
# Inspect a transparency service's verification key
./scitt diagnose ./demo/pub.cbor

# Save diagnostic report for audit purposes
./scitt diagnose ./demo/pub.cbor --output verification-key-report.md

# Diagnose a signed statement before registration
./scitt diagnose ./demo/statement.cbor --output statement-analysis.md
```

<details>
<summary>Example output for COSE EC2 public key</summary>

```markdown
# CBOR Diagnostic Report

**File:** `./demo/pub.cbor`

**Size:** 112 bytes

**Generated:** 2025-10-17T12:28:58Z

**Type:** COSE Key

---

## Commented EDN

```cbor-diag
/ COSE Key /
{
  1: 2, / kty: EC2 /
  2: h'5e0ca47c6c859a147b81b0d91896c8d990b6ccf450563466e0b654bd0d3973c9', / kid /
  3: -7, / alg: ES256 /
  -1: 1, / crv: P-256 /
  -2: h'720feeb0b1dbaaef4342519a48183a324a361d11b99e33a2f0bdc94f48877ce2', / x /
  -3: h'44bcec684e579193eeef50b933069cd0aa3ba7d3fac6cd8b776859e6151e167d', / y /
}
```

## Hex

```
a6 01 02 02 58 20 5e 0c a4 7c 6c 85 9a 14 7b 81 b0 d9 18 96 c8 d9 90 b6 cc f4 50 56 34 66 e0 b6 54 bd 0d 39 73 c9 03 26 20 01 21 58 20 72 0f ee b0 b1 db aa ef 43 42 51 9a 48 18 3a 32 4a 36 1d 11 b9 9e 33 a2 f0 bd c9 4f 48 87 7c e2 22 58 20 44 bc ec 68 4e 57 91 93 ee ef 50 b9 33 06 9c d0 aa 3b a7 d3 fa c6 cd 8b 77 68 59 e6 15 1e 16 7d
```
```

</details>

### Generate Issuer Keys

Generate cryptographic key pairs for signing transparency statements and receipts. 
The private key must remain confidential for integrity and authenticity of the statements and receipts to be trustworthy.

```bash
# Generate ES256 key pair for a production transparency service
./scitt issuer key generate \
  --private-key ./keys/transparency-service-private.cbor \
  --public-key ./keys/transparency-service-public.cbor

# Generate keys in demo directory for testing
./scitt issuer key generate \
  --private-key ./demo/priv.cbor \
  --public-key ./demo/pub.cbor
```

<details>
<summary>Example output</summary>

```
✓ Key pair generated successfully
  Thumbprint:  fe7946c94dc273e63c1511eb36580468b0924693481d9a77e40d9f5e8f226f0c
  Algorithm:   ES256 (ECDSA P-256 with SHA-256)
  Private key: ./demo/priv.cbor (147 bytes)
  Public key:  ./demo/pub.cbor (112 bytes)
```

</details>

### Create Transparency Service

Initialize a new transparency service with cryptographic configuration and storage backends. 
This service provides tamper-evident logging for supply chain artifacts, ensuring auditability and non-repudiation of recorded statements.

```bash
# Create a production transparency service
./scitt service create \
  --receipt-issuer https://transparency.example.com \
  --receipt-signing-key ./keys/transparency-service-private.cbor \
  --receipt-verification-key ./keys/transparency-service-public.cbor \
  --tile-storage /var/lib/scitt/tiles \
  --metadata-storage /var/lib/scitt/scitt.db \
  --definition /etc/scitt/service.yaml

# Create a local demo service for testing
./scitt service create \
  --receipt-issuer http://127.0.0.1:56177 \
  --receipt-signing-key ./demo/priv.cbor \
  --receipt-verification-key ./demo/pub.cbor \
  --tile-storage ./demo/tiles \
  --metadata-storage ./demo/scitt.db \
  --definition ./demo/scitt.yaml
```

<details>
<summary>Example output</summary>

```
✓ Service definition created successfully
  Issuer:       http://127.0.0.1:56177
  API Key:      6f41f04b25e84943c7d9c6158c24d2fe0ffcb5613e1bb238650a770daf7fd98d
  Tiles:        ./demo/tiles
  Metadata:     ./demo/scitt.db
  Definition:   ./demo/scitt.yaml

Start the service with:
  ./scitt service start --definition ./demo/scitt.yaml
```

</details>

### Start the Transparency Service

Launch the transparency service to accept and log supply chain statements. 
The running service provides HTTP APIs for statement registration and maintains the cryptographically verifiable audit log.

```bash
# Start server using configuration file
./scitt service start --definition ./demo/scitt.yaml

# Start with custom host and port override
./scitt service start --definition ./demo/scitt.yaml --host 0.0.0.0 --port 9000

# Start production service
./scitt service start --definition /etc/scitt/service.yaml
```

<details>
<summary>Example output</summary>

```
2025/10/17 07:38:51 SCITT Transparency Service
2025/10/17 07:38:51 Documentation: http://127.0.0.1:56177/
```

</details>

### Sign Statements

Create cryptographically signed statements about supply chain artifacts. 
This binds artifact identity to metadata claims, enabling downstream consumers to verify provenance, integrity and authenticity of training data, models, or other critical assets.

```bash
# Sign a statement for AI training data containing vulnerability information
./scitt statement sign \
  --content ./demo/test.parquet \
  --content-type application/vnd.apache.parquet \
  --content-location https://datasets.security-ai.example.com/cve-training-2024-q4.parquet \
  --issuer "https://security-ai.example.com" \
  --subject "urn:security-ai:training-data:cve-2024-q4" \
  --signing-key ./demo/priv.cbor \
  --signed-statement ./demo/statement.cbor
```

<details>
<summary>Example output</summary>

```
✓ Hash envelope created successfully
  Content:          ./demo/test.parquet (852302 bytes)
  Content Hash:     873f9824c3821978219b126536581c0c6ecedd746115885fa468b0bba4a138fe
  Content Type:     application/vnd.apache.parquet
  Content Location: https://datasets.security-ai.example.com/cve-training-2024-q4.parquet
  Issuer:           https://security-ai.example.com
  Subject:          urn:security-ai:training-data:cve-2024-q4
  Signed Statement: ./demo/statement.cbor (335 bytes)
  Leaf Hash:        5b768587e71491d0bce16ce5427261e226fc8da3aa0ce3b9e3c8311d0f4dc7d1 (stored in the tile log)
```

</details>

### Verify Statements

Cryptographically verify signed statements and confirm artifact integrity. 
This ensures that received artifacts match their claimed identity and haven't been tampered with, protecting against supply chain attacks.

```bash
# Verify a statement and check artifact integrity
./scitt statement verify \
  --artifact ./demo/test.parquet \
  --signed-statement ./demo/statement.cbor \
  --verification-key ./demo/pub.cbor

# Verify without checking artifact (validates signature only)
./scitt statement verify \
  --signed-statement ./demo/statement.cbor \
  --verification-key ./demo/pub.cbor
```

<details>
<summary>Example output</summary>

```
✓ Verification successful
  Signature:        Valid
  Artifact Hash:    873f9824c3821978219b126536581c0c6ecedd746115885fa468b0bba4a138fe (matches)
  Hash Algorithm:   SHA-256 (label -16)
  Content Type:     application/vnd.apache.parquet
  Content Location: https://datasets.security-ai.example.com/cve-training-2024-q4.parquet
  Issuer:           https://security-ai.example.com
  Subject:          urn:security-ai:training-data:cve-2024-q4
  Leaf Hash:        5b768587e71491d0bce16ce5427261e226fc8da3aa0ce3b9e3c8311d0f4dc7d1 (stored in the tile log)
```

</details>

### Register Statements

Submit signed statements to the transparency service for inclusion in the transparency log. 
Registration creates an immutable audit trail, enabling independent verification and detection of unauthorized modifications to supply chain artifacts.

```bash
# Register a statement with the transparency service
./scitt statement register \
  --service http://127.0.0.1:56177 \
  --api-key 6f41f04b25e84943c7d9c6158c24d2fe0ffcb5613e1bb238650a770daf7fd98d \
  --statement ./demo/statement.cbor \
  --receipt ./demo/statement.receipt.cbor
```

<details>
<summary>Example output</summary>

```
✓ Statement registered successfully
  Statement:  ./demo/statement.cbor (335 bytes)
  Leaf Hash:  5b768587e71491d0bce16ce5427261e226fc8da3aa0ce3b9e3c8311d0f4dc7d1
  Receipt:    ./demo/statement.receipt.cbor (149 bytes)
  Service:    http://127.0.0.1:56177
```

</details> 

### Verify Receipts

Verify transparency receipts to prove statement inclusion in the transparency log. 
Receipt verification provides cryptographic proof that an artifact's metadata was recorded in the transparency service, establishing trust in the supply chain provenance claims.

```bash
# Verify a receipt with artifact integrity check
./scitt receipt verify \
  --artifact ./demo/test.parquet \
  --statement ./demo/statement.cbor \
  --receipt ./demo/statement.receipt.cbor

# Verify receipt only (without artifact check)
./scitt receipt verify \
  --statement ./demo/statement.cbor \
  --receipt ./demo/statement.receipt.cbor
```

<details>
<summary>Example output</summary>

```
✓ Receipt verification successful
  Artifact: ./demo/test.parquet
  Statement: ./demo/statement.cbor
  Receipt: ./demo/statement.receipt.cbor
  Issuer: http://127.0.0.1:56177
  Tree size: 1
  Leaf index: 0
```

</details>
## Contributing

This implementation maintains 100% API parity with the TypeScript implementation in `../scitt-typescript/`. Changes should be coordinated across both implementations.

## License

See repository root LICENSE file.
