# Data Model: Cross-Implementation Integration Test Suite

**Feature**: Cross-Implementation Integration Test Suite
**Phase**: 1 - Design
**Date**: 2025-10-12

## Purpose

This document defines the data structures used by the cross-implementation integration test suite, including test fixtures, test results, comparison reports, and RFC test vectors.

---

## Core Entities

### 1. Test Execution Context

Represents the environment and configuration for a single test run.

**Fields**:
```go
type TestExecutionContext struct {
    // Identification
    test_id           string    // Unique test execution ID (UUID)
    test_name         string    // Human-readable test name
    timestamp         string    // ISO 8601 timestamp

    // Implementation metadata
    go_version        string    // Go implementation version/commit
    ts_version        string    // TypeScript implementation version/commit
    go_binary_path    string    // Path to Go CLI binary
    ts_cli_command    []string  // Command to invoke TypeScript CLI

    // Environment
    go_work_dir       string    // Temporary directory for Go
    ts_work_dir       string    // Temporary directory for TypeScript
    go_server_port    int       // Allocated port for Go server
    ts_server_port    int       // Allocated port for TypeScript server

    // Test configuration
    timeout_seconds   int       // Maximum test duration
    parallel_enabled  bool      // Whether test can run in parallel
    cleanup_on_pass   bool      // Clean temp dirs after success
}
```

**Validation Rules**:
- `test_id` must be unique per test run (UUID v4)
- `timestamp` must be valid ISO 8601 format
- Ports must be in range 20000-30000
- Work dirs must exist and be writable
- Timeout must be > 0 and < 600 seconds

**State Transitions**:
1. **Created** → test context initialized
2. **Setup** → temporary directories and ports allocated
3. **Running** → test execution in progress
4. **Cleanup** → temporary resources being released
5. **Complete** → test finished (pass/fail)

---

### 2. Test Fixture

Represents test input data (keys, payloads, statements) used across test runs.

**Fields**:
```go
type TestFixture struct {
    // Identification
    fixture_id        string    // Unique fixture identifier
    fixture_type      string    // Type: "keypair", "payload", "statement", "rfc_vector"
    version           string    // Fixture version (for regeneration tracking)

    // Cryptographic fixtures
    keypair           *KeypairFixture    // If fixture_type == "keypair"

    // Payload fixtures
    payload           *PayloadFixture    // If fixture_type == "payload"

    // Statement fixtures
    statement         *StatementFixture  // If fixture_type == "statement"

    // RFC test vectors
    rfc_vector        *RFCVectorFixture  // If fixture_type == "rfc_vector"

    // Metadata
    description       string    // Human-readable description
    tags              []string  // Tags for filtering (e.g., "crypto", "merkle", "negative")
    created_at        string    // ISO 8601 timestamp
    created_by        string    // Tool/implementation that generated fixture
}
```

#### 2.1 KeypairFixture

```go
type KeypairFixture struct {
    // Keys
    private_key_pem   string    // PEM-encoded private key (ECDSA P-256)
    public_key_jwk    string    // JWK-encoded public key (JSON string)
    algorithm         string    // "ES256"

    // Computed values
    jwk_thumbprint    string    // RFC 7638 thumbprint (hex-encoded)
    key_id            string    // Key identifier (optional)
}
```

#### 2.2 PayloadFixture

```go
type PayloadFixture struct {
    // Payload data
    content           []byte    // Raw payload bytes
    content_type      string    // MIME type (e.g., "application/json")
    encoding          string    // "raw", "base64", or "hex"

    // Expected hash
    sha256_hash       string    // SHA-256 hash (hex-encoded)
}
```

#### 2.3 StatementFixture

```go
type StatementFixture struct {
    // COSE Sign1 components
    cose_sign1_cbor   []byte    // Encoded COSE Sign1 structure

    // Metadata
    issuer            string    // CWT claim: iss
    subject           string    // CWT claim: sub
    content_type      string    // Protected header: cty
    algorithm         string    // Protected header: alg

    // Computed values
    statement_hash    string    // SHA-256 of COSE Sign1 (hex-encoded)
    payload_hash      string    // SHA-256 of payload (hex-encoded)

    // Signing key reference
    keypair_fixture_id string   // Reference to KeypairFixture used
}
```

#### 2.4 RFCVectorFixture

```go
type RFCVectorFixture struct {
    // RFC reference
    rfc_number        string    // e.g., "RFC 6962", "RFC 9052"
    section           string    // RFC section (e.g., "2.1.1")
    test_case_name    string    // Descriptive name

    // Test data
    input_data        map[string]interface{}  // Test inputs (JSON)
    expected_output   map[string]interface{}  // Expected outputs (JSON)

    // Generation metadata
    generated_by      string    // "go-tlog", "go-cose", etc.
    generation_date   string    // ISO 8601 timestamp
}
```

**Validation Rules**:
- `fixture_id` must be unique across all fixtures
- `fixture_type` must be one of: "keypair", "payload", "statement", "rfc_vector"
- Hex-encoded values must be valid hex (lowercase)
- PEM keys must have valid header/footer
- JWK must be valid JSON and conform to RFC 7517
- CBOR must be decodable

---

### 3. Test Result

Represents the outcome of a single test case.

**Fields**:
```go
type TestResult struct {
    // Identification
    result_id         string    // Unique result ID (UUID)
    test_name         string    // Test function name
    execution_context_id string // Reference to TestExecutionContext

    // Outcome
    status            string    // "pass", "fail", "skip", "error"
    duration_ms       int       // Test duration in milliseconds

    // Results per implementation
    go_result         *ImplementationResult
    ts_result         *ImplementationResult

    // Comparison
    comparison        *ComparisonResult

    // Diagnostics
    error_message     string    // If status == "error" or "fail"
    stack_trace       string    // Error stack trace (if applicable)
    logs              []string  // Test execution logs

    // Metadata
    timestamp         string    // ISO 8601 timestamp
    rfc_violations    []RFCViolation  // Any RFC compliance violations detected
}
```

#### 3.1 ImplementationResult

```go
type ImplementationResult struct {
    // Implementation
    implementation    string    // "go" or "typescript"

    // Execution
    command           []string  // CLI command or API endpoint
    exit_code         int       // CLI exit code (0 = success)
    stdout            string    // Standard output
    stderr            string    // Standard error

    // Output
    output_format     string    // "json", "cbor", "text"
    output_data       interface{}  // Parsed output (if structured)
    output_raw        []byte    // Raw output bytes

    // Timing
    duration_ms       int       // Implementation-specific duration

    // Success
    success           bool      // Whether implementation succeeded
}
```

#### 3.2 ComparisonResult

```go
type ComparisonResult struct {
    // Equivalence
    outputs_equivalent bool      // Are outputs semantically equivalent?

    // Differences
    differences       []Difference

    // RFC compliance
    both_rfc_compliant bool     // Do both comply with RFCs?

    // Verdict
    verdict           string    // "identical", "equivalent", "divergent", "both_invalid"
    verdict_reason    string    // Explanation of verdict
}
```

#### 3.3 Difference

```go
type Difference struct {
    // Location
    field_path        string    // JSON path (e.g., "$.entry_id")

    // Values
    go_value          interface{}  // Value from Go implementation
    ts_value          interface{}  // Value from TypeScript implementation

    // Severity
    severity          string    // "critical", "major", "minor", "acceptable"

    // Explanation
    explanation       string    // Why values differ
    rfc_reference     string    // RFC section if relevant
}
```

#### 3.4 RFCViolation

```go
type RFCViolation struct {
    // RFC reference
    rfc_number        string    // e.g., "RFC 9052"
    section           string    // Section number
    requirement       string    // Specific requirement violated

    // Violation details
    implementation    string    // "go", "typescript", or "both"
    violation_type    string    // "missing_field", "invalid_value", "wrong_format", etc.
    description       string    // Human-readable violation description

    // Evidence
    actual_value      interface{}  // What was observed
    expected_value    interface{}  // What RFC requires
}
```

**Validation Rules**:
- `status` must be one of: "pass", "fail", "skip", "error"
- `duration_ms` must be >= 0
- If `status == "fail"`, must have at least one difference with severity != "acceptable"
- If `status == "pass"`, `outputs_equivalent` must be true
- RFC violations must reference valid RFC numbers and sections

---

### 4. Test Report

Aggregates multiple test results into a comprehensive report.

**Fields**:
```go
type TestReport struct {
    // Identification
    report_id         string    // Unique report ID (UUID)
    report_date       string    // ISO 8601 timestamp

    // Summary
    total_tests       int       // Total test count
    passed_tests      int       // Tests that passed
    failed_tests      int       // Tests that failed
    skipped_tests     int       // Tests that were skipped
    error_tests       int       // Tests that encountered errors

    // Duration
    total_duration_ms int       // Total execution time

    // Implementation versions
    go_version        string    // Go implementation version
    ts_version        string    // TypeScript implementation version

    // Results
    test_results      []TestResult  // All test results

    // Analysis
    summary_by_category map[string]CategorySummary  // Results grouped by test category
    rfc_compliance_summary RFCComplianceSummary
    performance_summary PerformanceSummary

    // Failures
    failed_test_details []FailedTestDetail
}
```

#### 4.1 CategorySummary

```go
type CategorySummary struct {
    category_name     string    // e.g., "CLI Compatibility", "HTTP API"
    total             int       // Tests in category
    passed            int       // Passed tests
    failed            int       // Failed tests
    pass_rate         float64   // Percentage (0-100)
}
```

#### 4.2 RFCComplianceSummary

```go
type RFCComplianceSummary struct {
    total_violations  int       // Total RFC violations found
    violations_by_rfc map[string]int  // Count per RFC
    critical_violations []RFCViolation  // High-severity violations
    go_violations     int       // Violations in Go implementation
    ts_violations     int       // Violations in TypeScript implementation
}
```

#### 4.3 PerformanceSummary

```go
type PerformanceSummary struct {
    avg_test_duration_ms   float64    // Average per-test duration
    slowest_tests          []SlowTest // Top 10 slowest tests
    parallelization_ratio  float64    // Actual vs theoretical speedup
}
```

#### 4.4 SlowTest

```go
type SlowTest struct {
    test_name         string    // Test name
    duration_ms       int       // Duration
    category          string    // Test category
}
```

#### 4.5 FailedTestDetail

```go
type FailedTestDetail struct {
    test_name         string    // Test name
    error_summary     string    // Brief error description
    expected_output   string    // What was expected
    actual_output     string    // What was observed
    diff              string    // Formatted diff (if applicable)
    rfc_violations    []RFCViolation
}
```

---

## Data Relationships

```
TestExecutionContext (1) ─── contains ───> (*) TestResult
TestResult (1) ─── references ───> (1) TestFixture
TestResult (*) ─── aggregated by ───> (1) TestReport
RFCVectorFixture (1) ─── validates ───> (*) TestResult
KeypairFixture (1) ─── used by ───> (*) StatementFixture
```

---

## File Formats

### JSON Schema for Test Fixtures

Location: `specs/003-create-an-integration/contracts/test-fixtures.schema.json`

All test fixtures stored as JSON files with hex-encoded binary data (per user requirement).

### JSON Schema for Test Reports

Location: `specs/003-create-an-integration/contracts/test-report.schema.json`

Test reports generated in JSON format with snake_case fields (per user requirement).

### Go Struct Tags

All Go structs use JSON struct tags with snake_case:

```go
type TestResult struct {
    ResultID string `json:"result_id"`
    TestName string `json:"test_name"`
    Status   string `json:"status"`
    // ...
}
```

---

## Data Validation

### Required Validations

1. **Hex Encoding**: All binary data (hashes, identifiers) must be lowercase hex
2. **snake_case**: All JSON field names must use snake_case
3. **ISO 8601**: All timestamps must be valid ISO 8601 format
4. **RFC Compliance**: All cryptographic artifacts must be RFC-compliant
5. **Uniqueness**: All IDs must be unique (UUIDs)

### Validation Functions

```go
// In tests/interop/lib/validate.go
func ValidateTestFixture(fixture *TestFixture) error
func ValidateTestResult(result *TestResult) error
func ValidateTestReport(report *TestReport) error
func ValidateRFCCompliance(data interface{}, rfcNumber string) error
```

---

## Storage

### Fixtures

- **Location**: `tests/interop/fixtures/`
- **Format**: JSON files with `.json` extension
- **Naming**: `{fixture_type}_{fixture_id}.json`
- **Example**: `keypair_alice.json`, `statement_001.json`

### Results

- **Location**: Temporary directory during test run
- **Format**: JSON (in-memory during execution)
- **Persistence**: Only on failure or when explicitly saved

### Reports

- **Location**: `tests/interop/reports/` (CI artifacts)
- **Format**: Markdown + JSON
- **Naming**: `test-report_{timestamp}.{md|json}`

---

## Sample Data

### Sample KeypairFixture

```json
{
  "fixture_id": "keypair_alice",
  "fixture_type": "keypair",
  "version": "1.0",
  "keypair": {
    "private_key_pem": "-----BEGIN EC PRIVATE KEY-----\n...\n-----END EC PRIVATE KEY-----",
    "public_key_jwk": "{\"kty\":\"EC\",\"crv\":\"P-256\",\"x\":\"...\",\"y\":\"...\"}",
    "algorithm": "ES256",
    "jwk_thumbprint": "a1b2c3d4e5f6...",
    "key_id": "alice-2025"
  },
  "description": "Test keypair for Alice (signer)",
  "tags": ["crypto", "positive"],
  "created_at": "2025-10-12T15:00:00Z",
  "created_by": "go-keygen-v1.24"
}
```

### Sample TestResult

```json
{
  "result_id": "550e8400-e29b-41d4-a716-446655440000",
  "test_name": "TestCLIStatementSign",
  "execution_context_id": "ctx_001",
  "status": "pass",
  "duration_ms": 42,
  "go_result": {
    "implementation": "go",
    "command": ["scitt", "statement", "sign", "--input", "payload.json"],
    "exit_code": 0,
    "stdout": "{\"statement_hash\":\"a1b2c3...\"}",
    "output_format": "json",
    "success": true
  },
  "ts_result": {
    "implementation": "typescript",
    "command": ["bun", "run", "cli", "statement", "sign", "--input", "payload.json"],
    "exit_code": 0,
    "stdout": "{\"statement_hash\":\"a1b2c3...\"}",
    "output_format": "json",
    "success": true
  },
  "comparison": {
    "outputs_equivalent": true,
    "differences": [],
    "both_rfc_compliant": true,
    "verdict": "identical",
    "verdict_reason": "Both implementations produced identical statement hash"
  },
  "timestamp": "2025-10-12T15:30:45Z",
  "rfc_violations": []
}
```

---

## Extensibility

### Adding New Test Categories

1. Add new category constant in `tests/interop/lib/categories.go`
2. Update `CategorySummary` aggregation logic
3. Add category-specific validation rules if needed

### Adding New Fixture Types

1. Define new fixture struct in `tests/interop/lib/fixtures.go`
2. Add fixture type to `TestFixture.fixture_type` enum
3. Implement fixture generation tool in `tests/interop/tools/`
4. Update JSON schema in `contracts/test-fixtures.schema.json`

### Adding New RFC Validations

1. Add RFC reference to `tests/interop/lib/rfc_refs.go`
2. Implement validation function in `tests/interop/lib/rfc_validate.go`
3. Add RFC to `RFCComplianceSummary.violations_by_rfc` map
4. Document RFC requirements in research.md

---

## Summary

This data model provides:

✅ **Structured test execution context** with isolated environments
✅ **Comprehensive test fixtures** (keys, payloads, statements, RFC vectors)
✅ **Detailed test results** with per-implementation and comparison data
✅ **Aggregated reports** with summaries, RFC compliance, and performance metrics
✅ **snake_case and hex encoding** as required by user
✅ **RFC compliance tracking** with violation details and references
✅ **Extensibility** for new test categories and fixture types

All data structures support the 41 functional requirements defined in the specification and enable comprehensive cross-implementation validation.
