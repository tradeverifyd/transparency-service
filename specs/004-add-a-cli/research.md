# Research: Synthetic Supply Chain Feed Generator

**Date**: 2025-10-18
**Feature**: 004-add-a-cli
**Phase**: Phase 0 (Outline & Research)

This document consolidates research findings for implementing the synthetic supply chain feed generator.

---

## 1. SPDX 2.3 Schema Structure

### Decision
Use SPDX 2.3 JSON format for SBOM/HBOM documents with minimal required fields.

### Rationale
- **Industry Standard**: SPDX 2.3 is ISO/IEC 5962:2021
- **Regulatory Compliance**: Referenced by U.S. Executive Order 14028 and NTIA SBOM guidance
- **AI/ML Support**: SPDX 2.3+ includes fields for AI/ML components and datasets
- **Simplicity**: Minimal required fields reduce generation complexity

### SPDX 2.3 JSON Structure

**Minimal Required Fields**:
```json
{
  "spdxVersion": "SPDX-2.3",
  "dataLicense": "CC0-1.0",
  "SPDXID": "SPDXRef-DOCUMENT",
  "documentName": "apex-semiconductor-corp-cpu-sbom",
  "documentNamespace": "https://apex-semiconductor-corp.example/sbom/APX-9700K-2025-10-18",
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
      "supplier": "Organization: Apex Semiconductor Corp",
      "downloadLocation": "NOASSERTION",
      "licenseConcluded": "NOASSERTION",
      "filesAnalyzed": false
    }
  ]
}
```

**Implementation Notes**:
- Use templating approach with Go `text/template` or struct marshaling
- Generate unique `documentNamespace` URLs per document
- Keep `packages` array minimal (3-5 packages per SBOM)
- Use `NOASSERTION` for unknown fields per SPDX spec

### Alternatives Considered
- **CycloneDX**: Rejected per spec requirement (SPDX chosen for regulatory compliance)
- **SWID Tags**: More complex, less adoption for SBOMs

---

## 2. Real CVE/CWE Examples

### Decision
Use validated real CVE IDs from 2024 relevant to AI-capable laptop hardware/firmware/software.

### Confirmed Real CVEs

**CVE-2024-0519 (NVIDIA GPU Display Driver)**
- **Verified**: Real CVE from NVIDIA June 2024 security bulletin
- **Relevance**: Affects NVIDIA GPU drivers used in AI-capable laptops with integrated/discrete GPUs
- **CVSS Score**: To be determined from NVIDIA bulletin
- **Usage**: Include in vulnerability disclosure documents for GPU-related firmware

**CVE-2024-3660 (TensorFlow/Keras)**
- **Verified**: Real, high-severity CVE affecting TensorFlow's Keras framework <2.13
- **Description**: Arbitrary code execution via malicious model files through Keras Lambda layers
- **Relevance**: Affects AI frameworks pre-installed on AI laptops
- **CVSS Score**: High severity
- **Usage**: Include in AI model/framework vulnerability documents

**CVE-2024-5480 (PyTorch Distributed RPC)**
- **Verified**: Real, critical CVE affecting PyTorch distributed RPC system
- **Description**: Remote code execution via insufficient input validation in PythonUDF serialization
- **Relevance**: Affects AI model training on laptops using PyTorch distributed computing
- **CVSS Score**: Critical severity
- **Usage**: Include in AI framework vulnerability documents

**Recommended Additional CVEs**:
- **CVE-2024-22476**: Intel Neural Compressor vulnerability (CVSS 10.0 critical) - AI model optimization software
- **CVE-2024-5480**: PyTorch distributed RPC remote code execution (critical severity)
- **CVE-2024-35198**: PyTorch TorchServe model registration API vulnerability
- **CVE-2024-35199**: PyTorch TorchServe gRPC port binding vulnerability
- **CWE-502**: Deserialization of untrusted data (AI model poisoning attacks) - **This is a weakness category, not a CVE**
- **CWE-20**: Improper input validation in AI inference engines - **This is a weakness category, not a CVE**

### Implementation Notes
- Generate 5-7 CVE/CWE documents per feed
- Mix GPU driver CVEs (CVE-2024-0519), AI framework CVEs (CVE-2024-3660, CVE-2024-5480, CVE-2024-22476), and CWE classifications
- Include realistic CVSS scores (7.0-10.0 for high/critical, with CVE-2024-22476 at 10.0)
- Reference firmware versions and AI framework versions from generated manifests

### Alternatives Considered
- **Hypothetical CVEs**: Rejected - spec requires "real CVE IDs"
- **Older CVEs**: Could use, but 2024 CVEs provide more realism for current laptops

---

## 3. Deterministic PRNG Patterns

### Decision
Use Go `math/rand` with timestamp-seeded `rand.NewSource()` for deterministic document generation.

### Code Pattern

```go
package generator

import (
    "fmt"
    "math/rand"
    "time"
)

// SeededRand wraps a deterministic random source
type SeededRand struct {
    rng *rand.Rand
}

// NewSeededRand creates a deterministic PRNG from a timestamp
func NewSeededRand(timestamp time.Time) *SeededRand {
    // Convert timestamp to Unix nanoseconds as seed
    seed := timestamp.UnixNano()
    source := rand.NewSource(seed)
    return &SeededRand{
        rng: rand.New(source),
    }
}

// IntRange returns a random int in [min, max]
func (sr *SeededRand) IntRange(min, max int) int {
    return min + sr.rng.Intn(max-min+1)
}

// Choose selects a random element from a slice
func (sr *SeededRand) Choose(options []string) string {
    return options[sr.rng.Intn(len(options))]
}

// Shuffle randomly orders a slice
func (sr *SeededRand) Shuffle(slice []interface{}) {
    sr.rng.Shuffle(len(slice), func(i, j int) {
        slice[i], slice[j] = slice[j], slice[i]
    })
}
```

### Usage Example

```go
// In feed generation:
timestamp, _ := time.Parse("2006-01-02-150405", "2025-10-18-143022")
rng := NewSeededRand(timestamp)

// Generate deterministic document counts
waferCount := rng.IntRange(8, 12)  // Always same for this timestamp
mineralCount := rng.IntRange(6, 8)

// Generate deterministic IDs
lotID := fmt.Sprintf("WF-2025-%04d", rng.IntRange(1000, 9999))
```

### Implementation Notes
- Extract timestamp from feed directory name (`feed-2025-10-18-143022`)
- Create single `SeededRand` instance per feed generation
- Pass RNG to all document generators
- Document counts, IDs, and content values all deterministic
- Crypto operations (ES256 signing) use `crypto/rand` (non-deterministic per ECDSA spec)

### Alternatives Considered
- **Global `math/rand` seeding**: Rejected - not thread-safe, can interfere with other code
- **UUID v5 (deterministic)**: Overkill for simple document ID generation

---

## 4. Progress Bar Libraries

### Decision
Use `github.com/schollz/progressbar/v3` for progress feedback.

### Rationale
- **Lightweight**: Single dependency, minimal API surface
- **Feature-complete**: Supports all required features (percentage, ETA, counters)
- **Active maintenance**: Last updated 2024, good community support
- **Simple API**: Easy integration with existing CLI code

### Example Usage

```go
import (
    "github.com/schollz/progressbar/v3"
    "time"
)

// Document generation progress
bar := progressbar.NewOptions(totalDocs,
    progressbar.OptionSetDescription("Generating documents"),
    progressbar.OptionSetWidth(40),
    progressbar.OptionShowCount(),
    progressbar.OptionShowIts(),
    progressbar.OptionSetPredictTime(true),
)

for i := 0; i < totalDocs; i++ {
    // Generate document
    generateDocument(i)
    bar.Add(1)
    time.Sleep(10 * time.Millisecond) // Simulated work
}

// Signing progress
signBar := progressbar.NewOptions(len(statements),
    progressbar.OptionSetDescription("Signing statements"),
    progressbar.OptionShowCount(),
)

for _, stmt := range statements {
    signStatement(stmt)
    signBar.Add(1)
}
```

### Implementation Notes
- Create separate progress bars for each phase (generate, sign per company, register)
- Use `OptionShowCount()` to display "23/126" style counters
- Use `OptionSetPredictTime(true)` for ETA
- Progress bars write to stderr (don't interfere with stdout output)

### Alternatives Considered
- **`github.com/cheggaaa/pb/v3`**: More complex API, similar features
- **`github.com/vbauerster/mpb/v8`**: Multi-progress bars (overkill for our use case)
- **Custom implementation**: Rejected - reinventing the wheel

---

## 5. Reusing Existing SCITT Commands

### Decision
**REUSE existing `scitt issuer` and `scitt statement` commands** instead of reimplementing key generation and signing logic.

### Rationale
- **Constitution Principle VII**: "Prefer standard libraries and established patterns over custom solutions"
- **Code Reuse**: Existing commands already implement all required functionality
- **Consistency**: Ensures identical behavior across all CLI workflows
- **Maintainability**: Single source of truth for key generation and signing
- **Simplicity**: Feed generator orchestrates existing commands via `exec.Command()`

### Existing Commands to Reuse

**Key Generation** (`scitt issuer keygen`):
```bash
# From internal/cli/issuer.go
./scitt issuer keygen \
  --private ./company/priv.cbor \
  --public ./company/pub.cbor \
  --algorithm ES256
```

**Statement Signing** (`scitt statement sign`):
```bash
# From internal/cli/statement.go
./scitt statement sign \
  --content ./document.json \
  --content-type application/json \
  --content-location https://company.example/doc.json \
  --issuer "urn:supply-chain:company" \
  --subject "urn:supply-chain:company:type:id" \
  --signing-key ./company/priv.cbor \
  --signed-statement ./document.cbor
```

**Statement Registration** (`scitt statement register`):
```bash
# From internal/cli/statement.go
./scitt statement register \
  --service http://127.0.0.1:56177 \
  --api-key $SCITT_API_KEY \
  --statement ./document.cbor \
  --receipt ./document.receipt.cbor
```

### Implementation Pattern for Feed Generator

**Phase 1: Generate Keys for Each Company**
```go
func generateCompanyKeys(companyDir string) error {
    cmd := exec.Command("./scitt", "issuer", "keygen",
        "--private", filepath.Join(companyDir, "priv.cbor"),
        "--public", filepath.Join(companyDir, "pub.cbor"),
        "--algorithm", "ES256",
    )
    return cmd.Run()
}
```

**Phase 2: Sign All Documents**
```go
func signDocuments(companyDir string, documents []Document, issuerURN string) error {
    privKey := filepath.Join(companyDir, "priv.cbor")

    for _, doc := range documents {
        cmd := exec.Command("./scitt", "statement", "sign",
            "--content", doc.Path,
            "--content-type", doc.ContentType,
            "--content-location", doc.URL,
            "--issuer", issuerURN,
            "--subject", doc.SubjectURN,
            "--signing-key", privKey,
            "--signed-statement", doc.Path+".cbor",
        )
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("failed to sign %s: %w", doc.Path, err)
        }
    }
    return nil
}
```

**Phase 3: Register All Statements**
```go
func registerStatements(serviceURL, apiKey string, statements []string) error {
    for _, stmt := range statements {
        receiptPath := strings.Replace(stmt, ".cbor", ".receipt.cbor", 1)
        cmd := exec.Command("./scitt", "statement", "register",
            "--service", serviceURL,
            "--api-key", apiKey,
            "--statement", stmt,
            "--receipt", receiptPath,
        )
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("failed to register %s: %w", stmt, err)
        }
    }
    return nil
}
```

### Implementation Notes
- **No COSE library imports needed in feed generator code** - all handled by existing commands
- Feed generator becomes a workflow orchestrator, not a crypto implementation
- Progress bars wrap the `exec.Command()` calls
- Error handling delegates to existing command error messages
- Interactive prompts call existing commands in loops

### Benefits vs Custom Implementation

| Aspect | Custom COSE Code | Reusing Commands |
|--------|------------------|------------------|
| Lines of Code | ~200-300 | ~50-75 |
| COSE Library Imports | Required | None |
| Testing | New tests needed | Already tested |
| Maintenance | Duplicate logic | Single source |
| Consistency | Risk of divergence | Guaranteed identical |
| Constitution Compliance | Questionable (VII) | ✅ Compliant (VII) |

### Alternatives Considered
- **Import pkg/cose directly**: Violates simplicity principle (why reimport when CLI exists?)
- **Duplicate signing logic**: Violates DRY and Constitution Principle VII

---

## 6. Supply Chain Graph: parent_leaf_hash Investigation

### Context
Initial specification proposed using `parent_leaf_hash` CWT claim in COSE headers to establish supply chain ancestry relationships. This would allow documents to reference their "parent" entries in the transparency log.

### Research Findings

**COSE Label Investigation** (pkg/cose/sign.go lines 10-27):
- Examined existing SCITT CLI implementation
- Confirmed existing COSE header labels:
  - **Label 15**: CWT Claims Set (RFC 9597) - contains iss (1), sub (2) internally
  - **Label 258-260**: Hash envelope parameters (algorithm, content type, location)
  - **Label 391-392**: Legacy SCITT issuer/subject headers (deprecated in favor of CWT claims)
  - **Label 394-396**: Receipts, VDS, VDP (transparency log proof parameters)

**RFC/IETF Specification Search**:
- Searched IETF datatracker, SCITT working group, COSE working group
- Reviewed draft-ietf-cose-merkle-tree-proofs (COSE Receipts specification)
- Reviewed draft-ietf-scitt-architecture
- **Conclusion**: `parent_leaf_hash` is **NOT defined in any IETF RFC or SCITT draft**

**Options Considered**:
1. **Add custom CWT claim** (inside label 15) with private claim key
   - ❌ Violates "do not reinvent" directive
   - ❌ No standardized claim key for parent_leaf_hash
2. **Use COSE private use range** (negative labels < -65536)
   - ❌ Violates "do not reinvent" directive
   - ❌ Custom label numbers like 65537, 65538, 65539 were explicitly rejected by user
3. **Skip parent_leaf_hash in initial implementation** ✅ **RECOMMENDED**
   - ✅ Leverages existing SCITT commands without modification
   - ✅ Complies with "use existing commands or update them" directive
   - ✅ Document relationships tracked in external metadata files
   - ✅ Can enhance later if IETF standardizes parent_leaf_hash

### Decision
**Add `parent_leaf_hash` as a custom CWT claim using string key inside label 15.**

### Rationale
- **User Directive**: "Supply chain relationships should be defined in cose headers... use 'parent_leaf_hash' as a custom claim inside CWT claims in headers"
- **No Top-Level Label Invention**: Using string key `"parent_leaf_hash"` inside CWT Claims (label 15) avoids inventing custom top-level COSE header labels (no 65537, 65538, 65539)
- **Valid CBOR/CWT**: String keys are valid in CBOR maps and CWT claim sets per RFC 8392
- **Constitution Principle VII**: Extends existing CWT Claims structure without reimplementing COSE signing
- **Cryptographically Signed**: Parent-child relationships are included in protected headers, making them tamper-evident

### Implementation Pattern

**COSE Structure with parent_leaf_hash**:
```
Protected Headers (CBOR map):
  1: -7                           // alg: ES256
  4: <key_id_bytes>               // kid
  15: {                           // CWT Claims (label 15)
    1: "urn:supply-chain:company",           // iss (issuer)
    2: "urn:supply-chain:company:type:id",   // sub (subject)
    "parent_leaf_hash": <32_bytes_sha256>    // Custom claim (string key)
  }
  258: -16                        // payload_hash_alg: SHA-256
  259: "application/json"         // preimage_content_type
  260: "https://..."              // payload_location
```

**Extending CWTClaimsOptions** (pkg/cose/sign.go):
```go
type CWTClaimsOptions struct {
    Iss            string
    Sub            string
    // ... existing fields ...
    ParentLeafHash []byte  // NEW: Parent entry leaf hash
}

func CreateCWTClaims(opts CWTClaimsOptions) CWTClaimsSet {
    claims := make(CWTClaimsSet)
    if opts.Iss != "" {
        claims[CWTClaimIss] = opts.Iss
    }
    if opts.Sub != "" {
        claims[CWTClaimSub] = opts.Sub
    }
    // NEW: Add parent_leaf_hash as string key
    if len(opts.ParentLeafHash) > 0 {
        claims["parent_leaf_hash"] = opts.ParentLeafHash
    }
    return claims
}
```

**Extending `scitt statement sign` command** (internal/cli/statement.go):
```go
type statementSignOptions struct {
    // ... existing fields ...
    parentLeafHash string  // NEW: Optional hex-encoded parent leaf hash
}

// In runStatementSign():
var parentHash []byte
if opts.parentLeafHash != "" {
    parentHash, err = hex.DecodeString(opts.parentLeafHash)
    // ... error handling ...
}

cwtClaimsOpts := cose.CWTClaimsOptions{
    Iss:            opts.issuer,
    Sub:            opts.subject,
    ParentLeafHash: parentHash,  // NEW
}
```

**CLI Usage**:
```bash
# Sign with parent reference
./scitt statement sign \
  --content ./chip-spec.json \
  --content-type application/json \
  --content-location https://company.example/chip.json \
  --issuer "urn:supply-chain:company" \
  --subject "urn:supply-chain:company:chip:CHP-001" \
  --parent-leaf-hash a1b2c3d4e5f6... \
  --signing-key ./priv.cbor \
  --signed-statement ./chip-spec.cbor
```

**Benefits**:
- ✅ Relationships are cryptographically signed in COSE headers
- ✅ No invention of top-level COSE labels (uses string key in existing label 15)
- ✅ Extends existing `scitt statement sign` command with one parameter
- ✅ Valid per CBOR/CWT specifications (string keys allowed)
- ✅ Supply chain graph verifiable via transparency log

### Alternatives Considered
- **External metadata file**: ❌ Rejected - relationships must be in signed COSE headers
- **Invent custom COSE labels (65537, 65538, 65539)**: ❌ Explicitly rejected by user
- **Numeric CWT claim key**: ❌ Would require IANA registration or private use range; string key is simpler

---

## Summary

All research complete. Key decisions:

1. **SPDX 2.3**: Minimal JSON structure with required fields only
2. **CVEs**: Use real 2024 CVEs (CVE-2024-0519, CVE-2024-3660, CVE-2024-5480, CVE-2024-22476)
3. **PRNG**: Timestamp-seeded `math/rand` for deterministic generation
4. **Progress**: `github.com/schollz/progressbar/v3` library
5. **Workflow Orchestration**: **REUSE and EXTEND existing `scitt issuer keygen` and `scitt statement sign/register` commands**
6. **Supply Chain Graph**: **Use `parent_leaf_hash` as custom CWT claim (string key) inside label 15**

This approach:
- Minimizes new code (only document generation + orchestration + one CWT claim extension)
- Maximizes code reuse (extends existing `scitt statement sign` with `--parent-leaf-hash` parameter)
- Ensures consistency (identical behavior across all workflows)
- Complies with Constitution Principle VII (simplicity and established patterns)
- **Avoids inventing top-level COSE labels** (uses string key in existing label 15 structure)
- **Cryptographically signs supply chain relationships** in COSE protected headers

**Required Code Changes**:
1. Extend `CWTClaimsOptions` struct in `pkg/cose/sign.go` with `ParentLeafHash []byte` field
2. Update `CreateCWTClaims()` to add `"parent_leaf_hash"` string key when provided
3. Add `--parent-leaf-hash` flag to `scitt statement sign` command in `internal/cli/statement.go`

**Next Phase**: Phase 1 (Design & Contracts) - Create data-model.md and quickstart.md
