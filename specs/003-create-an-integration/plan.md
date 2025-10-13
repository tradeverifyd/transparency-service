# Implementation Plan: Cross-Implementation Integration Test Suite

**Branch**: `003-create-an-integration` | **Date**: 2025-10-12 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-create-an-integration/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Create a comprehensive integration test suite that validates 100% API compatibility, cryptographic interoperability, and user experience parity between the Go and TypeScript SCITT implementations. The test suite will validate both implementations against external RFC specifications (9052, 6962, 8392, 9597, 7638, C2SP tlog-tiles) as the authoritative reference, ensuring standards compliance rather than implementation lock-in. Tests will execute in both CI/CD pipelines and local developer environments with clean-slate test data management for deterministic results.

## Technical Context

**Language/Version**:
- Go 1.24 (for Go implementation testing and test suite orchestration)
- Bun latest (for TypeScript implementation testing)
- Shell scripting (bash) for cross-platform test coordination

**Primary Dependencies**:
- Go: `testing` package (standard library), `httptest` for server testing
- Bun: `bun:test` for TypeScript test execution
- Shell: `jq` for JSON processing, `diff` for response comparison
- Both implementations: existing CLI tools and HTTP servers

**Storage**:
- SQLite databases (temporary, cleaned per test run)
- Filesystem storage (temporary directories, cleaned per test run)
- Test fixtures: JSON payloads, PEM/JWK key files, CBOR statements

**Testing**:
- Go test framework for test orchestration
- Shell scripts for CLI invocation and output comparison
- HTTP clients for API testing (Go `net/http`, TypeScript fetch/axios)
- Test data generators for creating COSE Sign1 statements, keys, statements

**Target Platform**:
- CI/CD: GitHub Actions (Linux Ubuntu latest)
- Local: macOS, Linux developer machines
- Docker containers for isolated CI environments

**Project Type**: Integration test suite (cross-implementation validation)

**Performance Goals**:
- Complete test suite execution <5 minutes in CI
- Individual test isolation <100ms overhead per test
- Parallel test execution support (up to 10 concurrent tests)
- Test report generation <10 seconds

**Constraints**:
- Must not modify existing implementation code
- Test data must be RFC-compliant (validated against specs)
- Hex encoding for identifiers (not base64url per user requirement)
- snake_case for REST/JSON interfaces (not camelCase per user requirement)
- Clean slate per test run (no persistent state)
- Both implementations must be built/available before tests run

**Scale/Scope**:
- 41 functional requirements to validate
- Minimum 20 CLI command variations
- Minimum 10 HTTP endpoint variations
- Minimum 50 sign/verify combinations per direction
- Minimum 30 Merkle proof cross-verifications
- Minimum 15 error scenarios
- 100+ test statements for query testing
- 6 prioritized user story categories

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle Compliance Analysis

✅ **I. Transparency by Design**: Test suite will produce detailed execution logs showing all operations, decisions, and comparisons. Test reports will include data provenance (which implementation generated each artifact).

✅ **II. Verifiable Audit Trails**: All test executions will be logged with timestamps, test names, input data, expected vs actual outcomes. Test artifacts (statements, receipts, checkpoints) will be preserved in test reports.

✅ **III. Test-First Development (NON-NEGOTIABLE)**: This feature IS the test suite. The test suite will be developed in phases with clear test scenarios documented before implementation. User approval of test design (this plan) required before implementation.

✅ **IV. API-First Architecture**: Test suite will validate OpenAPI/REST compliance for HTTP endpoints. Test contracts will be defined in `/contracts/` before implementation.

✅ **V. Observability and Monitoring**: Test suite will emit structured logs (JSON format), include timing metrics, report per-test and aggregate statistics. CI integration will report test results as GitHub Actions artifacts.

✅ **VI. Data Integrity and Versioning**: Test fixtures will be versioned, schema validation will verify RFC compliance. Test data will include checksums for integrity verification.

✅ **VII. Simplicity and Maintainability**: Test suite will use standard Go testing patterns, shell scripts for CLI orchestration. No custom test frameworks - standard tools only. Complexity justified below.

✅ **VIII. Go Interoperability as Source of Truth (NON-NEGOTIABLE)**: **CRITICAL FOR THIS FEATURE**. This test suite validates Principle VIII compliance by:
- Using Go tlog as canonical reference for Merkle operations
- Validating TypeScript Merkle proofs against Go-generated proofs
- Using RFC compliance as implemented by Go as the authoritative interpretation
- Generating test vectors from Go implementation
- Ensuring 100% interoperability test compliance
- Validating that TypeScript conforms to Go's RFC interpretation

### Quality Gates

**Before Phase 0 (Research)**:
- ✅ Feature spec complete with 41 functional requirements
- ✅ Clarifications resolved (execution environment, baseline, data lifecycle)
- ✅ Constitution compliance verified

**Before Phase 1 (Design)**:
- ⏳ Research complete (RFC interpretation, test patterns, Go test vectors)
- ⏳ Test framework decisions documented
- ⏳ Data model for test results defined

**Before Phase 2 (Implementation)**:
- ⏳ API contracts defined (test report format, test data schemas)
- ⏳ Quickstart guide written (how to run tests locally)
- ⏳ Test scenarios approved by stakeholders

### Complexity Justification

| Complexity | Why Needed | Simpler Alternative Rejected Because |
|------------|------------|-------------------------------------|
| Cross-language test orchestration | Must validate Go and TypeScript implementations independently and cross-verify results | Single-language test suite cannot validate cross-implementation compatibility |
| Shell script coordination | Need to invoke CLI tools from both implementations and capture outputs for comparison | Pure Go or TypeScript test suite cannot invoke other language's CLI tools cleanly |
| Temporary environment management | Each test needs isolated databases, storage, and ports to avoid interference | Shared test environment leads to flaky tests and debugging nightmares |
| RFC test vector generation | Need canonical reference data to validate both implementations against specs | Using either implementation as reference creates circular validation |

## Project Structure

### Documentation (this feature)

```
specs/003-create-an-integration/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0: RFC interpretation, test patterns, Go reference
├── data-model.md        # Phase 1: Test result schemas, fixture formats
├── quickstart.md        # Phase 1: How to run tests locally
├── contracts/           # Phase 1: Test report schemas, fixture formats
│   ├── test-report.schema.json
│   ├── test-fixtures.schema.json
│   └── api-comparison.schema.json
└── tasks.md             # Phase 2: Implementation tasks (NOT created by /speckit.plan)
```

### Source Code (repository root)

```
tests/
├── interop/                    # Cross-implementation integration tests
│   ├── README.md              # Test suite overview and running instructions
│   ├── go.mod                 # Go test dependencies
│   ├── main_test.go           # Test orchestration entry point
│   │
│   ├── fixtures/              # Test data and expected outputs
│   │   ├── keys/              # Test keypairs (PEM, JWK)
│   │   ├── payloads/          # Test payloads (JSON)
│   │   ├── statements/        # Pre-generated COSE Sign1 statements
│   │   └── rfc-vectors/       # RFC test vectors from Go tlog
│   │
│   ├── lib/                   # Test utilities
│   │   ├── setup.go           # Test environment setup/teardown
│   │   ├── compare.go         # Response comparison utilities
│   │   ├── rfc_validate.go    # RFC compliance validation
│   │   └── report.go          # Test report generation
│   │
│   ├── cli/                   # CLI compatibility tests
│   │   ├── init_test.go       # Test 'init' command parity
│   │   ├── statement_test.go  # Test 'statement' command parity
│   │   └── serve_test.go      # Test 'serve' command parity
│   │
│   ├── http/                  # HTTP API compatibility tests
│   │   ├── entries_test.go    # POST /entries, GET /entries/{id}
│   │   ├── checkpoint_test.go # GET /checkpoint
│   │   ├── config_test.go     # GET /.well-known/transparency-configuration
│   │   └── health_test.go     # GET /health
│   │
│   ├── crypto/                # Cryptographic interoperability tests
│   │   ├── sign_verify_test.go     # Cross-impl sign/verify
│   │   ├── hash_envelope_test.go   # Hash envelope compatibility
│   │   └── jwk_thumbprint_test.go  # JWK thumbprint consistency
│   │
│   ├── merkle/                # Merkle tree proof tests
│   │   ├── inclusion_test.go       # Inclusion proof cross-validation
│   │   ├── consistency_test.go     # Consistency proof cross-validation
│   │   └── root_hash_test.go       # Tree root computation consistency
│   │
│   ├── e2e/                   # End-to-end workflow tests
│   │   ├── pure_go_test.go         # Go CLI + Go Server
│   │   ├── pure_ts_test.go         # TS CLI + TS Server
│   │   ├── cross_go_ts_test.go     # Go CLI + TS Server
│   │   └── cross_ts_go_test.go     # TS CLI + Go Server
│   │
│   └── scripts/               # Test orchestration scripts
│       ├── setup_env.sh           # Create isolated test environment
│       ├── build_impls.sh         # Build both implementations
│       ├── run_cli.sh             # Invoke CLI with logging
│       └── compare_outputs.sh     # Compare JSON/CBOR outputs
│
└── contract/                   # Existing contract tests (untouched)
    └── [existing structure]
```

**Structure Decision**:

The test suite is organized as a **standalone Go test package** (`tests/interop/`) that orchestrates tests across both implementations. This structure was chosen because:

1. **Independence**: Test suite is isolated from implementation code, preventing accidental coupling
2. **Orchestration**: Go's `testing` package provides excellent test organization, parallel execution, and CI integration
3. **Cross-language**: Shell scripts bridge Go test orchestration with TypeScript CLI/server invocation
4. **Fixtures**: Centralized test data management with RFC-validated vectors from Go tlog
5. **Modularity**: Tests organized by capability (CLI, HTTP, crypto, Merkle, e2e) for maintainability
6. **Reporting**: Go test output formats (JSON, verbose) integrate with CI systems

The structure avoids placing tests inside either implementation's directory to maintain neutrality and prevent bias toward either Go or TypeScript as the reference.

## Complexity Tracking

*Justifications for constitutional principle deviations*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Shell script coordination | Must invoke CLI tools from both implementations (Go binary, Bun CLI) and coordinate server lifecycle | Pure Go test suite cannot cleanly invoke Bun/TypeScript commands; pure TypeScript cannot invoke Go binaries without complex build tooling |
| Temporary port allocation | Tests need isolated HTTP servers to avoid port conflicts during parallel execution | Sequential tests would be too slow (5+ min runtime requirement); shared server would create test interdependencies |
| CBOR/COSE parsing in tests | Test suite needs to decode COSE Sign1 structures to validate field-level compatibility | String comparison of CBOR bytes would miss semantic equivalence (e.g., different map orderings that are valid per CBOR spec) |
| RFC test vector generation | Need authoritative test data to validate BOTH implementations against specs, not each other | Using either Go or TypeScript as reference would create circular validation (implementation validates itself) |

**Note**: These complexities are inherent to cross-implementation validation and cannot be simplified without compromising test integrity or violating Principle VIII (Go as canonical reference for Merkle/crypto operations).
