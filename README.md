# Transparency Service

[![CI - Go](https://github.com/tradeverifyd/transparency-service/actions/workflows/ci-golang.yml/badge.svg)](https://github.com/tradeverifyd/transparency-service/actions/workflows/ci-golang.yml)

[![CI - Interoperability](https://github.com/tradeverifyd/transparency-service/actions/workflows/ci-interop.yml/badge.svg)](https://github.com/tradeverifyd/transparency-service/actions/workflows/ci-interop.yml)

Experimental IETF standards-based transparency service implementing SCITT (Supply Chain Integrity, Transparency, and Trust) using COSE (CBOR Object Signing and Encryption) and RFC 6962 Merkle trees.

Both go and typescript implementations are under development.

The go implementation is the canonical reference for the IETF drafts and RFCs.

The typescript implementation is used to cross test the go implementation.

## Quick Start

### Generate Synthetic Supply Chain Data

The Go CLI includes a feed generator that creates synthetic supply chain datasets for testing and development. The workflow is split into three separate commands for flexibility:

#### 1. Generate Documents

```bash
# Build the CLI
cd scitt-golang
go build -o scitt ./cmd/scitt

# Generate synthetic supply chain feed
./scitt feed generate
```

Output:
```
Synthetic Supply Chain Feed Generator

[1/3] Creating feed directory...
   Created: feed-2025-10-18-194550

[2/3] Generating documents for 3 companies...
   Pacific Silicon Foundry (foundry)
       Wrote 26 documents
   Apex Semiconductor Corp (IDM)
       Wrote 33 documents
   Quantum Chip Design (fabless)
       Wrote 22 documents

[3/3] Feed generated successfully!
   Location: feed-2025-10-18-194550
   Total documents: 81
   Timestamp: 2025-10-18 19:45:50

Next steps:
  1. Sign documents:    scitt feed sign feed-2025-10-18-194550
  2. Register to SCITT: scitt feed register feed-2025-10-18-194550 --service-url http://localhost:8000 --api-key YOUR_API_KEY
```

#### 2. Sign Documents

```bash
# Sign all documents in the feed
./scitt feed sign feed-2025-10-18-194550
```

This generates ES256 key pairs and creates COSE Sign1 statements (.cbor files) for each document.

#### 3. Register to SCITT Service

```bash
# Register to a SCITT transparency service (requires API key)
./scitt feed register feed-2025-10-18-194550 \
  --service-url http://localhost:8000 \
  --api-key YOUR_API_KEY
```

**Compare Multiple Services:** You can register the same feed to multiple transparency services to compare tile generation:

```bash
# Register to first service
./scitt feed register feed-2025-10-18-194550 \
  --service-url http://service1:8000 \
  --api-key SERVICE1_KEY

# Register same feed to second service for comparison
./scitt feed register feed-2025-10-18-194550 \
  --service-url http://service2:9000 \
  --api-key SERVICE2_KEY
```

#### What Gets Created

The generator creates:
- **80-110 JSON documents** across 10 supply chain categories (wafers, minerals, chips, firmware, SBOMs, memory, AI datasets/models, CVEs, logistics)
- **3 semiconductor company identities** (foundry, IDM, fabless)
- **ES256 key pairs** for each company (when signing)
- **COSE Sign1 statements** for each document (when signing)
- **SCITT receipts** for each registered statement (when registering)
- **Timestamped feed directory** with organized output structure

#### Directory Structure

After completing all steps:
```
feed-2025-10-18-194550/
├── metadata.json                          # Feed metadata with timestamp and seed
├── pacific-silicon-foundry/               # Foundry company
│   ├── private_key.cbor                   # ES256 private key
│   ├── public_key.cbor                    # ES256 public key
│   └── documents/
│       ├── wafer-batch-001.json           # Original JSON document
│       ├── wafer-batch-001.cbor           # Signed COSE statement
│       └── wafer-batch-001.receipt.cbor   # SCITT receipt with inclusion proof
├── apex-semiconductor-corp/               # IDM company (chips, firmware, SBOMs)
└── quantum-chip-design/                   # Fabless company (AI, memory, CVEs)
```

