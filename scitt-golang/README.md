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
go build -o scitt

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

### Generate Issuer Keys

Generate cryptographic key pairs for signing transparency statements and receipts. 
The private key must remain confidential for integrity and authenticity of the statements and receipts to be trustworthy.

```bash
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


### Diagnose CBOR Files

Analyze CBOR files with extended diagnostic notation, recognizing COSE Keys and COSE Sign1 structures.
This helps explore keys, transparency statements and receipts.

```bash
# Inspect a transparency service's verification key
./scitt diagnose ./demo/pub.cbor

# Inspect a signed statement
./scitt diagnose demo/feed-2025-10-18-205342/pacific-silicon-foundry/documents/wafer-batch-010.cbor

# Inspect a transparency receipt
./scitt diagnose demo/feed-2025-10-18-205342/pacific-silicon-foundry/documents/wafer-batch-010.receipt.cbor
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
  / kty: EC2 /
  1: 2,
  / kid /
  2: h'5e0ca47c6c859a147b81b0d91896c8d990b6ccf450563466e0b654bd0d3973c9',
  / alg: ES256 /
  3: -7,
  / crv: P-256 /
  -1: 1,
  / x /
  -2: h'720feeb0b1dbaaef4342519a48183a324a361d11b99e33a2f0bdc94f48877ce2',
  / y /
  -3: h'44bcec684e579193eeef50b933069cd0aa3ba7d3fac6cd8b776859e6151e167d',
}
```

## Hex

```
a6 01 02 02 58 20 5e 0c a4 7c 6c 85 9a 14 7b 81 b0 d9 18 96 c8 d9 90 b6 cc f4 50 56 34 66 e0 b6 54 bd 0d 39 73 c9 03 26 20 01 21 58 20 72 0f ee b0 b1 db aa ef 43 42 51 9a 48 18 3a 32 4a 36 1d 11 b9 9e 33 a2 f0 bd c9 4f 48 87 7c e2 22 58 20 44 bc ec 68 4e 57 91 93 ee ef 50 b9 33 06 9c d0 aa 3b a7 d3 fa c6 cd 8b 77 68 59 e6 15 1e 16 7d
```
```

</details>

<details>
<summary>Example output for transparency receipt (COSE Sign1 with Merkle inclusion proof)</summary>

```markdown
# CBOR Diagnostic Report

**File:** `demo/feed-2025-10-18-205342/pacific-silicon-foundry/documents/wafer-batch-010.receipt.cbor`

**Size:** 287 bytes

**Type:** COSE Sign1

---

## Commented EDN

```cbor-diag
/ cose-sign1 / 18([
  / protected   / <<{
    / kid / 4: h'6bd6e4b8...52850e8b',
    / alg / 1: -7,  # ES256
    / vds / 395: 1,  # RFC9162 SHA-256
    / cwt_claims / 15: {
      / iss / 1: "http://127.0.0.1:56177",
    },
  }>>,
  / unprotected / {
    / vdp / 396: {
      / inclusion / -1: [
        <<[
          / size / 24, / leaf / 23,
          / inclusion path /
          h'124476e2...6e37d703',
          h'4ee37da7...20d1e9e3',
          h'413039d0...02628fe0',
          h'08672f10...0234d9e3'
        ]>>
      ],
    },
  },
  / payload     / null,
  / signature   / h'2ed591ff...6595f4b9'
])```

## Hex

```
84 58 44 a4 04 58 20 6b d6 e4 b8 f1 77 41 18 a5 22 b6 8f b4 8f 71 2d 38 41 5c a9 ca b7 16 33 84 89 18 62 52 85 0e 8b 01 26 19 01 8b 01 0f a1 01 76 68 74 74 70 3a 2f 2f 31 32 37 2e 30 2e 30 2e 31 3a 35 36 31 37 37 a1 19 01 8c a1 20 58 8d 83 18 18 17 84 58 20 12 44 76 e2 ce 39 32 ca cd a4 7c 05 95 85 4f 8c 61 f0 f9 98 76 86 80 79 52 a2 5e 4a 6e 37 d7 03 58 20 4e e3 7d a7 fa 03 78 c0 e3 48 5b ae 80 11 cf 87 00 4e 82 ed cb 8c f2 b3 2e 14 ce 01 20 d1 e9 e3 58 20 41 30 39 d0 70 f9 10 38 44 aa 0e a9 c2 e7 34 77 94 36 5e 46 97 68 6b 82 4b b6 6d 5a 02 62 8f e0 58 20 08 67 2f 10 6f 9e 00 d5 f6 cb 99 2c 83 a4 c6 01 cd 2c d0 5d 9d e9 3f 4c 69 90 da fa 02 34 d9 e3 f6 58 40 2e d5 91 ff 81 bf 3f 5e 1c 20 76 e0 2d b9 92 a1 f8 0c f1 32 78 23 5f 85 9c ec 4f c5 53 13 39 16 2e 45 cc 28 0e bf af 17 ad b7 55 c4 4b b1 35 9f 5d 0e f7 89 17 1e ef 16 66 31 4d 6c 65 95 f4 b9
```
```

</details>

### Create Transparency Service

Initialize a new transparency service with cryptographic configuration and storage backends.
This service provides tamper-evident logging for supply chain artifacts, ensuring auditability and non-repudiation of recorded statements.

The service supports two database backends:
- **SQLite** (default): Lightweight, file-based database ideal for single-node deployments
- **MongoDB**: Distributed database for high-availability, multi-node deployments

**Security Note**: Sensitive values (API keys, MongoDB credentials) are stored as environment variable references in the YAML config file (e.g., `${SCITT_API_KEY}`). The actual values should be placed in a `.env` file which is gitignored and never committed.

```bash
# Create a demo service with SQLite (default)
./scitt service create \
  --receipt-issuer http://127.0.0.1:56177 \
  --receipt-signing-key ./demo/priv.cbor \
  --receipt-verification-key ./demo/pub.cbor \
  --tile-storage ./demo/tiles \
  --metadata-storage ./demo/scitt.db \
  --definition ./demo/scitt.yaml

```

<details>
<summary>Example output (SQLite)</summary>

```
✓ Service definition created successfully
  Issuer:       http://127.0.0.1:56177
  Database:     sqlite (./demo/scitt.db)
  Tiles:        ./demo/tiles
  Definition:   ./demo/scitt.yaml

⚠ API Key Required:
  Generate a secure API key and add to .env file:
    SCITT_API_KEY=$(openssl rand -hex 32)

Start the service with:
  ./scitt service start --definition ./demo/scitt.yaml
```

</details>

<details>
<summary>Example output (MongoDB)</summary>

```
✓ Service definition created successfully
  Issuer:       http://127.0.0.1:56177
  Database:     mongodb (configured via environment variables)
  Tiles:        (configured via environment variables)
  Definition:   ./demo/scitt-mongodb.yaml

⚠ API Key Required:
  Generate a secure API key and add to .env file:
    SCITT_API_KEY=$(openssl rand -hex 32)

⚠ Additional Environment Variables Required:
  Add these to your .env file:
    SCITT_MONGODB_URI=<your-mongodb-uri>
    SCITT_MONGODB_DATABASE=<your-database-name>

Start the service with:
  ./scitt service start --definition ./demo/scitt-mongodb.yaml
```

**Note**: The generated YAML file contains environment variable references like `${SCITT_API_KEY}` and `${SCITT_MONGODB_URI}`. These are safe to commit to version control. The actual secret values should be stored in a `.env` file (which is gitignored).

</details>

### Start the Transparency Service

Launch the transparency service to accept and log supply chain statements. 
The running service provides HTTP APIs for statement registration and maintains the cryptographically verifiable audit log.

```bash
# Start server using configuration file
./scitt service start --definition ./demo/scitt.yaml
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
