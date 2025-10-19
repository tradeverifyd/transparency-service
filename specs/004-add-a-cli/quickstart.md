# Quickstart: Synthetic Supply Chain Feed Generator

**Feature**: 004-add-a-cli
**Audience**: Developers, QA engineers, researchers testing SCITT implementations
**Prerequisites**: Go 1.24+, running SCITT transparency service

This guide walks you through generating, signing, and registering a complete synthetic supply chain feed for an AI-capable laptop.

---

## Installation

### Build from Source

```bash
cd /path/to/transparency-service/scitt-golang
go build -o scitt ./cmd/scitt
```

### Verify Installation

```bash
./scitt --version
./scitt feed --help
```

---

## Quick Start (3 Steps)

### Step 1: Start SCITT Service

```bash
# Start local transparency service
./scitt service start \
  --definition ./demo/scitt-local.yaml \
  --verbose
```

Service runs at `http://127.0.0.1:56177` by default.

### Step 2: Set API Key

```bash
export SCITT_API_KEY="your-api-key-here"
```

Get API key from your SCITT service configuration.

### Step 3: Generate Feed

```bash
./scitt feed generate
```

This single command will:
1. ✅ Generate feed directory with timestamp
2. ✅ Create 3 company identities with ES256 keys
3. ✅ Generate 90-110 supply chain documents
4. ✅ Prompt to sign documents (interactive)
5. ✅ Prompt to register statements (interactive)

**Example Output**:
```
Generating synthetic supply chain feed...
Feed directory: feed-2025-10-18-143022/

Creating companies...
  ✓ Pacific Silicon Foundry (foundry)
  ✓ Apex Semiconductor Corp (IDM)
  ✓ Quantum Chip Design (fabless)

Generating documents...
[████████████████████████████████] 104/104 (100%)

Feed generated successfully!
  Companies: 3
  Documents: 104
  Feed directory: feed-2025-10-18-143022/

Ready to sign documents? (yes/no):
```

---

## Interactive Workflow

### Signing Phase

After generation completes, you'll be prompted:

```
Ready to sign documents? (yes/no): yes

Signing documents for Pacific Silicon Foundry (1/3)...
[████████████████████████████████] 28/28 (100%)
✓ Signed 28 documents

Signing documents for Apex Semiconductor Corp (2/3)...
[████████████████████████████████] 36/36 (100%)
✓ Signed 36 documents

Signing documents for Quantum Chip Design (3/3)...
[████████████████████████████████] 40/40 (100%)
✓ Signed 40 documents

All documents signed successfully. Total statements: 104
```

### Registration Phase

After signing completes, you'll be prompted:

```
Ready to register statements? (yes/no): yes

SCITT service URL [default: http://127.0.0.1:56177]:

Connected to SCITT service: https://transparency.example

Registering 104 statements...
[████████████████████████████████] 104/104 (100%)

Registration complete!
  Total statements registered: 104
  Tree size: 104
  Tiles created: 4 (entries 0-31, 32-63, 64-95, 96-103)
  Feed directory: feed-2025-10-18-143022/
```

---

## Manual Workflow (Advanced)

For finer control, you can run each phase separately:

### 1. Generate Only

```bash
# Generate documents and keys only (no signing/registration)
./scitt feed generate --no-sign --no-register
```

### 2. Sign Manually

```bash
cd feed-2025-10-18-143022/pacific-silicon-foundry/documents

# Sign a single document
../../scitt statement sign \
  --content wafer-batch-001.json \
  --content-type application/json \
  --content-location https://pacific-silicon-foundry.example/supply-chain/wafer-batch/WF-2025-1001.json \
  --issuer "urn:supply-chain:pacific-silicon-foundry" \
  --subject "urn:supply-chain:pacific-silicon-foundry:wafer-batch:WF-2025-1001" \
  --signing-key ../priv.cbor \
  --signed-statement wafer-batch-001.cbor
```

### 3. Register Manually

```bash
# Register a single statement
./scitt statement register \
  --service http://127.0.0.1:56177 \
  --api-key $SCITT_API_KEY \
  --statement feed-2025-10-18-143022/pacific-silicon-foundry/documents/wafer-batch-001.cbor \
  --receipt feed-2025-10-18-143022/pacific-silicon-foundry/documents/wafer-batch-001.receipt.cbor
```

---

## Exploring the Generated Feed

### Feed Structure

```bash
tree feed-2025-10-18-143022/ -L 3

feed-2025-10-18-143022/
├── metadata.json
├── pacific-silicon-foundry/
│   ├── priv.cbor
│   ├── pub.cbor
│   └── documents/
│       ├── wafer-batch-001.json
│       ├── wafer-batch-001.cbor
│       ├── wafer-batch-001.receipt.cbor
│       └── ...
├── apex-semiconductor-corp/
│   └── ...
└── quantum-chip-design/
    └── ...
```

### Inspecting Documents

**View JSON document**:
```bash
cat feed-2025-10-18-143022/pacific-silicon-foundry/documents/wafer-batch-001.json | jq
```

**Diagnose COSE statement**:
```bash
./scitt diagnose \
  --input feed-2025-10-18-143022/pacific-silicon-foundry/documents/wafer-batch-001.cbor
```

**Verify signature**:
```bash
./scitt statement verify \
  --artifact feed-2025-10-18-143022/pacific-silicon-foundry/documents/wafer-batch-001.json \
  --signed-statement feed-2025-10-18-143022/pacific-silicon-foundry/documents/wafer-batch-001.cbor \
  --verification-key feed-2025-10-18-143022/pacific-silicon-foundry/pub.cbor
```

### Feed Metadata

```bash
cat feed-2025-10-18-143022/metadata.json | jq
```

```json
{
  "timestamp": "2025-10-18T14:30:22Z",
  "seed": 1729262422000000000,
  "companies": [
    {
      "name": "Pacific Silicon Foundry",
      "role": "foundry",
      "urn": "urn:supply-chain:pacific-silicon-foundry",
      "directory": "pacific-silicon-foundry"
    }
  ],
  "document_count": 104,
  "tile_count_expected": 4
}
```

---

## Use Cases

### 1. Testing SCITT Service Performance

```bash
# Generate large feed
./scitt feed generate

# Measure registration time
time ./scitt feed generate  # Full workflow with timing
```

**Expected Performance** (per spec):
- Generation: <10 seconds for 100+ documents
- Full workflow: <2 minutes total

### 2. Developing Supply Chain Analysis Tools

```bash
# Generate feed
./scitt feed generate

# Analyze supply chain graph
python analyze_supply_chain.py feed-2025-10-18-143022/
```

Use generated documents with realistic URNs, content types, and cross-references to test:
- Knowledge graph construction
- Vulnerability tracking
- SBOM parsing
- Supply chain visualization

### 3. Demo Scenarios

```bash
# Generate feed
./scitt feed generate

# Show transparency log tiles
ls -lh /path/to/tile/storage/

# Query specific statements
./scitt statement hash --input feed-2025-10-18-143022/apex-semiconductor-corp/documents/chip-spec-001.cbor
```

### 4. Interoperability Testing

```bash
# Generate feed with Go implementation
./scitt-golang/scitt feed generate

# Verify with TypeScript implementation
./scitt-typescript statement verify \
  --artifact feed-2025-10-18-143022/pacific-silicon-foundry/documents/wafer-batch-001.json \
  --signed-statement feed-2025-10-18-143022/pacific-silicon-foundry/documents/wafer-batch-001.cbor \
  --verification-key feed-2025-10-18-143022/pacific-silicon-foundry/pub.cbor
```

---

## Troubleshooting

### Error: "SCITT service not available"

**Symptom**:
```
✗ Cannot connect to SCITT service at http://127.0.0.1:56177
```

**Solution**:
```bash
# Check service is running
curl http://127.0.0.1:56177/.well-known/scitt-configuration

# Start service if not running
./scitt service start --definition ./demo/scitt-local.yaml
```

### Error: "API key invalid"

**Symptom**:
```
✗ Registration failed: Unauthorized (401)
  The API key is invalid or missing
```

**Solution**:
```bash
# Set API key environment variable
export SCITT_API_KEY="your-api-key-from-config"

# Or provide interactively when prompted
```

### Error: "Insufficient disk space"

**Symptom**:
```
✗ Failed to create feed directory
  Insufficient disk space. Need ~5MB, have ~2MB
```

**Solution**:
```bash
# Free up disk space
df -h  # Check available space

# Or specify different output directory (future enhancement)
```

### Warning: "Feed directory already exists"

**Symptom**:
```
⚠ Feed directory feed-2025-10-18-143022/ already exists
  Creating feed-2025-10-18-143022-001/ instead
```

**Behavior**: Normal - system automatically appends suffix to avoid collisions.

### Documents Look Identical

**Issue**: Generated documents have same structure/content across runs.

**Expected Behavior**: This is correct! Document generation is deterministic (seeded by timestamp). Same timestamp = same documents. ES256 signatures will differ due to random nonces.

---

## Advanced Configuration

### Custom Service URL

```bash
# During registration, provide custom URL when prompted:
SCITT service URL [default: http://127.0.0.1:56177]: https://prod-scitt.example.com
```

### Skipping Phases

```bash
# Generate and sign, but skip registration
./scitt feed generate --no-register

# Generate only, no signing or registration
./scitt feed generate --no-sign --no-register
```

*(Note: `--no-sign` and `--no-register` flags are planned features - currently prompts are interactive)*

---

## Understanding the Output

### Document Types by Company

**Pacific Silicon Foundry (Foundry)**:
- Wafer batches (8-12 docs)
- Mineral sourcing (6-8 docs)
- Logistics tracking (8-12 docs)

**Apex Semiconductor Corp (IDM)**:
- Chip specifications (10-14 docs)
- Firmware manifests (8-10 docs)
- SBOMs/HBOMs (12-16 docs, SPDX format)

**Quantum Chip Design (Fabless)**:
- Memory specifications (6-8 docs)
- AI training datasets (4-6 docs)
- AI model specifications (3-5 docs)
- CVE/CWE vulnerability disclosures (5-7 docs)

### Supply Chain Story

The generated feed tells a coherent story:

1. **Raw Materials** → Mineral sourcing documents (conflict-free minerals)
2. **Manufacturing** → Wafer batches produced from minerals
3. **Components** → Chips fabricated from wafers
4. **Integration** → Firmware, memory, AI models combined
5. **Product** → Laptop SBOM references all components
6. **Logistics** → Shipment tracking from fab to assembly
7. **Security** → CVE disclosures for firmware and AI frameworks

All documents are linked via URN references to form a complete supply chain graph.

---

## Next Steps

- **Analyze Feed**: Use generated data with supply chain analysis tools
- **Performance Testing**: Measure SCITT service under load (100+ registrations)
- **Visualization**: Build knowledge graphs from document relationships
- **Compliance**: Validate SPDX SBOMs against regulatory requirements

---

## Related Documentation

- [spec.md](./spec.md) - Feature requirements and acceptance criteria
- [plan.md](./plan.md) - Implementation approach and architecture
- [data-model.md](./data-model.md) - Detailed entity and file structure
- [research.md](./research.md) - Technical decisions and rationale

---

## Feedback & Issues

Found a bug or have a feature request?
- File an issue: [GitHub Issues](https://github.com/tradeverifyd/transparency-service/issues)
- Include: `./scitt feed generate` output, error messages, environment details

---

**Version**: 1.0 (Feature 004-add-a-cli)
**Last Updated**: 2025-10-18
