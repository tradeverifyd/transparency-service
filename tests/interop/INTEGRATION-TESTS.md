# SCITT Cross-Implementation Integration Test Suite

## Overview

This test suite validates compatibility between Go and TypeScript implementations of the SCITT (Supply Chain Integrity, Transparency, and Trust) transparency service. The tests ensure both implementations expose identical APIs, produce compatible outputs, and maintain RFC compliance.

## Status: MVP Complete ✅

**38 of 38 MVP tasks implemented (100%)**

- ✅ Phase 1: Setup (6 tasks)
- ✅ Phase 2: Foundational Infrastructure (18 tasks)
- ✅ Phase 3: HTTP API Tests (7 tasks)
- ✅ Phase 4: CLI Tests (7 tasks)

## Quick Start

```bash
# Build both implementations
cd tests/interop
./scripts/build_impls.sh

# Run all tests
go test -v ./...

# Run with parallelism
go test -v -parallel 10 ./...

# Run specific test suite
go test -v ./http/...  # HTTP API tests
go test -v ./cli/...   # CLI tests
```

## Test Coverage

### Phase 3: HTTP API Tests (7 tasks) ✅

Tests all HTTP endpoints for cross-implementation compatibility:

- **`http/config_test.go`** - Transparency configuration
  - Validates GET `/.well-known/transparency-configuration`
  - Checks algorithms, endpoints, origins fields
  - Enforces snake_case convention

- **`http/entries_test.go`** - Statement registration
  - POST `/entries` with COSE Sign1 statements
  - GET `/entries/{id}` receipt retrieval
  - Concurrent registration testing

- **`http/checkpoint_test.go`** - Signed tree heads
  - GET `/checkpoint` validation
  - RFC 6962 compliance checking
  - Tree growth verification

- **`http/health_test.go`** - Health monitoring
  - GET `/health` availability
  - Response time validation
  - Reliability under load

- **`http/query_test.go`** - Statement queries
  - Query by issuer, subject, content_type
  - Pagination and ordering
  - Empty result handling

- **`http/errors_test.go`** - Error scenarios
  - 10+ error cases (400, 404, 405, 415)
  - Error structure validation
  - Concurrent error handling

**Functional Requirements:** FR-011 through FR-016, FR-038, FR-039

### Phase 4: CLI Tests (7 tasks) ✅

Tests command-line tools for identical behavior:

- **`cli/init_test.go`** - Initialization
  - Directory structure creation
  - Config file generation
  - Keypair generation
  - Idempotency validation

- **`cli/statement_test.go`** - Statement operations
  - Sign, verify, hash commands
  - Cross-implementation verification
  - JSON output validation

- **`cli/serve_test.go`** - Server operations
  - Server startup
  - Custom port configuration
  - Origin configuration

- **`cli/errors_test.go`** - Error handling
  - 8+ error scenarios
  - Exit code consistency
  - Error message quality

**Functional Requirements:** FR-006 through FR-010, FR-040, FR-041

## Architecture

### Test Library (`lib/`)

Core testing infrastructure:

- **`types.go`** - Type definitions with snake_case JSON tags
- **`setup.go`** - Test environment isolation (temp dirs, ports)
- **`ports.go`** - Port allocator (20000-30000 range)
- **`cli.go`** - CLI invocation for both implementations
- **`compare.go`** - Semantic JSON comparison
- **`rfc_validate.go`** - RFC 9052, 6962, 8392, 7638 validation
- **`report.go`** - Test report generation (JSON/Markdown)

### Test Fixtures

- **`tools/generate_keypair.go`** - ES256 keypair generator
- **`fixtures/keys/`** - 5 test keypairs with RFC 7638 thumbprints
- **`fixtures/payloads/`** - Test payloads (small, medium, large)

### Shell Scripts

- **`scripts/setup_env.sh`** - Environment configuration
- **`scripts/build_impls.sh`** - Implementation verification
- **`scripts/run_cli.sh`** - CLI wrapper with logging

## Test Isolation

Each test uses `lib.SetupTestEnv(t)` which provides:

```go
goDir, tsDir, cleanup := lib.SetupTestEnv(t)
defer cleanup()

// Creates isolated temp directories
// Allocates unique ports
// Automatic cleanup on test completion
```

### Port Allocation

```go
port := lib.GlobalPortAllocator.AllocatePort(t)
// Returns unique port in 20000-30000 range
// Verifies port is available
// Automatic cleanup on test completion
```

### Comparison Strategy

```go
result := lib.CompareOutputs(goResult, tsResult)
// Semantic deep equality (not string comparison)
// Handles JSON key ordering differences
// Tolerates numeric precision differences
// Reports detailed differences with JSON paths
// Verdict: identical, equivalent, divergent, both_invalid
```

## Project Conventions

### snake_case

All JSON field names use snake_case:
```json
{
  "entry_id": "abc123",
  "statement_hash": "def456",
  "tree_size": 42
}
```

### Hex Encoding

All identifiers use lowercase hex:
```json
{
  "entry_id": "a1b2c3d4",
  "root_hash": "e5f6a7b8"
}
```

### RFC Compliance

Tests validate:
- **RFC 9052** - COSE Sign1
- **RFC 6962** - Merkle trees
- **RFC 8392** - CWT
- **RFC 7638** - JWK Thumbprint
- **C2SP tlog-tiles** - Tile naming

## Environment Variables

Override default CLI paths:

```bash
export SCITT_GO_CLI=/path/to/go/scitt
export SCITT_TS_CLI="bun run /path/to/typescript/cli.ts"
```

## CI Integration

### GitHub Actions

```yaml
- name: Run Integration Tests
  run: |
    cd tests/interop
    ./scripts/build_impls.sh
    go test -v -parallel 10 -json ./... > results.json

- name: Upload Results
  uses: actions/upload-artifact@v3
  with:
    name: test-results
    path: tests/interop/results.json
```

## Troubleshooting

### Tests Skip

Some tests require running servers and will skip if unavailable:
- Ensure implementations are built: `./scripts/build_impls.sh`
- Check environment variables are set correctly
- Verify ports 20000-30000 are available

### Port Allocation Failures

If port allocation fails:
- Reduce parallelism: `go test -parallel 1 ./...`
- Check no services using ports 20000-30000
- Verify firewall allows localhost connections

### Fixture Errors

If fixtures are missing:
- Run from `tests/interop/` directory
- Regenerate: `cd tools && go run generate_keypair.go`

## Future Phases

Additional test phases available for implementation:

- **Phase 5**: Cryptographic Interoperability (5 tasks)
- **Phase 6**: Merkle Tree Proofs (6 tasks)
- **Phase 7**: Query Compatibility (6 tasks)
- **Phase 8**: Receipt Compatibility (7 tasks)
- **Phase 9**: End-to-End Workflows (5 tasks)
- **Phase 10**: Test Validation (3 tasks)
- **Phase 11**: Polish & Documentation (7 tasks)

See `specs/003-create-an-integration/tasks.md` for details.

## Contributing

When adding new tests:

1. Use `lib.SetupTestEnv(t)` for isolation
2. Allocate ports with `lib.GlobalPortAllocator.AllocatePort(t)`
3. Compare with `lib.CompareOutputs()` not string comparison
4. Validate RFC compliance with `lib.ValidateRFCCompliance()`
5. Follow snake_case and hex encoding conventions
6. Update documentation with new coverage

## Documentation

- `INTEGRATION-TESTS.md` - This file (comprehensive guide)
- `README.md` - Original Go tlog interoperability documentation
- `tools/README.md` - Fixture generation documentation
- `specs/003-create-an-integration/` - Complete specification

## License

See repository LICENSE file.
