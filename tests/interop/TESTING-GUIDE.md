# SCITT Integration Test Suite - Testing Guide

## Overview

This guide provides instructions for running the SCITT cross-implementation integration test suite and understanding test behaviors.

## Prerequisites

### Required Software

- **Go 1.22+**: For test orchestration
- **Bun**: For TypeScript implementation
- **Git**: For repository management

### Build Both Implementations

#### Build Go Implementation

```bash
cd scitt-golang
go build -o scitt ./cmd/scitt
```

#### Build TypeScript Implementation

```bash
cd scitt-typescript
bun install
```

## Running Tests

### Quick Start

```bash
cd tests/interop

# Run all tests
go test -v ./...

# Run with parallelism
go test -v -parallel 10 ./...

# Run specific test suite
go test -v ./http/...  # HTTP API tests
go test -v ./cli/...   # CLI tests
```

### Environment Variables

Override default CLI paths if needed:

```bash
# Set Go CLI path (default: ../../scitt-golang/scitt)
export SCITT_GO_CLI=/path/to/scitt

# Set TypeScript CLI command (default: bun run ../../scitt-typescript/src/cli/index.ts)
export SCITT_TS_CLI="bun run /path/to/cli.ts"
```

## CLI Flag Differences

The Go and TypeScript implementations have different CLI flags due to their independent development. The test suite **adapts to these differences** where possible.

### Init Command

#### Go Implementation
```bash
scitt init --dir ./path --origin https://example.com [--force]
```

Flags:
- `--dir`: Directory to initialize service in (required)
- `--origin`: Origin URL for transparency service (required)
- `--force`: Overwrite existing files
- `--db`: Database file path (default: scitt.db)
- `--storage`: Storage directory path (default: ./storage)

#### TypeScript Implementation
```bash
bun run cli.ts init --path ./path [--generate-key] [--origin https://example.com]
```

Flags:
- `--path`: Directory to initialize service in
- `--generate-key`: Generate ES256 keypair during init
- `--origin`: Origin URL (optional)

**Test Adaptation**: Tests use different flags for each implementation when calling init commands.

### Statement Commands

#### Go Implementation
```bash
scitt statement sign --payload file.json --key key.pem
scitt statement verify --statement statement.cose --key key.pem
scitt statement hash --payload file.json
```

#### TypeScript Implementation
```bash
bun run cli.ts statement sign --payload file.json --key key.json
bun run cli.ts statement verify --statement statement.cose --key key.json
bun run cli.ts statement hash --payload file.json
```

**Key Difference**: Key format (PEM vs JWK JSON)

### Serve Command

#### Go Implementation
```bash
scitt serve --port 8080 --origin https://example.com
```

#### TypeScript Implementation
```bash
bun run cli.ts serve --port 8080 --origin https://example.com
```

## Test Behavior

### Tests That May Skip

Some tests will skip under certain conditions:

#### HTTP API Tests
- **All tests** skip if server startup is not implemented
- Message: "Go server startup not yet implemented"
- **Why**: Server process management needs implementation-specific details

#### CLI Tests
- **Init tests** may fail if:
  - CLI binaries are not built
  - Required flags are missing
  - Wrong flag names are used
- **Statement tests** may skip if:
  - Key formats differ between implementations
  - Signature verification is not cross-compatible (yet)

### Expected Test Outcomes

With both implementations built and available:

#### ✅ Should Pass
- Type system compilation
- Test environment setup
- Port allocation
- Directory structure creation

#### ⚠️ May Skip (Implementation-Dependent)
- Server startup tests (stub implementations)
- Cross-signature verification (depends on key format compatibility)
- Some init command variations (depends on CLI flag support)

#### ❌ Currently Fail (Known Issues)
- Go init without `--origin` flag (required but not provided in tests)
- TypeScript CLI path resolution (needs correct module path)

## Implementation Status

### Phase 1-4: Complete ✅
- Test infrastructure (types, setup, comparison, validation)
- HTTP API test structure (6 files)
- CLI test structure (4 files)
- Reporting system

### Server Integration: Stub Implementation 🚧
- `startGoServer()` - Ready for implementation details
- `startTsServer()` - Ready for implementation details
- Tests will skip until implemented

### Cross-Implementation Verification: Pending 🔜
- Go signs → TypeScript verifies
- TypeScript signs → Go verifies
- Depends on key format compatibility

## Fixing Common Issues

### Issue: "Go init failed: exit=-1"

**Cause**: Go CLI requires `--origin` flag

**Fix Options**:
1. Update tests to include `--origin` flag
2. Make `--origin` optional in Go CLI
3. Use environment variable for default origin

### Issue: "Module not found" (TypeScript)

**Cause**: TypeScript CLI path is incorrect

**Fix**:
```bash
# Check actual TypeScript CLI location
ls ../../scitt-typescript/src/cli/index.ts

# Set environment variable
export SCITT_TS_CLI="bun run $(pwd)/../../scitt-typescript/src/cli/index.ts"
```

### Issue: Port allocation failures

**Cause**: Too many parallel tests or ports in use

**Fix**:
```bash
# Reduce parallelism
go test -v -parallel 1 ./...

# Or check for processes using ports 20000-30000
lsof -i :20000-30000
```

## Test Output

### JSON Output (for CI)
```bash
go test -json ./... > results.json
```

### Generate Reports
```bash
# After running tests
go run ./lib/report.go --input results.json --output ./reports
```

Reports generated:
- `test-report-{id}.json` - Machine-readable format
- `test-report-{id}.md` - Human-readable Markdown

## CI Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'

      - name: Setup Bun
        uses: oven-sh/setup-bun@v1

      - name: Build Implementations
        run: |
          cd scitt-golang && go build -o scitt ./cmd/scitt
          cd ../scitt-typescript && bun install

      - name: Run Integration Tests
        run: |
          cd tests/interop
          go test -v -parallel 10 -json ./... > results.json

      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: tests/interop/results.json
```

## Next Steps

### Immediate (for full test execution)

1. **Align CLI flags**: Update tests to use correct flags per implementation
2. **Implement server starters**: Add `startGoServer()` and `startTsServer()` logic
3. **Key format bridge**: Create conversion utilities for PEM ↔ JWK

### Future Phases

- Phase 5: Cryptographic Interoperability (5 tasks)
- Phase 6: Merkle Tree Proofs (6 tasks)
- Phase 7: Query Compatibility (6 tasks)
- Phase 8: Receipt Compatibility (7 tasks)
- Phase 9: End-to-End Workflows (5 tasks)

## Contributing

When adding new tests:

1. Use `lib.SetupTestEnv(t)` for test isolation
2. Allocate ports with `lib.GlobalPortAllocator.AllocatePort(t)`
3. Use semantic comparison with `lib.CompareOutputs()`
4. Validate RFC compliance with `lib.ValidateRFCCompliance()`
5. Follow snake_case and hex encoding conventions
6. Document CLI flag differences in this guide

## Reference

- **Main Documentation**: `INTEGRATION-TESTS.md`
- **MVP Summary**: `MVP-COMPLETE.md`
- **Original Interop Doc**: `README.md`
- **Specification**: `../../specs/003-create-an-integration/`

## Support

For issues or questions:
1. Check this guide first
2. Review test output and error messages
3. Verify both implementations are built correctly
4. Check environment variables are set correctly
5. File an issue in the repository with test output
