# Release Validation Guide

This guide describes how to use the SCITT interoperability test suite as part of your release validation process.

## Table of Contents

- [Overview](#overview)
- [Pre-Release Validation](#pre-release-validation)
- [Release Checklist](#release-checklist)
- [Validation Scenarios](#validation-scenarios)
- [Interpreting Results](#interpreting-results)
- [Troubleshooting](#troubleshooting)

## Overview

The SCITT interoperability test suite ensures both Go and TypeScript implementations maintain:
- **Standards compliance** (SCRAPI, COSE, Merkle Trees, JWK)
- **API compatibility** (REST endpoints, CLI commands)
- **Cross-implementation interoperability** (shared data formats)

### When to Run Validation

Run the test suite:
- ✅ Before creating a release candidate
- ✅ Before merging major features
- ✅ After dependency updates
- ✅ Before publishing packages
- ✅ When troubleshooting interoperability issues

## Pre-Release Validation

### 1. Environment Setup

Ensure both implementations are built:

```bash
# From repository root
cd tests/interop

# Build both implementations
./scripts/build_impls.sh
```

**Expected Output:**
```
Building Go implementation...
✓ Go CLI built: /path/to/scitt-golang/scitt

Building TypeScript implementation...
✓ TypeScript dependencies installed
✓ TypeScript implementation ready
```

### 2. Run Full Test Suite

Execute all tests with verbose output:

```bash
# Run all tests in parallel
go test -v -parallel 10 ./...

# Or run with coverage
go test -v -parallel 10 -coverprofile=coverage.out ./...
```

**Expected Duration:** < 2 minutes for full suite

### 3. Verify Critical Paths

Run specific test categories:

```bash
# CLI interoperability
go test -v ./cli/

# HTTP API interoperability
go test -v ./http/

# Crypto and standards compliance
go test -v ./crypto/
```

## Release Checklist

Use this checklist before each release:

### Go Implementation Release

- [ ] **Build succeeds**: `cd scitt-golang && go build -v ./cmd/scitt`
- [ ] **Unit tests pass**: `cd scitt-golang && go test ./...`
- [ ] **CLI tests pass**: `cd tests/interop && go test -v ./cli/`
- [ ] **HTTP tests pass**: `cd tests/interop && go test -v ./http/`
- [ ] **Crypto tests pass**: `cd tests/interop && go test -v ./crypto/`
- [ ] **Version updated**: Check `scitt-golang/internal/version/version.go`
- [ ] **Changelog updated**: Document changes
- [ ] **Tag created**: `git tag -a go/v1.x.x -m "Release v1.x.x"`

### TypeScript Implementation Release

- [ ] **Dependencies install**: `cd scitt-typescript && bun install`
- [ ] **Build succeeds**: `cd scitt-typescript && bun build`
- [ ] **Unit tests pass**: `cd scitt-typescript && bun test`
- [ ] **CLI tests pass**: `cd tests/interop && go test -v ./cli/`
- [ ] **HTTP tests pass**: `cd tests/interop && go test -v ./http/`
- [ ] **Crypto tests pass**: `cd tests/interop && go test -v ./crypto/`
- [ ] **Version updated**: Check `scitt-typescript/package.json`
- [ ] **Changelog updated**: Document changes
- [ ] **Tag created**: `git tag -a ts/v1.x.x -m "Release v1.x.x"`

### Combined Release

- [ ] **Both implementations built**
- [ ] **Full interop suite passes**: `cd tests/interop && go test -v -parallel 10 ./...`
- [ ] **Cross-signing works**: CLI tests verify statements signed by one impl are verified by the other
- [ ] **Data format compatibility**: HTTP tests verify both impls read each other's data
- [ ] **Documentation updated**: Update API docs, README files
- [ ] **Release notes prepared**: Include breaking changes, new features, bug fixes

## Validation Scenarios

### Scenario 1: Basic Functionality

**Goal**: Verify core registration and retrieval operations

```bash
# Test CLI registration
go test -v -run TestCLIRegisterStatement ./cli/

# Test HTTP registration
go test -v -run TestHTTPRegisterStatement ./http/
```

**Success Criteria**:
- Both implementations register statements successfully
- Entry IDs returned as integers
- Response format matches SCRAPI specification (snake_case)

### Scenario 2: Cross-Implementation Compatibility

**Goal**: Verify statements from one implementation work with the other

```bash
# Test cross-signing
go test -v -run TestCLICrossSigning ./cli/

# Test cross-retrieval
go test -v -run TestHTTPCrossRetrieval ./http/
```

**Success Criteria**:
- Statements signed by Go CLI can be submitted to TypeScript server
- Statements signed by TypeScript CLI can be submitted to Go server
- Both implementations generate compatible receipts

### Scenario 3: Standards Compliance

**Goal**: Verify adherence to IETF RFCs and SCRAPI

```bash
# Test crypto compliance
go test -v ./crypto/

# Specifically test SCRAPI format
go test -v -run TestRegistrationResponseFormat ./http/
```

**Success Criteria**:
- COSE signatures comply with RFC 9052
- JWK thumbprints comply with RFC 7638
- Merkle tree operations comply with RFC 6962
- API responses use snake_case with integer entry_id

### Scenario 4: Error Handling

**Goal**: Verify both implementations handle errors consistently

```bash
# Test error scenarios
go test -v -run TestErrorHandling ./http/
go test -v -run TestInvalidInputs ./cli/
```

**Success Criteria**:
- Invalid COSE statements rejected by both implementations
- Appropriate HTTP status codes returned
- Error messages are informative

### Scenario 5: Performance and Concurrency

**Goal**: Verify implementations handle load

```bash
# Run with high parallelism
go test -v -parallel 20 ./...

# Run with timeout monitoring
go test -v -timeout 5m ./...
```

**Success Criteria**:
- No race conditions detected
- Tests complete within timeout (< 5 minutes)
- No resource leaks (ports released, files closed)

## Interpreting Results

### Successful Test Run

```
PASS: TestCLIRegisterStatement/go_implementation (0.25s)
PASS: TestCLIRegisterStatement/typescript_implementation (0.32s)
PASS: TestHTTPRegisterStatement/go_server (0.18s)
PASS: TestHTTPRegisterStatement/typescript_server (0.21s)
ok      github.com/tradeverifyd/scitt/tests/interop/cli    1.234s
ok      github.com/tradeverifyd/scitt/tests/interop/http   0.987s
```

**Interpretation**: All tests passed, implementations are compatible.

### Failed Test Example

```
FAIL: TestHTTPRegisterStatement/typescript_server (0.52s)
    register_test.go:45: Expected snake_case field 'entry_id', got 'entryId'
    register_test.go:67: Response validation failed
```

**Interpretation**: TypeScript implementation not returning SCRAPI-compliant response format.

**Action Required**: Update TypeScript implementation to use snake_case.

### Flaky Test Example

```
FAIL: TestHTTPCrossRetrieval/go_to_typescript (0.89s)
    retrieval_test.go:78: context deadline exceeded
```

**Interpretation**: Test timed out, possible port conflict or server startup issue.

**Action Required**: Check for port conflicts, increase timeout if necessary.

## Troubleshooting

### Issue 1: Binary Not Found

**Symptom:**
```
Error: Go binary not found at /path/to/scitt
```

**Solution:**
```bash
# Rebuild Go implementation
cd scitt-golang
go build -v -o scitt ./cmd/scitt
chmod +x scitt

# Set environment variable if needed
export SCITT_GO_CLI=/absolute/path/to/scitt-golang/scitt
```

### Issue 2: Port Conflicts

**Symptom:**
```
Error: Failed to start server: address already in use
```

**Solution:**
```bash
# Find and kill processes using test ports
lsof -ti:20000-30000 | xargs kill -9

# Or wait for automatic cleanup
sleep 5 && go test -v ./...
```

### Issue 3: Fixture Loading Errors

**Symptom:**
```
Error: Failed to read keypair: no such file or directory
```

**Solution:**
```bash
# Ensure running from correct directory
cd tests/interop
go test -v ./crypto/

# Verify fixtures exist
ls -la fixtures/keys/
ls -la fixtures/cose/
```

### Issue 4: Response Format Mismatch

**Symptom:**
```
Expected field 'entry_id' (integer), got 'entryId' (string)
```

**Solution:**
1. Verify implementation is returning SCRAPI-compliant format
2. Check JSON struct tags use snake_case
3. Ensure entry_id is returned as integer, not string

**Go Implementation Check:**
```go
type RegisterStatementResponse struct {
    EntryID       int64  `json:"entry_id"`       // ✓ Correct
    StatementHash string `json:"statement_hash"` // ✓ Correct
}
```

**TypeScript Implementation Check:**
```typescript
const response: RegistrationResponse = {
  entry_id: leafIndex,  // ✓ integer, not string
  receipt: receipt
};
```

### Issue 5: Cross-Signing Failures

**Symptom:**
```
Verification failed: signature verification failed
```

**Solution:**
1. Check COSE signature format (IEEE P1363, not DER)
2. Verify JWK public key format matches RFC 7517
3. Ensure ES256 algorithm used consistently
4. Check signature detachment is correct

### Issue 6: Test Timeouts

**Symptom:**
```
panic: test timed out after 2m0s
```

**Solution:**
```bash
# Increase timeout
go test -v -timeout 5m ./...

# Run with less parallelism
go test -v -parallel 5 ./...

# Debug specific test
go test -v -run TestSpecificTest ./path/
```

## Advanced Validation

### Generate Test Coverage Report

```bash
cd tests/interop

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# View report
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

**Target Coverage**: > 80% for interop test code

### Run Tests in CI Environment

```bash
# Simulate CI environment
env -i HOME="$HOME" \
  PATH="$PATH" \
  SCITT_GO_CLI="$(pwd)/../../scitt-golang/scitt" \
  SCITT_TS_CLI="bun run $(pwd)/../../scitt-typescript/src/cli/index.ts" \
  go test -v -parallel 10 ./...
```

### Validate Against Previous Release

```bash
# Test new Go implementation with old TypeScript
git worktree add /tmp/ts-old ts/v1.0.0
SCITT_TS_CLI="bun run /tmp/ts-old/scitt-typescript/src/cli/index.ts" \
  go test -v ./...

# Test new TypeScript implementation with old Go
git worktree add /tmp/go-old go/v1.0.0
SCITT_GO_CLI="/tmp/go-old/scitt-golang/scitt" \
  go test -v ./...
```

## Release Approval Criteria

Before approving a release, ensure:

### Must Pass (Blocking)
- ✅ All CLI tests pass
- ✅ All HTTP tests pass
- ✅ All crypto tests pass
- ✅ No SCRAPI compliance violations
- ✅ Cross-implementation compatibility maintained

### Should Pass (Non-Blocking)
- ⚠️ Test coverage > 80%
- ⚠️ No flaky tests (< 1% failure rate)
- ⚠️ Performance benchmarks within acceptable range
- ⚠️ Documentation updated

### Nice to Have
- 📋 New test cases for new features
- 📋 Expanded test fixtures
- 📋 Performance improvements documented

## Integration with Release Process

### GitHub Actions

The test suite runs automatically in CI:

```yaml
# .github/workflows/ci-interop.yml triggers on:
- Push to main
- Pull requests to main
```

View results at: `https://github.com/your-org/transparency-service/actions`

### Manual Release Process

```bash
# 1. Create release branch
git checkout -b release/v1.x.x

# 2. Update versions
# Edit scitt-golang/internal/version/version.go
# Edit scitt-typescript/package.json

# 3. Run full validation
cd tests/interop
./scripts/build_impls.sh
go test -v -parallel 10 ./...

# 4. If all tests pass, commit and tag
git add .
git commit -m "Prepare release v1.x.x"
git tag -a v1.x.x -m "Release v1.x.x"
git push origin release/v1.x.x
git push origin v1.x.x

# 5. Create pull request for main
# Wait for CI to pass
# Merge to main
```

## Best Practices

1. **Run tests frequently** - Don't wait until release time
2. **Fix failures immediately** - Don't accumulate test debt
3. **Add tests for bugs** - Prevent regressions
4. **Monitor CI results** - Set up notifications for failures
5. **Version carefully** - Use semantic versioning
6. **Document breaking changes** - Update migration guides
7. **Test backwards compatibility** - Validate against older versions

## Support

For issues with the test suite:

1. Check [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) for common solutions
2. Review [ARCHITECTURE.md](./ARCHITECTURE.md) for test design details
3. Check test logs for specific error messages
4. Open an issue with reproduction steps

## References

- [ARCHITECTURE.md](./ARCHITECTURE.md) - Test suite design
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - Debug guide
- [COMPLETION-REPORT.md](./COMPLETION-REPORT.md) - Implementation status
- [SCRAPI Specification](https://datatracker.ietf.org/doc/html/draft-ietf-scitt-scrapi)
- [RFC 9052 (COSE)](https://datatracker.ietf.org/doc/html/rfc9052)
- [RFC 6962 (Merkle Trees)](https://datatracker.ietf.org/doc/html/rfc6962)
