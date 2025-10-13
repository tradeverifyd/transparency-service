# SCITT Integration Tests - Next Steps

**Updated:** 2025-10-12
**Current Status:** 9/9 operational tests passing (100%)
**COSE Fixtures:** ✅ **COMPLETED** - See `COSE-FIXTURES-COMPLETE.md`
**New Primary Focus:** Implementation difference handling in entry tests

## ✅ Completed: COSE Test Fixtures (2025-10-12)

**Achievement:** Successfully generated valid COSE Sign1 test fixtures using internal COSE package.

**Files Created:**
- `fixtures/statements/small.cose` (295 bytes)
- `fixtures/statements/medium.cose` (1,241 bytes)
- `fixtures/statements/large.cose` (10,239 bytes)

**Status:** Both Go and TypeScript servers successfully accept and process COSE Sign1 statements.

**Key Finding:** Tests revealed important implementation differences in response field naming (camelCase vs snake_case).

See detailed report: `COSE-FIXTURES-COMPLETE.md`

## Immediate Next Step: Handle Implementation Differences

### Current Situation

Entry tests are now operational and loading valid COSE Sign1 statements. However, tests revealed significant implementation differences that need to be handled:

**Response Field Naming:**
- **Go:** Uses camelCase (`entryId`, `statementHash`)
- **TypeScript:** Uses snake_case (`entry_id`, `receipt`)

**Entry ID Format:**
- **Go:** Integer (e.g., `1`)
- **TypeScript:** Base64url string (e.g., `"b9qeIRhJtLsfI7CdGnwX_NvFhjSBHPcjZaSrJwuOXUA"`)

### Tests Affected

1. ⚠️ `TestPostEntries` - Executes but reports field name differences
2. ❌ `TestGetEntries` - Panics on entry_id extraction (nil interface conversion)
3. ⚠️ `TestPostEntriesWithMultiplePayloads` - Fails validation (expects wrong field names)
4. ⚠️ `TestEntriesConcurrentRegistration` - Has directory/uniqueness issues
5. ❓ `TestGetEntriesReceipt` - Not yet tested

### Recommended Solution: Implementation-Aware Field Extraction

**Step 1:** Create helper function in `lib/response.go` (Est: 20 minutes)

```go
// ExtractEntryID extracts entry_id from response, handling implementation differences
func ExtractEntryID(data map[string]interface{}, impl string) (string, error) {
    if impl == "go" {
        // Go uses camelCase "entryId" as integer
        if id, ok := data["entryId"].(float64); ok {
            return fmt.Sprintf("%d", int(id)), nil
        }
        return "", fmt.Errorf("entryId not found or invalid type in Go response")
    }

    // TypeScript uses snake_case "entry_id" as string
    if id, ok := data["entry_id"].(string); ok {
        return id, nil
    }
    return "", fmt.Errorf("entry_id not found or invalid type in TypeScript response")
}

// ValidateRegistrationResponse validates POST /entries response fields
func ValidateRegistrationResponse(data map[string]interface{}, impl string) []string {
    var missing []string

    if impl == "go" {
        // Go returns: entryId, statementHash
        if _, ok := data["entryId"]; !ok {
            missing = append(missing, "entryId")
        }
        if _, ok := data["statementHash"]; !ok {
            missing = append(missing, "statementHash")
        }
    } else {
        // TypeScript returns: entry_id, receipt
        if _, ok := data["entry_id"]; !ok {
            missing = append(missing, "entry_id")
        }
        // Note: statementHash may be omitted in TypeScript
    }

    return missing
}
```

**Step 2:** Update `http/entries_test.go` to use helpers (Est: 15 minutes)

```go
// In TestPostEntries
requiredFields := lib.ValidateRegistrationResponse(goData, "go")
if len(requiredFields) > 0 {
    t.Errorf("Go response missing required fields: %v", requiredFields)
}

requiredFields = lib.ValidateRegistrationResponse(tsData, "typescript")
if len(requiredFields) > 0 {
    t.Errorf("TypeScript response missing required fields: %v", requiredFields)
}

// In TestGetEntries
goEntryID, err := lib.ExtractEntryID(goRegData, "go")
if err != nil {
    t.Fatalf("Failed to extract Go entry ID: %v", err)
}

tsEntryID, err := lib.ExtractEntryID(tsRegData, "typescript")
if err != nil {
    t.Fatalf("Failed to extract TypeScript entry ID: %v", err)
}
```

**Step 3:** Fix concurrent registration test (Est: 25 minutes)

Generate unique statements per request:

```go
func generateUniqueStatement(t *testing.T, size string, index int) []byte {
    t.Helper()

    // Create unique payload
    payload := map[string]interface{}{
        "type":    "test-statement",
        "size":    size,
        "index":   index,
        "nonce":   fmt.Sprintf("%d-%d", time.Now().UnixNano(), index),
        "issuer":  "https://example.com/test",
        "subject": fmt.Sprintf("test-subject-%d", index),
    }

    payloadBytes, _ := json.Marshal(payload)

    // Sign with COSE (use existing tool logic)
    return signPayloadCOSE(t, payloadBytes)
}
```

### Time Estimate

- Create helper functions: 20 minutes
- Update TestPostEntries: 10 minutes
- Update TestGetEntries: 5 minutes
- Update TestPostEntriesWithMultiplePayloads: 5 minutes
- Fix concurrent test: 25 minutes
- Test and debug: 15 minutes
- **Total: ~1.5 hours**

### Expected Outcome

After implementing these changes:
- ✅ TestPostEntries validates correctly for both implementations
- ✅ TestGetEntries no longer panics
- ✅ TestPostEntriesWithMultiplePayloads passes
- ✅ TestEntriesConcurrentRegistration handles concurrent requests
- 📈 4-5 entry tests passing (up from 0 currently fully passing)

### Implementation Options

#### Option 1: Use Go Implementation's COSE Package (Recommended)

The Go implementation at `scitt-golang/pkg/cose/` has complete COSE Sign1 signing:

**Key files:**
- `hash_envelope.go` - `SignHashEnvelope()` function
- `sign.go` - `CreateCoseSign1()` function
- `signer.go` - `Signer` interface
- `es256.go` - ES256 signer implementation

**Approach:**
```go
// In tools/generate_cose_statement.go
import "github.com/tradeverifyd/scitt/scitt-golang/pkg/cose"

// 1. Create ES256 signer from keypair
signer, err := cose.NewES256Signer(privateKey)

// 2. Create protected headers
headers := cose.ProtectedHeaders{
    cose.HeaderLabelAlg: cose.AlgorithmES256,
}

// 3. Sign payload
coseSign1, err := cose.CreateCoseSign1(headers, payloadBytes, signer, cose.CoseSign1Options{})

// 4. Marshal to CBOR
coseBytes, err := coseSign1.MarshalCBOR()

// 5. Save to fixtures/statements/{size}.cose
```

**Pros:**
- Uses same code as production
- Guaranteed compatibility
- Already has all dependencies

**Cons:**
- Needs to import internal package
- May need to adjust module paths

#### Option 2: Use TypeScript Implementation

Generate fixtures using the TypeScript CLI:

```bash
cd scitt-typescript
bun run src/cli/index.ts transparency statement sign \
  --payload ../tests/interop/fixtures/payloads/small.json \
  --key service-key.json \
  --output ../tests/interop/fixtures/statements/small.cose
```

**Pros:**
- Uses CLI (public interface)
- No Go code needed
- Simple shell script

**Cons:**
- Depends on TypeScript CLI being fully implemented
- May have different COSE structure than Go
- Need to verify TypeScript CLI has this command

#### Option 3: Use go-cose Library Directly (Started)

**File:** `tools/generate_cose_statement.go` (partially created)

**Status:** Started but blocked on API differences with `github.com/veraison/go-cose`

The Go implementation uses a custom COSE package, not the standard go-cose library.

**Pros:**
- Standalone tool
- No internal dependencies

**Cons:**
- Need to learn go-cose API
- May produce different structure than implementations expect
- Extra maintenance burden

### Recommended Approach

**Use Option 1** - Import Go implementation's COSE package

**Steps:**

1. **Update `tools/go.mod` to use internal package:**
```go
module github.com/tradeverifyd/scitt/tests/interop/tools

require github.com/tradeverifyd/scitt/scitt-golang v0.0.0

replace github.com/tradeverifyd/scitt/scitt-golang => ../../scitt-golang
```

2. **Rewrite `generate_cose_statement.go`:**
```go
import (
    "github.com/tradeverifyd/scitt/scitt-golang/pkg/cose"
    "github.com/tradeverifyd/scitt/scitt-golang/pkg/keys"
)

func signCOSE(payload []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
    // Create signer
    signer, err := cose.NewES256Signer(privateKey)
    if err != nil {
        return nil, err
    }

    // Create headers
    headers := cose.ProtectedHeaders{
        cose.HeaderLabelAlg: cose.AlgorithmES256,
    }

    // Sign
    sign1, err := cose.CreateCoseSign1(headers, payload, signer, cose.CoseSign1Options{})
    if err != nil {
        return nil, err
    }

    // Marshal
    return sign1.MarshalCBOR()
}
```

3. **Generate fixtures:**
```bash
cd tests/interop/tools
go run generate_cose_statement.go -size small
go run generate_cose_statement.go -size medium
go run generate_cose_statement.go -size large
```

4. **Update `http/entries_test.go`:**
```go
func loadTestStatement(t *testing.T, size string) []byte {
    t.Helper()

    // Load COSE Sign1 statement from fixtures
    statementPath := filepath.Join("..", "fixtures", "statements", fmt.Sprintf("%s.cose", size))

    data, err := os.ReadFile(statementPath)
    if err != nil {
        t.Fatalf("Failed to load test statement: %v", err)
    }

    return data
}
```

5. **Run entry tests:**
```bash
cd tests/interop
go test -v ./http/... -run "TestPostEntries"
```

**Expected Result:** 5 entry tests should pass

### Time Estimate

- Update tool: 30 minutes
- Generate fixtures: 5 minutes
- Update test: 10 minutes
- Run and debug: 15 minutes
- **Total: ~1 hour**

### Impact

**Unlocks:** 5 entry endpoint tests
**New Total:** 14 operational tests (from current 9)
**Pass Rate:** Should remain 100%

## After COSE Fixtures

### Next Priority: Checkpoint Tests (3-4 tests)

File: `http/checkpoint_test.go`

**Tests:**
- `TestCheckpoint` - Basic checkpoint retrieval
- `TestCheckpointConsistency` - Tree growth validation
- `TestCheckpointFormat` - RFC 6962 compliance

**Requirements:**
- Servers support GET `/checkpoint`
- Both implementations return signed tree heads
- Validate signature format

**Time Estimate:** 1-2 hours

### Phase 5: Cryptographic Interoperability (10+ tests)

**Tests:**
- Cross-signature verification (Go signs → TS verifies)
- Cross-signature verification (TS signs → Go verifies)
- Key format bridge (PEM ↔ JWK)
- JWK thumbprint consistency

**Requirements:**
- Implement `lib/keys.go` with conversion functions
- Update statement tests
- Create cross-verification test matrices

**Time Estimate:** 4-6 hours

### Phase 6: Merkle Proof Validation (8-10 tests)

**Tests:**
- Inclusion proof cross-validation
- Consistency proof cross-validation
- Root hash consistency
- Tile naming validation

**Requirements:**
- Both implementations support proof generation
- Implement proof verification in test library
- Validate C2SP tlog-tiles compliance

**Time Estimate:** 6-8 hours

## Long-Term Roadmap

| Phase | Tests | Est. Time | Priority | Status |
|-------|-------|-----------|----------|--------|
| ~~COSE Fixtures~~ | ~~5~~ | ~~1 hour~~ | ~~HIGH~~ | ✅ **DONE** |
| Entry Test Fixes | 5 | 1.5 hours | **HIGH** | 🔄 Next |
| Checkpoint | 3-4 | 2 hours | **HIGH** | Pending |
| Phase 5: Crypto | 10+ | 6 hours | Medium | Pending |
| Phase 6: Proofs | 8-10 | 8 hours | Medium | Pending |
| Phase 7: Queries | 6 | 4 hours | Low | Pending |
| Phase 8: Receipts | 7 | 6 hours | Low | Pending |
| Phase 9: Database | 5 | 4 hours | Low | Pending |
| Phase 10: E2E | 5 | 6 hours | Low | Pending |
| Phase 11: Performance | 7 | 8 hours | Low | Pending |

**Completed:** 1 phase (COSE Fixtures)
**Total Potential:** 60+ tests across all phases

## Success Metrics

**Current:**
- 9 operational tests
- 100% pass rate
- Sub-millisecond performance
- Production-ready infrastructure

**Target (After COSE + Checkpoint):**
- 18 operational tests
- 100% pass rate maintained
- Full CRUD testing coverage
- RFC compliance validated

**Ultimate Goal (All Phases):**
- 65+ comprehensive tests
- End-to-end workflow validation
- Performance benchmarking
- Complete RFC compliance matrix

## Getting Started

To continue from where this session ended:

```bash
# 1. Navigate to tools directory
cd tests/interop/tools

# 2. Update go.mod with internal package
# (see Option 1 above)

# 3. Rewrite generate_cose_statement.go
# (use code snippet from Option 1)

# 4. Generate fixtures
go run generate_cose_statement.go -size small
go run generate_cose_statement.go -size medium
go run generate_cose_statement.go -size large

# 5. Verify fixtures created
ls -lh ../fixtures/statements/

# 6. Run entry tests
cd ..
go test -v ./http/... -run "TestPostEntries"

# 7. Celebrate when all 5 pass! 🎉
```

## Questions?

Refer to comprehensive documentation:
- `TESTING-GUIDE.md` - How to run tests
- `TEST-STATUS.md` - Current status details
- `SESSION-COMPLETE.md` - Session summary
- `FINAL-STATUS.md` - Complete status report

The test infrastructure is solid. Adding COSE fixtures is the only blocker for significant expansion.

---

**Summary:** 1 hour of work unlocks 5 more tests (55% increase in test coverage)
