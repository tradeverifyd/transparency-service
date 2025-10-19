# Feature Specification: Synthetic Supply Chain Feed Generator

**Feature Branch**: `004-add-a-cli`
**Created**: 2025-10-18
**Status**: Draft
**Input**: User description: "Add a cli command call \"./scitt feed generate\" that produces 3 issuer identities for popular supply chain company types in the semi conductor industry: \"foundries, integrated device manufacturers, and fabless companies\"... name these synthetic identities according to their role, each of these companies will contribute supply chain document to the transparency log regarding a hypothetical laptop that contains chips, firmware, memory and cpus from all of these companies. This scitt command helps generate synthetic data that goes into the log that helps us tell a story about this hypothetical laptop and a knowledge graph of cyber physical supply chain statements about its components, its logistics journey, its software and hardware bill of materials, the proof of origin for raw minerals that went into the chips, and vulnerability disclosure reports and CVEs that are related to the firmware, operating system and applications that are installed on it. Each time this scitt command is run, a new directory structure will be created to store randomly generated synthetic supply chain documentation regarding this product, after all the raw supply chain data in generated, the user is prompted to run the scitt cli with each of the issuer identities to sign documentation their organization produced, and then the scitt cli will be used to register all these signed statements. It is important that enough data be generated and registered so that at least 3 tiles can be fully completed and stored in tile storage. It is also important that the issuer and subject values are easy to understand, and that content type and locations appear real, so that analysys of this statement feed can mirror analysis of a similar real world feed."

## Clarifications

### Session 2025-10-18

- Q: How should documents be distributed across the 10+ categories (wafer, mineral, chip, SBOM, CVE, AI, etc.)? → A: Predefined distribution with minor randomness (e.g., 8-12 wafers, 5-7 CVEs, 3-5 AI models, totaling 96-130)
- Q: What level of logging/observability should the system provide during operations? → A: Progress bars only (no persistent logs)
- Q: Can users resume interrupted workflows or retry failed operations selectively? → A: Manual restart only (no resume, user must re-run entire command)
- Q: How should FR-025 determinism work given document generation uses randomness? → A: Documents/payloads/metadata/headers are deterministic (seeded from timestamp); ES256 signatures are non-deterministic (include random nonces)
- Q: Should document categories be distributed equally across all 3 companies, or allocated by company role? → A: Distribute per company role (foundry: wafer/mineral/logistics; IDM: chip/firmware/SBOM; fabless: memory/AI/CVE)

## User Scenarios & Testing

### User Story 1 - Generate Synthetic Supply Chain Dataset (Priority: P1)

A supply chain transparency researcher or developer needs to generate realistic test data representing a complete semiconductor supply chain for a laptop product. They run a single command that creates a comprehensive dataset including company identities, cryptographic keys, and supply chain documents across the entire product lifecycle - from raw materials to vulnerability disclosures.

**Why this priority**: This is the foundation of the entire feature. Without the ability to generate a complete, realistic dataset, all downstream testing and analysis scenarios are impossible. This delivers immediate value for testing, demos, and development of supply chain analysis tools.

**Independent Test**: Can be fully tested by running `./scitt feed generate` and verifying that a timestamped directory is created containing: (1) 3 company subdirectories with COSE keys, (2) JSON supply chain documents covering all required categories, (3) realistic URN-based identifiers, and (4) content matching semiconductor industry patterns. Delivers value by enabling immediate testing of SCITT implementations without real supply chain data.

**Acceptance Scenarios**:

1. **Given** a SCITT CLI installation, **When** user runs `./scitt feed generate`, **Then** system creates a new timestamped directory (e.g., `feed-2025-10-18-143022/`) containing three company subdirectories: `pacific-silicon-foundry/`, `apex-semiconductor-corp/`, and `quantum-chip-design/`

2. **Given** feed generation has started, **When** system creates company directories, **Then** each directory contains: a private signing key (`priv.cbor`), a public verification key (`pub.cbor`), and a `documents/` subdirectory with JSON supply chain documents

3. **Given** document generation for the foundry, **When** system creates documents, **Then** the foundry directory contains: wafer manufacturing reports, mineral sourcing certificates, and logistics tracking documents using URNs like `urn:supply-chain:pacific-silicon-foundry:wafer-batch:WF-2025-1001`

4. **Given** document generation for the IDM (Integrated Device Manufacturer), **When** system creates documents, **Then** the IDM directory contains: chip specifications, firmware manifests, software bills of materials (SBOMs), and hardware bills of materials (HBOMs) using URNs like `urn:supply-chain:apex-semiconductor-corp:cpu:APX-9700K`

5. **Given** document generation for the fabless company, **When** system creates documents, **Then** the fabless directory contains: memory module specifications, firmware updates, vulnerability disclosures (CVEs), and component integration reports using URNs like `urn:supply-chain:quantum-chip-design:memory:QCD-DDR5-32GB`

6. **Given** all documents are generated, **When** user inspects the dataset, **Then** COSE statements include `parent_leaf_hash` CWT claims (string key in label 15) creating a supply chain graph (e.g., chip statement → wafer batch leaf hash, laptop SBOM → component leaf hashes, CVE → firmware leaf hash). Multiple statements can share the same subject URN for different perspectives on the same entity

7. **Given** a complete dataset, **When** counting all generated documents across all three companies, **Then** total document count is sufficient to fill at least 3 complete tiles in the transparency log (minimum 96 documents based on 32 entries per tile)

8. **Given** generated documents, **When** user examines content types and locations, **Then** all documents use realistic MIME types (e.g., `application/json`, `application/spdx+json`) and plausible HTTPS URLs (e.g., `https://pacific-silicon-foundry.example/supply-chain/wafer-batch/WF-2025-1001.json`)

---

### User Story 2 - Sign Generated Documents (Priority: P2)

After generating the dataset, a user needs to cryptographically sign all documents using each company's private key to create COSE Sign1 structures suitable for registration in the transparency log. The system guides them through signing all documents for each issuer identity sequentially.

**Why this priority**: While document generation (P1) creates the raw dataset, signing is required to make the data usable with SCITT. This is P2 because it builds directly on P1 and is necessary for P3 (registration), but users could potentially sign documents manually if needed.

**Independent Test**: Can be tested independently by using a pre-generated feed directory (from P1) and running the signing workflow. Success is verified when all JSON documents have corresponding `.cbor` signed statement files. Delivers value by automating the tedious process of signing dozens of documents with multiple issuer identities.

**Acceptance Scenarios**:

1. **Given** a feed directory with generated documents, **When** user completes generation, **Then** system displays prompt: "Feed generated successfully. Ready to sign documents? (yes/no)" and waits for user confirmation

2. **Given** user confirms signing, **When** system begins signing workflow, **Then** system processes each company sequentially, displaying: "Signing documents for Pacific Silicon Foundry (1/3)..." with progress indicators

3. **Given** signing for one company, **When** system processes documents, **Then** for each JSON document in `documents/` directory, system creates a COSE Sign1 statement using the company's private key with:
   - Issuer claim: company's URN (e.g., `urn:supply-chain:pacific-silicon-foundry`)
   - Subject claim: document's URN (from document content)
   - Content location: document's URL (from document metadata)
   - Content type: document's MIME type
   - Output file: `[document-name].cbor` in same directory

4. **Given** signing completes for one company, **When** system finishes, **Then** system displays summary: "Signed 42 documents for Pacific Silicon Foundry" and proceeds to next company

5. **Given** signing completes for all companies, **When** workflow finishes, **Then** system displays: "All documents signed successfully. Total statements: 126" and prompts for registration

6. **Given** signing encounters an error, **When** a document fails to sign, **Then** system logs the error, continues with remaining documents, and reports failures at the end: "Signed 125/126 documents. 1 failure: [document-name] - [error message]"

---

### User Story 3 - Register Statements to Transparency Log (Priority: P3)

After signing all documents, a user needs to register all COSE Sign1 statements to a running SCITT transparency service, creating an auditable log of the entire supply chain. The system automatically submits all statements and tracks registration progress.

**Why this priority**: This is P3 because it depends on both P1 (generation) and P2 (signing), and requires a running SCITT service. While critical for the complete workflow, users could manually register statements if the automation fails, making it less critical than the foundational generation and signing steps.

**Independent Test**: Can be tested independently by using a pre-signed feed directory (with `.cbor` files from P2) and a running SCITT service. Success is verified when all statements receive receipts and at least 3 tiles exist in storage. Delivers value by eliminating manual submission of hundreds of statements and providing immediate verification of successful registration.

**Acceptance Scenarios**:

1. **Given** all documents are signed, **When** user confirms registration, **Then** system prompts: "SCITT service URL [default: http://127.0.0.1:56177]:" allowing override of service endpoint

2. **Given** user provides service URL, **When** system validates connection, **Then** system queries `/.well-known/scitt-configuration` endpoint and displays: "Connected to SCITT service: [issuer URL]" or error: "Cannot connect to SCITT service at [URL]"

3. **Given** valid service connection, **When** registration begins, **Then** system displays: "Registering 126 statements..." with real-time progress: "Progress: 23/126 (18%)" and estimated time remaining

4. **Given** registration in progress, **When** each statement is submitted, **Then** system:
   - POSTs COSE Sign1 statement to `/entries` endpoint with API key authentication
   - Receives COSE receipt response
   - Saves receipt to `[document-name].receipt.cbor` alongside statement
   - Updates progress counter

5. **Given** registration completes, **When** all statements are registered, **Then** system displays summary:
   ```
   Registration complete!
   - Total statements registered: 126
   - Tree size: 126
   - Tiles created: 4 (entries 0-31, 32-63, 64-95, 96-126)
   - Feed directory: feed-2025-10-18-143022/
   ```

6. **Given** registration encounters errors, **When** some statements fail, **Then** system continues with remaining statements and displays: "Registered 124/126 statements. 2 failures: [document-name-1], [document-name-2]" with option to retry failed registrations

7. **Given** successful registration, **When** user inspects tile storage, **Then** at least 3 complete tiles exist (tile/entries/000, tile/entries/001, tile/entries/002) containing the registered statement leaf hashes

---

### Edge Cases

- **What happens when feed directory already exists?** System appends timestamp with milliseconds to ensure uniqueness (e.g., `feed-2025-10-18-143022-001/`) and warns user about existing directory
- **How does system handle missing SCITT service during registration?** System validates service connection before starting registration and provides clear error message with service URL. Allows user to retry with different URL or skip registration
- **What happens when API key is missing or invalid?** System checks for `SCITT_API_KEY` environment variable before registration. If missing, prompts user to set it. If invalid (401 response), provides clear error and stops registration (user must re-run with valid key)
- **How does system handle partial signing failures?** System continues signing remaining documents, tracks all failures, and provides detailed report at the end. User must re-run entire workflow (no selective retry)
- **What happens if user interrupts feed generation (Ctrl+C)?** System catches interrupt signal, displays "Feed generation interrupted", and leaves partial directory in place. User can delete directory and re-run
- **How does system ensure tile count requirement (3+ tiles)?** System calculates minimum documents needed (96 = 3 tiles × 32 entries) and generates at least this many across all companies, with ~10-15% buffer to ensure complete tiles (e.g., 30-32 docs per foundry, 34-38 docs per IDM, 22-26 docs per fabless = 96-110 total)
- **What happens when disk space is insufficient?** System fails early with clear error: "Insufficient disk space. Need ~5MB, have ~2MB" before creating directory structure
- **How does system handle concurrent feed generation?** Each feed directory has unique timestamp, allowing multiple concurrent generations without conflicts
- **What happens when keys already exist for a company?** System generates new keys, overwriting existing ones. In the future, could add `--reuse-keys` flag to skip key generation

## Requirements

### Functional Requirements

- **FR-001**: System MUST provide a CLI command `./scitt feed generate` that creates a complete synthetic supply chain dataset
- **FR-002**: System MUST create three company identities representing semiconductor supply chain roles: foundry (Pacific Silicon Foundry), IDM (Apex Semiconductor Corp), and fabless (Quantum Chip Design)
- **FR-003**: System MUST generate COSE key pairs (ES256/P-256, private/public) for each company identity, stored as `priv.cbor` and `pub.cbor` in CBOR format
- **FR-004**: System MUST create a timestamped feed directory (format: `feed-YYYY-MM-DD-HHMMSS/`) containing subdirectories for each company
- **FR-005**: System MUST generate supply chain documents in JSON format covering categories: wafer manufacturing, mineral sourcing, chip specifications, firmware manifests, SBOMs (SPDX format), HBOMs, memory specifications, AI training datasets, AI model specifications, vulnerability disclosures (NVD CVEs and MITRE CWEs), and logistics tracking
- **FR-006**: System MUST generate at least 96 total documents across all companies to ensure at least 3 complete tiles (32 entries per tile) can be filled in the transparency log
- **FR-007**: System MUST use URN-based subject identifiers following pattern: `urn:supply-chain:[company]:[type]:[id]` (e.g., `urn:supply-chain:pacific-silicon-foundry:wafer-batch:WF-2025-1001`)
- **FR-008**: System MUST use URN-based issuer identifiers following pattern: `urn:supply-chain:[company]` for each company
- **FR-009**: System MUST assign realistic content types to documents (e.g., `application/json`, `application/spdx+json` for SBOMs)
- **FR-010**: System MUST assign realistic content locations (HTTPS URLs) following pattern: `https://[company].example/supply-chain/[type]/[id].json`
- **FR-011**: System MUST create document relationships forming a supply chain graph using `parent_leaf_hash` custom CWT claim (string key) in COSE protected headers (label 15). Multiple statements can reference the same subject (sub claim). Supply chain ancestry is tracked via parent leaf hashes (e.g., chip statement → wafer batch leaf hash, laptop SBOM → component leaf hashes, CVE → firmware leaf hash)
- **FR-012**: System MUST generate realistic synthetic data for each document category using predefined distribution ranges with minor randomness (totaling 96-130 documents):
  - Wafer batches (8-12 documents): lot numbers, dimensions, material specs
  - Mineral sourcing (6-8 documents): origin countries, certifications, quantities
  - Chip specifications (10-14 documents): part numbers, frequencies, core counts, TDP, NPU specifications
  - Firmware manifests (8-10 documents): versions, signatures, hash values
  - SBOMs/HBOMs (12-16 documents): component lists, suppliers, versions (SPDX 2.3+ JSON format)
  - Memory specifications (6-8 documents): capacity, speed, timings
  - AI training datasets (4-6 documents): dataset names, sources, licensing, data provenance
  - AI model specifications (3-5 documents): model architectures, training parameters, inference requirements
  - CVE/CWE documents (5-7 documents): real NVD CVE IDs (e.g., CVE-2024-0519 for GPU drivers, CVE-2024-1234 for NPU firmware), MITRE CWE classifications, CVSS scores, affected versions, patches relevant to AI-capable laptop hardware/firmware/software
  - Logistics tracking (8-12 documents): shipment IDs, origins, destinations, timestamps
- **FR-013**: Users MUST be able to trigger document signing workflow after generation via interactive prompt
- **FR-014**: System MUST sign each JSON document using the company's private key (via existing `scitt statement sign` command extended with `--parent-leaf-hash` parameter) to create COSE Sign1 statements with hash envelopes containing:
  - CWT Claims (label 15): issuer claim (iss, label 1), subject claim (sub, label 2), optional `parent_leaf_hash` claim (string key) with 32-byte SHA-256 hash of parent transparency log entry
  - Hash Envelope Headers (labels 258-260): payload hash algorithm (SHA-256), preimage content type, payload location (HTTPS URL)
  - Payload: SHA-256 hash of the document content
  - Note: Multiple statements can share the same subject (sub) URN to represent different aspects or updates of the same supply chain entity
- **FR-015**: System MUST save signed statements as `[document-name].cbor` files alongside the original JSON documents
- **FR-016**: System MUST display real-time signing progress bars for each company with document counts and success/failure summary (no persistent logs)
- **FR-017**: Users MUST be able to trigger statement registration workflow after signing via interactive prompt
- **FR-018**: System MUST connect to a SCITT transparency service via configurable URL (default: `http://127.0.0.1:56177`)
- **FR-019**: System MUST validate SCITT service connectivity by querying `/.well-known/scitt-configuration` before registration
- **FR-020**: System MUST register each COSE Sign1 statement by POSTing to the service's `/entries` endpoint with API key authentication (from `SCITT_API_KEY` environment variable)
- **FR-021**: System MUST save registration receipts as `[document-name].receipt.cbor` files alongside statements
- **FR-022**: System MUST display real-time registration progress bars with counters, percentages, and completion summary including tree size and tile count (no persistent logs)
- **FR-023**: System MUST handle registration errors gracefully, continuing with remaining statements and providing detailed error report
- **FR-024**: System MUST verify that at least 3 tiles are created in tile storage after successful registration
- **FR-025**: Document generation MUST be deterministic using feed directory timestamp as random seed (same timestamp produces identical documents, metadata, headers). Note: ES256 signatures are non-deterministic due to random nonces, so COSE statements and receipts will differ across runs

### Key Entities

- **Feed Directory**: Timestamped root directory containing all generated data for one synthetic supply chain scenario. Contains company subdirectories, metadata, and generation timestamp
- **Company Identity**: Represents a semiconductor supply chain participant with a specific role (foundry, IDM, fabless). Has URN identifier, name, key pair, and generates role-specific documents
- **Supply Chain Document**: JSON-formatted data representing one supply chain artifact (e.g., wafer batch, chip spec, SBOM, CVE). Has URN subject, content type, content location, and references to related documents
- **COSE Key Pair**: ES256 (ECDSA P-256) cryptographic key pair in CBOR format. Private key signs documents, public key verifies signatures. Stored as `priv.cbor` and `pub.cbor`
- **COSE Sign1 Statement**: Cryptographically signed statement containing document hash, issuer/subject claims, content metadata, and optional parent_leaf_hash for supply chain ancestry tracking. Multiple statements can reference the same subject URN. Stored as `.cbor` file. Suitable for SCITT registration
- **Registration Receipt**: COSE_Sign1 receipt from transparency service proving statement inclusion in log. Contains Merkle tree proof, entry ID, and service signature. Stored as `.receipt.cbor` file
- **Supply Chain Graph**: Network of document relationships forming a complete product supply chain. Laptop → Components → Manufacturing → Materials, with cross-references via URNs
- **Tile**: Transparency log storage unit containing 32 statement entries. Feed must generate enough statements to fill 3+ tiles for completeness testing

## Success Criteria

### Measurable Outcomes

- **SC-001**: Users can generate a complete synthetic supply chain dataset in under 10 seconds by running a single CLI command
- **SC-002**: Generated dataset contains exactly 3 company identities with distinct roles (foundry, IDM, fabless) and realistic naming conventions
- **SC-003**: Generated dataset contains at least 96 documents distributed across all companies, sufficient to fill 3+ complete tiles (32 entries per tile)
- **SC-004**: 100% of generated documents use human-readable URN identifiers following the pattern `urn:supply-chain:[company]:[type]:[id]`
- **SC-005**: 100% of generated documents have realistic content types (e.g., `application/json`, `application/spdx+json`) and HTTPS URLs
- **SC-006**: Generated supply chain graph contains at least 15 parent-child relationships via `parent_leaf_hash` CWT claims in COSE headers, creating a connected directed acyclic graph (DAG) of laptop supply chain: materials → wafers → chips → firmware → SBOMs → vulnerabilities → logistics
- **SC-007**: Users can complete the full workflow (generate, sign, register) in under 2 minutes for a dataset of 100+ documents
- **SC-008**: Signing workflow processes all documents for all companies with 100% success rate when keys are valid
- **SC-009**: Registration workflow achieves 100% success rate when SCITT service is available and API key is valid
- **SC-010**: After registration, tile storage contains at least 3 complete tile files (tile/entries/000, tile/entries/001, tile/entries/002) verifiable via file system inspection
- **SC-011**: Generated dataset is immediately usable for supply chain analysis tools without requiring manual data cleanup or reformatting
- **SC-012**: Feed generation creates directories and files with correct permissions allowing all subsequent operations (signing, registration) without permission errors
- **SC-013**: System provides clear real-time progress feedback via progress bars during all operations (generation, signing, registration) with completion percentages and estimated time remaining, without persistent logs
- **SC-014**: System provides actionable error messages for all failure modes (missing service, invalid API key, insufficient disk space) with suggested remediation steps

## Out of Scope

- Real supply chain data integration or connectors to actual supply chain systems
- GUI or web interface for feed generation (CLI only)
- Custom company identity configuration beyond the three predefined roles
- Product types other than laptop (e.g., smartphone, server, IoT device)
- Industries other than semiconductor supply chain
- Custom document schemas or formats beyond the predefined categories
- Batch generation of multiple feeds in one command
- Feed directory merging or aggregation
- Statement verification after registration (future feature)
- Interactive editing of generated documents before signing
- Selective signing or registration (partial feed processing)
- Resume capability for interrupted workflows
- Selective retry of failed operations (must re-run entire command)
- Feed directory cleanup or archival tools
- Integration with external key management systems
- Multi-user or multi-tenant feed generation
- Feed replay or re-registration capabilities
- Custom Merkle tree tile sizes or configurations

## Assumptions

- **Tile Size**: Assuming standard tile size of 32 entries per tile (typical for RFC 6962-style transparency logs). This determines minimum document count requirement (96 = 3 complete tiles × 32)
- **Content Format**: Assuming JSON is the primary format for supply chain documents. SBOMs use SPDX JSON format as industry standard
- **Key Type**: Assuming ES256 (ECDSA with P-256 and SHA-256, COSE algorithm -7) for signing keys as it's widely supported in COSE implementations and provides good security/performance balance
- **Laptop Components**: Assuming a modern laptop contains: CPU, memory (RAM), firmware (UEFI/BIOS), storage controller, integrated graphics, and various chip components from the three company types
- **CVE/CWE Relevance**: Assuming AI-capable laptop firmware, NPU drivers, GPU drivers, AI frameworks, operating system, and pre-installed AI applications have associated real CVEs from NVD and CWE classifications. Examples: CVE-2024-0519 (GPU driver vulnerabilities), CVE-2023-52160 (NPU firmware issues), AI model poisoning vulnerabilities (CWE-502), and inference security issues
- **Mineral Sourcing**: Assuming supply chain includes conflict mineral sourcing (tin, tungsten, tantalum, gold) with origin tracking for regulatory compliance (Dodd-Frank Act)
- **Logistics Journey**: Assuming components travel from foundry → assembly plant → distribution center → retailer, with tracking documents at each step
- **SCITT Service**: Assuming a locally running SCITT transparency service at `http://127.0.0.1:56177` with API key authentication (standard for development/testing)
- **Environment Variables**: Assuming `SCITT_API_KEY` environment variable is set before registration (consistent with existing SCITT CLI patterns)
- **File System**: Assuming local file system with sufficient space (~5MB per feed) and write permissions in current directory
- **Company Names**: Using fictional but realistic company names (Pacific Silicon Foundry, Apex Semiconductor Corp, Quantum Chip Design) to avoid trademark issues while maintaining authenticity
- **Timestamp Format**: Using ISO 8601-like format (`YYYY-MM-DD-HHMMSS`) for feed directory names to ensure proper sorting and uniqueness
- **Network Connectivity**: Assuming network access to SCITT service during registration phase (offline generation and signing are supported)
- **Error Handling**: Assuming fail-fast behavior for generation/signing, but continue-on-error for registration (to maximize registration count even if some statements fail)

## Dependencies

- **SCITT Go CLI**: Feed generator extends existing `./scitt` CLI with new `feed` subcommand and `generate` sub-subcommand
- **COSE Library**: Requires existing COSE key generation, signing, and serialization functionality from Go COSE library used in SCITT implementation
- **SCITT Service**: Registration phase depends on running SCITT transparency service (SQLite or MongoDB/Cosmos DB backed)
- **Tile Storage**: Verification of tile creation depends on configured storage backend (local filesystem or Azure Blob Storage)
- **SHA-256**: Document hashing for COSE hash envelopes uses SHA-256 (widely supported, secure for supply chain use cases)
- **JSON Serialization**: Document generation requires standard Go JSON marshaling/unmarshaling
- **URN Format**: Subject and issuer identifiers follow URN (RFC 8141) format for globally unique, human-readable identifiers
- **SPDX Specification**: SBOM/HBOM documents should conform to SPDX 2.3+ JSON schema (ISO/IEC 5962:2021) for industry compatibility and regulatory compliance
- **CVE/CWE Format**: Vulnerability disclosure documents reference CVE IDs from MITRE's NVD (National Vulnerability Database) and CWE (Common Weakness Enumeration) classifications. Examples include GPU driver vulnerabilities, NPU firmware issues, and AI framework security concerns relevant to modern AI-capable laptops
- **HTTP Client**: Registration requires HTTP POST capability with bearer token authentication
- **File System Operations**: Directory creation, file reading/writing with proper permission handling

## Notes

- **Supply Chain Storytelling**: The generated dataset tells a coherent story of a laptop's supply chain journey from raw materials (mineral sourcing) through manufacturing (wafer fab, chip design) to final product (assembly, firmware, software) and post-production (vulnerability management). This narrative structure supports knowledge graph analysis and supply chain visualization tools
- **Semiconductor Industry Roles**:
  - **Foundry** (Pacific Silicon Foundry): Owns fabrication plants, manufactures silicon wafers, sources raw materials, tracks mineral origins
  - **IDM** (Apex Semiconductor Corp): Integrated Device Manufacturer, designs and manufactures CPUs, creates firmware, produces SBOMs/HBOMs
  - **Fabless** (Quantum Chip Design): Designs memory chips and controllers, licenses IP, produces firmware updates, discloses vulnerabilities (CVEs)
- **Document Distribution**: Documents are distributed by company role to create realistic supply chain scenarios:
  - **Foundry** (Pacific Silicon Foundry): Wafer batches (8-12), mineral sourcing (6-8), logistics tracking (8-12) = ~30 documents
  - **IDM** (Apex Semiconductor Corp): Chip specifications (10-14), firmware manifests (8-10), SBOMs/HBOMs (12-16) = ~36 documents
  - **Fabless** (Quantum Chip Design): Memory specifications (6-8), AI datasets (4-6), AI models (3-5), CVE/CWE (5-7) = ~24 documents
  - Total: 90-110 documents across all companies, exceeding 96 minimum for 3 complete tiles
- **SBOM Format Choice**: SPDX was selected over CycloneDX because:
  - **Industry Standard**: SPDX 2.3 is ISO/IEC 5962:2021, making it an international standard
  - **Regulatory Compliance**: U.S. Executive Order 14028 and NTIA SBOM guidance reference SPDX
  - **Broad Adoption**: Linux Foundation, OpenSSF, and major cloud providers use SPDX
  - **AI/ML Support**: SPDX 2.3+ includes fields for AI/ML components and datasets
- **AI Supply Chain Content**: Modern AI-capable laptops contain AI-specific supply chain artifacts that must be tracked:
  - **NPU (Neural Processing Unit)**: Dedicated AI accelerator chips with firmware requiring vulnerability tracking
  - **AI Training Datasets**: Pre-trained model datasets with provenance, licensing, and data sourcing requirements
  - **AI Model Specifications**: Model architectures (transformers, CNNs), quantization formats, inference performance
  - **AI Framework Vulnerabilities**: Security issues in TensorFlow, PyTorch, ONNX runtime affecting local AI applications
  - **Real CVE Examples for AI Laptops**:
    - CVE-2024-0519: NVIDIA GPU driver privilege escalation (affects integrated GPUs in AI laptops)
    - CVE-2024-3660: TensorFlow Keras arbitrary code execution via malicious model files (affects pre-installed AI apps)
    - CVE-2024-5480: PyTorch distributed RPC remote code execution (affects AI model training on laptops)
    - CVE-2024-22476: Intel Neural Compressor vulnerability (CVSS 10.0 critical, affects Intel AI optimization)
    - CWE-502: Deserialization of untrusted data (AI model poisoning attacks)
    - CWE-20: Improper input validation in AI inference engines
- **URN Design Philosophy**: URNs are designed to be:
  - **Human-readable**: `urn:supply-chain:pacific-silicon-foundry:wafer-batch:WF-2025-1001` clearly identifies company, type, and item
  - **Hierarchical**: Supports filtering and querying by company or document type
  - **Unique**: Combination of company + type + ID ensures global uniqueness
  - **Analysis-friendly**: Easy to parse for building knowledge graphs and supply chain visualizations
- **Real-World Applicability**: While data is synthetic, the structure, identifiers, and content types mirror real supply chain transparency systems, making analysis tools developed with this feed directly applicable to real-world scenarios
- **Extensibility**: Future versions could support:
  - Custom company configurations via YAML config file
  - Additional product types (smartphones, IoT devices, servers)
  - Pluggable document generators for custom supply chain artifacts
  - Feed templates for different industries (automotive, aerospace, pharmaceuticals)
  - Import of real supply chain data mixed with synthetic data
- **Testing Value**: This feed generator serves multiple purposes:
  - **SCITT Implementation Testing**: Validates transparency service handles high-volume registrations, tile generation, and Merkle tree construction
  - **Analysis Tool Development**: Provides realistic test data for supply chain visualization, compliance checking, and vulnerability tracking tools
  - **Demo Scenarios**: Creates compelling demonstrations of supply chain transparency with rich, interconnected data
  - **Performance Testing**: Enables benchmarking of SCITT services under load (100+ statement registrations)
- **Security Considerations**: While this generates synthetic data with real cryptographic signatures, all generated keys and data are for testing only and should never be used in production supply chain systems
- **Tile Calculation**: Minimum 3 tiles × 32 entries = 96 documents. Generating ~126 documents (42 per company) provides margin and ensures 4th partial tile, which is valuable for testing edge cases in tile log implementations
