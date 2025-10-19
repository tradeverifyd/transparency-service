# Transparency Service

[![CI - Go](https://github.com/tradeverifyd/transparency-service/actions/workflows/ci-golang.yml/badge.svg)](https://github.com/tradeverifyd/transparency-service/actions/workflows/ci-golang.yml)

[![CI - Interoperability](https://github.com/tradeverifyd/transparency-service/actions/workflows/ci-interop.yml/badge.svg)](https://github.com/tradeverifyd/transparency-service/actions/workflows/ci-interop.yml)

Experimental IETF standards-based transparency service implementing SCITT (Supply Chain Integrity, Transparency, and Trust) using COSE (CBOR Object Signing and Encryption) and RFC 6962 Merkle trees.

Both go and typescript implementations are under development.

The go implementation is the canonical reference for the IETF drafts and RFCs.

The typescript implementation is used to cross test the go implementation.

## Quick Start

### Generate Synthetic Supply Chain Data

The Go CLI includes a feed generator that creates synthetic supply chain datasets for testing and development:

```bash
# Build the CLI
cd scitt-golang
go build -o scitt ./cmd/scitt

# Generate feed with interactive prompts for signing and registration
./scitt feed generate

# Generate and sign documents, but skip registration
./scitt feed generate --no-register

# Generate documents only (no signing or registration)
./scitt feed generate --no-sign --no-register

# Generate, sign, and register to custom service
./scitt feed generate --service-url http://localhost:3000
```

The generator creates:
- **80-110 JSON documents** across 10 supply chain categories (wafers, minerals, chips, firmware, SBOMs, memory, AI datasets/models, CVEs, logistics)
- **3 semiconductor company identities** (foundry, IDM, fabless)
- **ES256 key pairs** for each company (when signing is enabled)
- **Interactive workflows** for signing documents as COSE statements and registering them to a SCITT service
- **Timestamped feed directory** with organized output structure

Example output structure (with signing and registration):
```
feed-2025-10-18-143022/
├── metadata.json
├── pacific-silicon-foundry/
│   ├── private_key.cbor                   # Generated when signing
│   ├── public_key.cbor                    # Generated when signing
│   └── documents/
│       ├── wafer-batch-001.json
│       ├── wafer-batch-001.cbor           # Generated when signing
│       └── wafer-batch-001.receipt.cbor   # Generated when registering
├── apex-semiconductor-corp/
└── quantum-chip-design/
```

