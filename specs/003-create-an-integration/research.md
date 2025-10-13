# Research: Cross-Implementation Integration Test Suite

**Feature**: Cross-Implementation Integration Test Suite
**Phase**: 0 - Research & Technical Decisions
**Date**: 2025-10-12

## Purpose

This document records research findings and technical decisions made during the planning phase to resolve unknowns identified in the Technical Context. All decisions are informed by RFC specifications, Go tlog reference implementation, and best practices for cross-implementation validation.

---

## Decision 1: Test Orchestration Strategy

**Question**: How to orchestrate tests across two different language implementations (Go and TypeScript/Bun)?

**Decision**: Use Go's `testing` package as the primary test orchestrator, with shell scripts bridging to TypeScript/Bun CLI invocations.

**Rationale**:
- Go `testing` package provides excellent parallel execution, subtests, and CI integration
- Go is already used in CI for the Go implementation, minimizing additional CI dependencies
- Shell scripts are portable across macOS/Linux for local developer testing
- Go's `os/exec` package provides clean process management for CLI invocations
- Go's `httptest` can easily start/stop servers and capture outputs

**Alternatives Considered**:
1. **Pure TypeScript/Bun tests** - Rejected because invoking Go binaries from Bun adds complexity and Bun test ecosystem is less mature for CI integration
2. **Python pytest orchestration** - Rejected because it adds a third language dependency and team has less Python expertise
3. **Shell scripts only** - Rejected because lacks structured test organization, parallel execution, and clean reporting

**Implementation Notes**:
- Use `t.Run()` for subtest organization by capability (CLI, HTTP, crypto, Merkle)
- Use `t.Parallel()` for independent tests to achieve <5 min runtime goal
- Shell scripts will be idempotent and handle cleanup on failure

---

## Decision 2: RFC Test Vector Generation

**Question**: How to generate authoritative test vectors that validate both implementations against RFCs rather than each other?

**Decision**: Generate canonical test vectors using Go's `golang.org/x/mod/sumdb/tlog` package and Go cryptography libraries, then validate both Go and TypeScript implementations against these vectors.

**Rationale**:
- Aligns with Constitution Principle VIII (Go as canonical reference)
- Go `sumdb/tlog` is the de facto standard for RFC 6962 Merkle tree operations
- Go's `crypto/ecdsa` and COSE libraries directly implement RFCs 9052, 8392
- Test vectors generated once during test setup, then reused for validation
- Prevents circular validation (implementation testing itself)

**Alternatives Considered**:
1. **Use TypeScript implementation to generate vectors** - Rejected because violates Principle VIII
2. **Manually create test vectors from RFC examples** - Rejected because insufficient coverage and error-prone
3. **Use external test vector repositories** - Rejected because no comprehensive SCITT test vector repository exists

**Implementation Notes**:
- Test vector generation will be a separate tool in `tests/interop/tools/generate_vectors.go`
- Vectors will be stored in `tests/interop/fixtures/rfc-vectors/` as JSON files
- Vector format will include: input data, expected COSE Sign1 bytes, expected Merkle proofs, expected checksums
- Vectors will be versioned and regenerated if RFC interpretations change

---

## Decision 3: API Response Comparison Strategy

**Question**: How to compare HTTP API responses from Go and TypeScript servers when JSON field ordering may differ?

**Decision**: Perform semantic JSON comparison using deep equality checks on parsed JSON objects, not string comparison. Use `snake_case` field names as specified by user.

**Rationale**:
- JSON objects with different key ordering are semantically equivalent per RFC 8259
- String comparison would report false positives for equivalent responses
- Go's `reflect.DeepEqual` and TypeScript's JSON comparison utilities handle nested objects
- Semantic comparison aligns with RFC compliance validation (content matters, not serialization)

**Alternatives Considered**:
1. **Normalize JSON key ordering before comparison** - Rejected because unnecessary complexity and doesn't handle nested objects well
2. **Use JSON Schema validation** - Considered for addition, but semantic comparison is more direct
3. **String diff of JSON** - Rejected because too brittle and reports false positives

**Implementation Notes**:
- Comparison utility will be in `tests/interop/lib/compare.go`
- Function signature: `func CompareJSON(a, b []byte) (equal bool, diffs []string, err error)`
- Will report specific field differences with JSON paths (e.g., `"$.entry_id"` mismatch)
- Will handle numeric precision differences (int vs float for same value)
- Will use hex encoding for identifiers per user requirement

---

## Decision 4: CLI Command Invocation and Output Capture

**Question**: How to invoke CLI commands from both implementations and capture structured output for comparison?

**Decision**: Use Go's `os/exec` to invoke CLI commands, capture stdout/stderr, parse structured output (JSON), and compare. Use dedicated shell script wrapper for complex CLI scenarios.

**Rationale**:
- `os/exec.Command` provides clean process management with timeout support
- Can set environment variables for test isolation (temp dirs, ports)
- Stdout/stderr capture enables detailed failure diagnostics
- Shell script wrapper handles multi-step CLI workflows (init → serve → test → cleanup)

**Alternatives Considered**:
1. **Direct library calls** - Rejected because bypasses CLI layer (we need to test CLI UX)
2. **Expect/TCL scripts** - Rejected because adds dependency and overkill for non-interactive CLI
3. **Docker exec** - Rejected because slower and unnecessary for local testing

**Implementation Notes**:
- CLI invocation utility in `tests/interop/lib/cli.go`
- Function signatures:
  ```go
  func RunGoCLI(args []string, env map[string]string) (stdout, stderr string, exitCode int, err error)
  func RunTsCLI(args []string, env map[string]string) (stdout, stderr string, exitCode int, err error)
  ```
- Timeouts: 30s for init/serve commands, 10s for sign/verify commands
- Environment variables for isolation: `SCITT_DB_PATH`, `SCITT_STORAGE_PATH`, `SCITT_PORT`

---

## Decision 5: Test Data Management and Cleanup

**Question**: How to manage test databases, storage directories, and ensure clean slate between test runs?

**Decision**: Use Go's `t.TempDir()` for automatic cleanup + explicit pre-test cleanup phase. Generate fresh SQLite databases and storage directories per test.

**Rationale**:
- `t.TempDir()` automatically cleans up on test completion (pass or fail)
- Ensures no state leakage between tests (deterministic results)
- Parallel tests get isolated directories (no conflicts)
- Pre-test cleanup handles orphaned data from interrupted test runs

**Alternatives Considered**:
1. **Persistent test databases** - Rejected because violates clean slate requirement from clarifications
2. **Docker volumes** - Rejected because slower and unnecessary for local testing
3. **Manual cleanup scripts** - Rejected because easy to forget and error-prone

**Implementation Notes**:
- Setup utility in `tests/interop/lib/setup.go`
- Function signature:
  ```go
  func SetupTestEnv(t *testing.T) (goDir, tsDir string, cleanup func())
  ```
- Each test gets:
  - Temporary directory for Go implementation (DB, storage, config)
  - Temporary directory for TypeScript implementation (DB, storage, config)
  - Unique ports allocated from range 20000-30000
  - Cleanup function that stops servers and removes temp dirs

---

## Decision 6: Parallel Test Execution Strategy

**Question**: Which tests can safely run in parallel and which require sequential execution?

**Decision**: Run all tests in parallel except those that use shared resources (network ports, global state). Use port allocation to avoid conflicts.

**Rationale**:
- Parallel execution critical for <5 min runtime goal (41+ functional requirements)
- Go's `t.Parallel()` makes parallel execution explicit and safe
- Port allocation prevents conflicts (each test gets unique ports)
- Temporary directories prevent filesystem conflicts

**Tests That Can Run in Parallel** (most tests):
- CLI command tests (isolated temp dirs)
- Cryptographic interop tests (stateless)
- HTTP API tests with unique ports
- Merkle proof validation tests (stateless)

**Tests That Must Run Sequentially**:
- Tests that validate port conflict handling
- Tests that intentionally test concurrent registration (within a single test)
- Teardown/cleanup validation tests

**Implementation Notes**:
- Mark parallel tests with `t.Parallel()` after setup
- Port allocator in `tests/interop/lib/ports.go`:
  ```go
  func AllocatePort(t *testing.T) int // Returns unique port, registers cleanup
  ```
- Port range: 20000-30000 (10,000 ports available)

---

## Decision 7: Error Scenario Testing Strategy

**Question**: How to test that both implementations handle errors identically (HTTP status codes, error messages)?

**Decision**: Create negative test cases that intentionally send invalid inputs (malformed CBOR, invalid signatures, missing headers) and compare error responses. Use error code categories, not exact message text.

**Rationale**:
- Error messages may differ in wording but should have same HTTP status codes
- Error categories (e.g., "invalid_signature", "not_found") are more stable than exact messages
- RFC compliance includes proper error handling per HTTP semantics
- Negative tests are critical for security validation

**Error Scenarios to Test** (minimum 15 per FR-038):
1. Invalid CBOR encoding → 400 Bad Request
2. Missing Content-Type header → 400 Bad Request
3. Wrong Content-Type (text/plain instead of application/cose) → 400 Bad Request
4. Corrupted COSE Sign1 signature → 400 Bad Request
5. Missing CWT claims (issuer, subject) → 400 Bad Request
6. Non-existent entry ID → 404 Not Found
7. Invalid entry ID format (non-integer) → 400 Bad Request
8. Malformed JWK public key → 400 Bad Request
9. Mismatched key for verification → 400 Bad Request (or 401)
10. Tampered Merkle proof → 400 Bad Request
11. Tree size mismatch for proof → 400 Bad Request
12. Server not initialized → 500 Internal Server Error
13. Database connection failure → 500 Internal Server Error
14. Large payload exceeding limits → 413 Payload Too Large
15. Concurrent writes (if not supported) → 409 Conflict

**Implementation Notes**:
- Error test utility in `tests/interop/lib/errors.go`
- Compare HTTP status codes (exact match required)
- Compare error response structure (expect snake_case fields: `error_code`, `error_message`, `details`)
- Do NOT compare exact error message text (too brittle)

---

## Decision 8: Test Report Format

**Question**: What format should test reports use to enable stakeholder review and CI integration?

**Decision**: Use Go's standard test output with JSON formatting for CI, plus generate human-readable Markdown summary with test details.

**Rationale**:
- Go test JSON output integrates with GitHub Actions, GitLab CI, Jenkins
- Markdown summary provides stakeholder-friendly format for manual review
- JSON format enables programmatic analysis (e.g., trend tracking)
- Standard Go test output is familiar to developers

**Report Components**:
1. **Go Test JSON Output** (`-json` flag) for CI parsing
2. **Markdown Summary** with:
   - Executive summary (pass/fail counts, duration)
   - Failed test details (expected vs actual, diff)
   - RFC compliance violations with RFC section references
   - Performance metrics (test duration, throughput)
3. **Detailed Logs** (verbose test output) for debugging

**Implementation Notes**:
- Report generator in `tests/interop/lib/report.go`
- Markdown template includes:
  ```markdown
  # Cross-Implementation Test Report

  **Date**: [timestamp]
  **Go Implementation**: [version/commit]
  **TypeScript Implementation**: [version/commit]
  **Duration**: [time]

  ## Summary
  - Total Tests: [count]
  - Passed: [count] ✅
  - Failed: [count] ❌
  - Skipped: [count] ⏭

  ## Failed Tests
  [Details with expected vs actual]

  ## RFC Compliance Violations
  [List with RFC section references]

  ## Performance
  [Test duration statistics]
  ```

---

## Decision 9: Merkle Proof Cross-Validation Strategy

**Question**: How to validate that Merkle proofs generated by one implementation can be verified by the other?

**Decision**: Generate proofs using each implementation's server API, then verify those proofs using the other implementation's verification functions (Go crypto libs for Go, TypeScript crypto libs for TS).

**Rationale**:
- Tests the complete proof lifecycle: generation (server) → serialization → deserialization → verification (client)
- Validates that proof formats are compatible (matching RFC 6962)
- Aligns with Principle VIII (Go tlog as canonical reference for proof structure)
- Enables detection of serialization bugs (e.g., big-endian vs little-endian)

**Test Scenarios** (minimum 30 proofs per FR-023):
- Inclusion proofs: tree sizes 1, 2, 3, 5, 10, 100, 256, 1000
- Entry positions: first, middle, last in tree
- Consistency proofs: tree growth from (1→2, 1→10, 10→20, 100→200)
- Edge cases: empty tree, single-element tree, full binary tree

**Implementation Notes**:
- Proof validator in `tests/interop/lib/proof_validator.go`
- Will use Go's `golang.org/x/mod/sumdb/tlog` for reference proof generation
- TypeScript proofs validated against Go-generated expected results
- Go proofs validated against TypeScript-generated expected results (if they pass Go validation)

---

## Decision 10: CLI Output Format for Comparison

**Question**: Should CLI tools output JSON by default, or should tests parse human-readable output?

**Decision**: CLI tools MUST support `--format=json` flag for structured output. Tests will use JSON format for comparison. Human-readable output tested separately for UX consistency.

**Rationale**:
- JSON output is deterministic and easy to parse programmatically
- Enables deep comparison of structured data (not fragile string matching)
- Human-readable output can vary (alignment, whitespace) without semantic difference
- Aligns with API-first architecture principle (CLI is a client)

**CLI Output Format** (snake_case per user requirement):
```json
{
  "command": "statement sign",
  "success": true,
  "result": {
    "statement_hash": "a1b2c3d4...",
    "signature": "9f8e7d...",
    "algorithm": "ES256"
  },
  "metadata": {
    "duration_ms": 42,
    "timestamp": "2025-10-12T15:30:00Z"
  }
}
```

**Error Output Format**:
```json
{
  "command": "statement verify",
  "success": false,
  "error": {
    "error_code": "invalid_signature",
    "error_message": "Signature verification failed",
    "details": {
      "expected_algorithm": "ES256",
      "actual_algorithm": "ES384"
    }
  }
}
```

**Implementation Notes**:
- Go CLI: Use `--format=json` flag (add if not present)
- TypeScript CLI: Use `--format=json` flag (add if not present)
- Fallback: If CLI doesn't support JSON yet, use regex parsing with clear TODOs

---

## Summary of Research Decisions

| Decision | Impact | Implementation Location |
|----------|--------|------------------------|
| Go test orchestration | Primary test framework | `tests/interop/*_test.go` |
| RFC test vectors from Go | Authoritative reference data | `tests/interop/fixtures/rfc-vectors/` |
| Semantic JSON comparison | Robust API response validation | `tests/interop/lib/compare.go` |
| CLI invocation via os/exec | Cross-language CLI testing | `tests/interop/lib/cli.go` |
| t.TempDir() cleanup | Deterministic test isolation | `tests/interop/lib/setup.go` |
| Parallel execution with port allocation | <5 min runtime | All `*_test.go` files with `t.Parallel()` |
| Error code comparison | Robust error validation | `tests/interop/lib/errors.go` |
| Markdown + JSON reports | Stakeholder & CI integration | `tests/interop/lib/report.go` |
| Cross-validation of Merkle proofs | Principle VIII compliance | `tests/interop/merkle/*_test.go` |
| JSON CLI output format | Structured comparison | CLI flag `--format=json` |

---

## Open Questions / Future Research

**None** - All technical decisions required for Phase 1 (Design) are complete.

---

## References

- **RFC 9052**: COSE (CBOR Object Signing and Encryption)
- **RFC 6962**: Certificate Transparency (Merkle tree structure)
- **RFC 8392**: CBOR Web Token (CWT)
- **RFC 9597**: CWT Claims in COSE Headers
- **RFC 7638**: JWK Thumbprint
- **RFC 8259**: JSON specification (semantic equivalence)
- **C2SP tlog-tiles**: Tile-based transparency log format
- **golang.org/x/mod/sumdb/tlog**: Go transparency log package (canonical reference)
- **Go testing package**: https://pkg.go.dev/testing
- **Constitution Principle VIII**: Go Interoperability as Source of Truth
