# Data Model: Synthetic Supply Chain Feed Generator

**Feature**: 004-add-a-cli
**Date**: 2025-10-18
**Phase**: Phase 1 (Design & Contracts)

This document defines the data structures, entities, and file system layout for the synthetic supply chain feed generator.

---

## Core Entities

### 1. Feed Directory

**Purpose**: Root container for all generated data for one synthetic supply chain scenario.

**Structure**:
```
feed-{YYYY-MM-DD-HHMMSS}/
├── pacific-silicon-foundry/    # Foundry company
├── apex-semiconductor-corp/     # IDM company
├── quantum-chip-design/         # Fabless company
└── metadata.json                # Feed metadata
```

**Fields** (metadata.json):
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
    },
    {
      "name": "Apex Semiconductor Corp",
      "role": "idm",
      "urn": "urn:supply-chain:apex-semiconductor-corp",
      "directory": "apex-semiconductor-corp"
    },
    {
      "name": "Quantum Chip Design",
      "role": "fabless",
      "urn": "urn:supply-chain:quantum-chip-design",
      "directory": "quantum-chip-design"
    }
  ],
  "document_count": 104,
  "tile_count_expected": 4
}
```

**Naming Rules**:
- Format: `feed-{YYYY-MM-DD-HHMMSS}` using current timestamp
- If collision: append `-{NNN}` suffix (e.g., `feed-2025-10-18-143022-001`)
- Timestamp extracted for deterministic RNG seeding

---

### 2. Company Identity

**Purpose**: Represents a semiconductor supply chain participant with specific role.

**Directory Structure**:
```
{company-slug}/
├── priv.cbor              # ES256 private key (CBOR COSE_Key format)
├── pub.cbor               # ES256 public key (CBOR COSE_Key format)
└── documents/
    ├── wafer-batch-001.json
    ├── wafer-batch-001.cbor              # Signed statement
    ├── wafer-batch-001.receipt.cbor      # Registration receipt
    ├── mineral-source-001.json
    ├── mineral-source-001.cbor
    └── ...
```

**Company Definitions**:

| Name | Role | URN | Document Types | Count Range |
|------|------|-----|----------------|-------------|
| Pacific Silicon Foundry | Foundry | `urn:supply-chain:pacific-silicon-foundry` | Wafer batches, Mineral sourcing, Logistics | 22-32 |
| Apex Semiconductor Corp | IDM | `urn:supply-chain:apex-semiconductor-corp` | Chip specs, Firmware, SBOMs/HBOMs | 30-40 |
| Quantum Chip Design | Fabless | `urn:supply-chain:quantum-chip-design` | Memory, AI datasets/models, CVE/CWE | 18-30 |

**Key Generation**:
- Algorithm: ES256 (ECDSA P-256)
- Format: CBOR COSE_Key (RFC 8152)
- Generated via: `./scitt issuer keygen`
- Permissions: `priv.cbor` = 0600, `pub.cbor` = 0644

---

### 3. Supply Chain Document

**Purpose**: JSON-formatted supply chain artifact representing one piece of the laptop's supply chain story.

**Base Fields** (all document types):
```json
{
  "document_id": "wafer-batch-001",
  "document_type": "wafer_batch",
  "urn": "urn:supply-chain:pacific-silicon-foundry:wafer-batch:WF-2025-1001",
  "issuer": "urn:supply-chain:pacific-silicon-foundry",
  "timestamp": "2025-10-18T14:30:22Z",
  "content_type": "application/json",
  "content_location": "https://pacific-silicon-foundry.example/supply-chain/wafer-batch/WF-2025-1001.json"
}
```

**Document Type Specifications**:

#### Wafer Batch (8-12 documents per feed)
```json
{
  "document_id": "wafer-batch-001",
  "document_type": "wafer_batch",
  "urn": "urn:supply-chain:pacific-silicon-foundry:wafer-batch:WF-2025-1001",
  "lot_number": "WF-2025-1001",
  "wafer_diameter_mm": 300,
  "thickness_um": 775,
  "material": "silicon",
  "crystal_orientation": "100",
  "resistivity_ohm_cm": 10,
  "produced_date": "2025-10-15"
}
```

#### Mineral Sourcing (6-8 documents)
```json
{
  "document_id": "mineral-source-001",
  "document_type": "mineral_sourcing",
  "urn": "urn:supply-chain:pacific-silicon-foundry:mineral:MS-2025-001",
  "mineral_type": "tantalum",
  "origin_country": "Rwanda",
  "mine_name": "Rutongo Mine",
  "certification": "RMI-Compliant",
  "quantity_kg": 150.5,
  "conflict_free": true
}
```

#### Chip Specification (10-14 documents)
```json
{
  "document_id": "chip-spec-001",
  "document_type": "chip_specification",
  "urn": "urn:supply-chain:apex-semiconductor-corp:cpu:APX-9700K",
  "part_number": "APX-9700K",
  "chip_type": "CPU",
  "cores": 8,
  "threads": 16,
  "base_frequency_ghz": 3.6,
  "boost_frequency_ghz": 5.1,
  "tdp_watts": 65,
  "process_node_nm": 7,
  "npu_included": true,
  "npu_tops": 15
}
```

#### Firmware Manifest (8-10 documents)
```json
{
  "document_id": "firmware-001",
  "document_type": "firmware_manifest",
  "urn": "urn:supply-chain:apex-semiconductor-corp:firmware:UEFI-2025.10",
  "firmware_type": "UEFI",
  "version": "2025.10.01",
  "sha256": "a1b2c3d4e5f6...",
  "signing_authority": "Apex Semiconductor Corp",
  "release_date": "2025-10-01"
}
```

#### SBOM/HBOM (12-16 documents, SPDX 2.3 format)
```json
{
  "spdxVersion": "SPDX-2.3",
  "dataLicense": "CC0-1.0",
  "SPDXID": "SPDXRef-DOCUMENT",
  "documentName": "apex-semiconductor-corp-laptop-sbom",
  "documentNamespace": "https://apex-semiconductor-corp.example/sbom/laptop-2025-10-18",
  "creationInfo": {
    "licenseListVersion": "3.21",
    "creators": ["Tool: scitt-feed-generator"],
    "created": "2025-10-18T14:30:22Z"
  },
  "packages": [
    {
      "SPDXID": "SPDXRef-Package-CPU",
      "name": "APX-9700K-CPU",
      "versionInfo": "1.0",
      "supplier": "Organization: Apex Semiconductor Corp"
    }
  ]
}
```

#### Memory Specification (6-8 documents)
```json
{
  "document_id": "memory-001",
  "document_type": "memory_specification",
  "urn": "urn:supply-chain:quantum-chip-design:memory:QCD-DDR5-32GB",
  "part_number": "QCD-DDR5-32GB",
  "memory_type": "DDR5",
  "capacity_gb": 32,
  "speed_mhz": 5600,
  "cas_latency": 40,
  "voltage": 1.1
}
```

#### AI Training Dataset (4-6 documents)
```json
{
  "document_id": "ai-dataset-001",
  "document_type": "ai_training_dataset",
  "urn": "urn:supply-chain:quantum-chip-design:ai-dataset:ImageNet-2024",
  "dataset_name": "ImageNet-2024-Subset",
  "source": "ImageNet Consortium",
  "license": "CC-BY-4.0",
  "size_gb": 150,
  "data_provenance": "Academic research dataset",
  "usage": "Pre-training laptop AI models"
}
```

#### AI Model Specification (3-5 documents)
```json
{
  "document_id": "ai-model-001",
  "document_type": "ai_model_specification",
  "urn": "urn:supply-chain:quantum-chip-design:ai-model:Vision-Transformer-v2",
  "model_name": "Vision-Transformer-v2",
  "architecture": "Transformer",
  "parameters_millions": 350,
  "quantization": "INT8",
  "inference_latency_ms": 25,
  "target_hardware": "NPU"
}
```

#### CVE/CWE Document (5-7 documents)
```json
{
  "document_id": "cve-001",
  "document_type": "vulnerability_disclosure",
  "urn": "urn:supply-chain:quantum-chip-design:cve:CVE-2024-0519",
  "cve_id": "CVE-2024-0519",
  "cwe_id": "CWE-787",
  "title": "NVIDIA GPU Driver Privilege Escalation",
  "cvss_score": 8.8,
  "severity": "HIGH",
  "affected_component": "NVIDIA GPU Driver",
  "affected_versions": ["535.x", "545.x"],
  "patched_version": "550.54.14",
  "disclosure_date": "2024-06-12"
}
```

#### Logistics Tracking (8-12 documents)
```json
{
  "document_id": "logistics-001",
  "document_type": "logistics_tracking",
  "urn": "urn:supply-chain:pacific-silicon-foundry:shipment:SHIP-2025-001",
  "shipment_id": "SHIP-2025-001",
  "origin": "Taiwan Fab",
  "destination": "Assembly Plant - Vietnam",
  "departure_date": "2025-10-10",
  "arrival_date": "2025-10-12",
  "contents": "Wafer batch WF-2025-1001"
}
```

**File Naming**:
- Format: `{type}-{NNN}.json` (e.g., `wafer-batch-001.json`)
- Signed statement: `{name}.cbor`
- Receipt: `{name}.receipt.cbor`

---

### 4. COSE Sign1 Statement

**Purpose**: Cryptographically signed hash envelope for a supply chain document.

**Structure** (COSE Sign1, RFC 8152):
- **Protected Headers** (label 15):
  - CWT Claims: `iss` (issuer URN), `sub` (subject URN)
- **Unprotected Headers**:
  - Label 258: Hash algorithm (SHA-256)
  - Label 259: Content type
  - Label 260: Content location URL
- **Payload**: SHA-256 hash of document (not document itself)
- **Signature**: ES256 signature (non-deterministic due to ECDSA random nonce)

**Generated via**: `./scitt statement sign` command

**Example Headers**:
```
Protected:
{
  15: {  // CWT Claims
    1: "urn:supply-chain:pacific-silicon-foundry",  // iss
    2: "urn:supply-chain:...:wafer-batch:WF-2025-1001"  // sub
  }
}

Unprotected:
{
  258: -16,  // SHA-256
  259: "application/json",
  260: "https://pacific-silicon-foundry.example/..."
}
```

---

### 5. Registration Receipt

**Purpose**: Proof of statement inclusion in transparency log with Merkle proof.

**Structure** (COSE Sign1):
- Contains: Entry ID, tree size, Merkle inclusion proof
- Generated by: SCITT transparency service
- Stored as: `{document-name}.receipt.cbor`

---

## Supply Chain Graph

**Purpose**: Documents reference each other to form a coherent supply chain narrative.

**Relationship Types**:

```
Laptop SBOM
├─ references → CPU (APX-9700K)
│  ├─ manufactured_from → Wafer Batch (WF-2025-1001)
│  │  └─ sourced_from → Mineral (Tantalum, Rwanda)
│  └─ has_firmware → UEFI Firmware v2025.10
│     └─ vulnerable_to → CVE-2024-0519
├─ references → Memory (QCD-DDR5-32GB)
│  └─ trained_with → AI Dataset (ImageNet-2024)
│     └─ powers_model → Vision Transformer v2
│        └─ vulnerable_to → CVE-2024-3660
└─ shipped_via → Logistics (SHIP-2025-001)
```

**Implementation**:
- Documents include `related_urns` array field
- Example: Laptop SBOM includes `["urn:supply-chain:...:cpu:APX-9700K", "urn:supply-chain:...:memory:QCD-DDR5-32GB"]`
- Minimum 15 cross-references per feed (per SC-006)

---

## File System Layout

**Complete Example**:
```
feed-2025-10-18-143022/
├── metadata.json
├── pacific-silicon-foundry/
│   ├── priv.cbor (ES256 private key, 0600)
│   ├── pub.cbor (ES256 public key, 0644)
│   └── documents/
│       ├── wafer-batch-001.json
│       ├── wafer-batch-001.cbor
│       ├── wafer-batch-001.receipt.cbor
│       ├── wafer-batch-002.json
│       ├── ...
│       ├── mineral-source-001.json
│       ├── ...
│       └── logistics-001.json
├── apex-semiconductor-corp/
│   ├── priv.cbor
│   ├── pub.cbor
│   └── documents/
│       ├── chip-spec-001.json
│       ├── ...
│       ├── firmware-001.json
│       ├── ...
│       └── sbom-001.json (SPDX format)
└── quantum-chip-design/
    ├── priv.cbor
    ├── pub.cbor
    └── documents/
        ├── memory-001.json
        ├── ...
        ├── ai-dataset-001.json
        ├── ai-model-001.json
        ├── ...
        └── cve-001.json
```

**Permissions**:
- Directories: 0755
- JSON documents: 0644
- Private keys: 0600
- Public keys, CBOR files: 0644

**Total Size Estimate**: ~5-10MB per feed directory

---

## URN Format Specification

**Pattern**: `urn:supply-chain:{company-slug}:{type}:{id}`

**Examples**:
- `urn:supply-chain:pacific-silicon-foundry:wafer-batch:WF-2025-1001`
- `urn:supply-chain:apex-semiconductor-corp:cpu:APX-9700K`
- `urn:supply-chain:quantum-chip-design:memory:QCD-DDR5-32GB`
- `urn:supply-chain:quantum-chip-design:cve:CVE-2024-0519`

**Company Slugs**:
- Pacific Silicon Foundry → `pacific-silicon-foundry`
- Apex Semiconductor Corp → `apex-semiconductor-corp`
- Quantum Chip Design → `quantum-chip-design`

**Type Values**:
- `wafer-batch`, `mineral`, `cpu`, `firmware`, `sbom`, `hbom`
- `memory`, `ai-dataset`, `ai-model`, `cve`, `shipment`

**ID Format**:
- Alphanumeric with hyphens
- Examples: `WF-2025-1001`, `APX-9700K`, `CVE-2024-0519`

---

## Document Distribution Summary

| Company | Role | Document Types | Total Count |
|---------|------|----------------|-------------|
| Pacific Silicon Foundry | Foundry | Wafer (8-12), Mineral (6-8), Logistics (8-12) | 22-32 |
| Apex Semiconductor Corp | IDM | Chip (10-14), Firmware (8-10), SBOM/HBOM (12-16) | 30-40 |
| Quantum Chip Design | Fabless | Memory (6-8), AI Dataset (4-6), AI Model (3-5), CVE (5-7) | 18-30 |
| **TOTAL** | | **10 categories** | **70-102** |

**Target**: 90-110 documents to exceed 96 minimum (3 complete tiles × 32 entries)

---

## State Transitions

**Feed Generation Lifecycle**:

```
1. INITIALIZED
   ↓ (create feed directory)
2. KEYS_GENERATED
   ↓ (generate all documents)
3. DOCUMENTS_GENERATED
   ↓ (sign all documents)
4. STATEMENTS_SIGNED
   ↓ (register all statements)
5. STATEMENTS_REGISTERED
   ↓ (verify tiles created)
6. COMPLETE
```

**No intermediate state persistence** - workflow must complete or be restarted (per clarification: no resume capability).

---

## Validation Rules

**Feed Directory**:
- ✅ Must contain exactly 3 company subdirectories
- ✅ Must contain metadata.json with valid timestamp
- ✅ Directory name must match metadata timestamp

**Company Directory**:
- ✅ Must contain priv.cbor and pub.cbor (ES256 keys)
- ✅ Must contain documents/ subdirectory
- ✅ Document count must be within role-specific range

**Documents**:
- ✅ All documents must have valid URN format
- ✅ Issuer URN must match company
- ✅ Subject URN must be unique within feed
- ✅ Content location URL must use HTTPS
- ✅ SPDX documents must validate against SPDX 2.3 schema

**Supply Chain Graph**:
- ✅ Minimum 15 cross-references (SC-006)
- ✅ No dangling references (all URNs must exist)
- ✅ Laptop SBOM must reference all major components

---

## Next Steps

This data model serves as the foundation for:
1. **Implementation tasks** - Developers know exactly what to generate
2. **Test scenarios** - QA can validate structure and content
3. **API contracts** - N/A (CLI-only feature)

**Related Artifacts**:
- [spec.md](./spec.md) - Functional requirements
- [plan.md](./plan.md) - Implementation approach
- [research.md](./research.md) - Technical decisions
- [quickstart.md](./quickstart.md) - User guide (next)
