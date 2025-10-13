# transparency-service Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-10-12

## Active Technologies
- Bun (latest stable) with TypeScript (001-build-a-bun)
- Go 1.24 (for test orchestration and Go implementation)
- Integration test suite (003-create-an-integration)

## Project Structure
```
src/                          # TypeScript implementation
tests/
  interop/                    # Cross-implementation integration tests
    cli/                      # CLI compatibility tests
    http/                     # HTTP API compatibility tests
    crypto/                   # Cryptographic interoperability tests
    merkle/                   # Merkle tree proof tests
    e2e/                      # End-to-end workflow tests
    fixtures/                 # Test data and RFC vectors
    lib/                      # Test utilities
    scripts/                  # Test orchestration scripts
  contract/                   # Existing contract tests
```

## Commands

### TypeScript/Bun
```bash
bun install                   # Install dependencies
bun run cli --help            # Run TypeScript CLI
```

### Integration Tests
```bash
cd tests/interop
go test -v ./...              # Run all integration tests
go test -v -parallel 10 ./... # Run tests in parallel
go test -v -run TestCLI ./cli # Run specific test category
go test -json ./... > results.json  # Generate JSON report
```

## Code Style

### TypeScript
Follow standard Bun/TypeScript conventions

### JSON/REST APIs (CRITICAL)
- **snake_case**: Use snake_case for all JSON field names (NOT camelCase)
- **hex encoding**: Use lowercase hex encoding for identifiers (NOT base64url)
- Example: `{"entry_id": "a1b2c3d4", "statement_hash": "9f8e7d6c"}`

### Go Test Code
Follow standard Go testing conventions with `t.Parallel()` for independent tests

## Integration Test Requirements

### Test Orchestration
- Use Go's `testing` package as primary orchestrator
- Use shell scripts for CLI invocation across implementations
- Clean slate per test: `t.TempDir()` for automatic cleanup
- Parallel execution: unique port allocation (20000-30000)

### RFC Compliance
- Go implementation is canonical reference (Constitution Principle VIII)
- Validate both implementations against RFC specifications:
  - RFC 9052 (COSE)
  - RFC 6962 (Merkle trees)
  - RFC 8392 (CWT)
  - RFC 9597 (CWT in COSE)
  - RFC 7638 (JWK Thumbprint)

### Test Fixtures
- Fixtures in `tests/interop/fixtures/` as JSON files
- snake_case field naming in all fixtures
- hex encoding for binary data (hashes, identifiers)
- Versioned fixtures for regeneration tracking

### Test Data Format
All test results and reports use:
- snake_case fields: `test_id`, `result_id`, `statement_hash`
- hex-encoded identifiers: lowercase, no hyphens
- ISO 8601 timestamps
- Semantic JSON comparison (not string comparison)

## Recent Changes
- 001-build-a-bun: Added Bun (latest stable) with TypeScript
- 003-create-an-integration: Added cross-implementation integration test suite with RFC compliance validation

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
