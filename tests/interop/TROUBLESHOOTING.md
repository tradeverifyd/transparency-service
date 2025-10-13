# Troubleshooting Guide - Integration Test Suite

**Version**: 1.0
**Date**: 2025-10-12

## Quick Diagnostics

### 1. Check Test Suite Health

```bash
cd tests/interop

# Verify Go module is valid
go mod verify

# Verify binaries exist
./scripts/build_impls.sh

# Run smoke test
go test -v -run TestCryptoFixturesExist ./crypto/
```

### 2. Check Implementation Status

```bash
# Verify Go implementation builds
cd ../../scitt-golang
go build -o scitt ./cmd/scitt
./scitt --version

# Verify TypeScript implementation works
cd ../scitt-typescript
bun install
bun src/cli/index.ts --help
```

## Common Issues

### Issue 1: Tests Can't Find Binaries

**Symptom**:
```
Error: exec: "scitt": executable file not found in $PATH
```

**Cause**: Go or TypeScript binaries not built or not in expected locations

**Solution**:
```bash
cd tests/interop

# Build both implementations
./scripts/build_impls.sh

# Verify binaries exist
ls -la ../../scitt-golang/scitt
ls -la ../../scitt-typescript/src/cli/index.ts

# Or set environment variables explicitly
export SCITT_GO_CLI=/abs/path/to/scitt-golang/scitt
export SCITT_TS_CLI="bun run /abs/path/to/scitt-typescript/src/cli/index.ts"

# Run tests again
go test -v ./http/
```

### Issue 2: Port Allocation Conflicts

**Symptom**:
```
Error: bind: address already in use
```

**Cause**: Previous test didn't clean up, or system service using port in 20000-30000 range

**Solution**:
```bash
# Check what's using ports
lsof -i :20000-30000

# Kill stray processes
pkill -f "scitt.*serve"
pkill -f "bun.*cli"

# Or run tests sequentially (slower but avoids conflicts)
go test -v -parallel 1 ./...
```

### Issue 3: Fixture Not Found

**Symptom**:
```
Failed to read keypair: open fixtures/keys/keypair_alice.json: no such file or directory
```

**Cause**: Test running from wrong directory or fixtures not generated

**Solution**:
```bash
cd tests/interop

# Verify fixtures exist
ls -la fixtures/keys/
ls -la fixtures/statements/

# Regenerate if missing
cd tools
go run generate_keypair.go
go run generate_cose_statement.go

# Run test from correct directory
cd ..
go test -v ./crypto/
```

### Issue 4: Module Import Errors

**Symptom**:
```
no required module provides package github.com/tradeverifyd/scitt/tests/interop/lib
```

**Cause**: Go module path mismatch or dependencies not downloaded

**Solution**:
```bash
cd tests/interop

# Update module dependencies
go mod tidy

# Verify module name matches
head -1 go.mod
# Should be: module github.com/tradeverifyd/scitt/tests/interop

# If module name is wrong, update it
go mod edit -module=github.com/tradeverifyd/scitt/tests/interop

# Download dependencies
go mod download

# Run tests
go test -v ./...
```

### Issue 5: JSON Comparison Failures

**Symptom**:
```
Implementation divergence detected:
  - Field: entry_id
  - Go: 1
  - TypeScript: 0
  - Severity: minor
```

**Cause**: Implementations use different starting indices (expected behavior)

**Solution**: This is EXPECTED and documented. Tests accept both starting indices.

If you see MAJOR differences:
```bash
# Check if both implementations are up to date
cd ../../scitt-golang
git pull
go build -o scitt ./cmd/scitt

cd ../scitt-typescript
git pull
bun install

# Rebuild and re-test
cd ../tests/interop
./scripts/build_impls.sh
go test -v ./http/entries_test.go
```

### Issue 6: Server Won't Start

**Symptom**:
```
Error: failed to start server: listen tcp :8080: bind: address already in use
```

**Cause**: Port already in use or insufficient permissions

**Solution**:
```bash
# Check what's on port 8080
lsof -i :8080

# Kill if it's a stray test server
kill $(lsof -t -i:8080)

# Or let test allocate unique port
# Tests use lib.AllocatePort() which avoids conflicts

# Verify port allocator works
cd tests/interop
go test -v -run TestPortAllocation ./lib/
```

### Issue 7: COSE Decoding Errors

**Symptom**:
```
Error: invalid COSE Sign1: expected array of length 4
```

**Cause**: COSE fixture corrupted or generated with wrong version

**Solution**:
```bash
cd tests/interop/tools

# Regenerate COSE statements
go run generate_cose_statement.go

# Verify they're valid CBOR
cd ../fixtures/statements
xxd small.cose | head -n 1
# Should start with: 84 (CBOR array of 4 elements)

# Inspect structure
cd ../../tools
go run -tags debug inspect_cose.go ../fixtures/statements/small.cose
```

### Issue 8: Timeout Errors

**Symptom**:
```
Error: context deadline exceeded
```

**Cause**: Server startup slow, network issues, or CPU constraints

**Solution**:
```bash
# Increase timeout in test
export SCITT_TEST_TIMEOUT=300  # 5 minutes

# Or edit test directly
# In test file, change timeoutSec parameter:
result := lib.RunGoCLI(args, dir, env, 120) // Was 30

# Check system resources
top
# If CPU/memory constrained, run tests sequentially:
go test -v -parallel 1 ./...
```

### Issue 9: Permission Denied Errors

**Symptom**:
```
Error: permission denied: /tmp/test-12345/scitt.db
```

**Cause**: Temp directory permissions or SELinux/AppArmor restrictions

**Solution**:
```bash
# Check temp directory permissions
ls -la /tmp/test-*

# On macOS, grant Full Disk Access to Terminal
# System Preferences > Security & Privacy > Full Disk Access

# On Linux, check SELinux
sestatus
# If enforcing, temporarily disable for testing:
sudo setenforce 0

# Or use different temp location
export TMPDIR=$HOME/tmp
mkdir -p $HOME/tmp
go test -v ./...
```

### Issue 10: Standards Alignment Issues

**Symptom**:
```
Expected snake_case field 'entry_id', got 'entryId'
```

**Cause**: Implementations not using consistent JSON field naming

**Solution**: This was fixed in the standards alignment work. If you see this:

```bash
# Verify you're on the correct branch
git branch
# Should show: 003-create-an-integration

# Pull latest changes
git pull origin 003-create-an-integration

# Rebuild implementations
cd scitt-golang && go build -o scitt ./cmd/scitt
cd ../scitt-typescript && bun install

# Verify standards alignment
cd ../tests/interop
go test -v ./http/entries_test.go

# Should see:
# ✓ Both use snake_case entry_id
# ✓ Both use integer entry IDs
```

## Debug Workflow

### Step 1: Isolate the Problem

```bash
# Run only the failing test
go test -v -run TestPostEntries ./http/

# If it passes, likely a parallel execution issue
# Run with less parallelism:
go test -v -parallel 2 ./http/
```

### Step 2: Inspect Test Output

```bash
# Run with verbose output
go test -v ./http/entries_test.go

# Look for:
# - HTTP status codes (should be 201, 200, etc.)
# - Response bodies (should be valid JSON)
# - Error messages (should be descriptive)
```

### Step 3: Check Implementation Logs

```bash
# Go implementation logs
cd ../../scitt-golang
./scitt serve --port 8080 --log-level debug

# TypeScript implementation logs
cd ../scitt-typescript
SCITT_LOG_LEVEL=debug bun src/cli/index.ts serve --port 8081
```

### Step 4: Manual Verification

```bash
# Start both servers manually
cd scitt-golang
./scitt serve --port 8080 &
GO_PID=$!

cd ../scitt-typescript
bun src/cli/index.ts serve --port 8081 &
TS_PID=$!

# Test manually with curl
curl -X POST http://localhost:8080/entries \
  -H "Content-Type: application/cose" \
  --data-binary @../tests/interop/fixtures/statements/small.cose

curl -X POST http://localhost:8081/entries \
  -H "Content-Type: application/cose" \
  --data-binary @../tests/interop/fixtures/statements/small.cose

# Compare responses
# Should both return 201 with snake_case fields

# Cleanup
kill $GO_PID $TS_PID
```

### Step 5: Inspect Test Artifacts

```bash
# Tests create temp directories
# To preserve them for inspection, modify test:

func TestExample(t *testing.T) {
    // Comment out cleanup temporarily
    // t.Cleanup(cleanup)

    env := lib.SetupTestEnv(t)
    t.Logf("Test dir: %s", env.TempDir)

    // Run test...
}

# After test fails, inspect the directory
ls -la /tmp/TestExample*
```

## Environment Variables

Set these to customize test behavior:

```bash
# Binary locations
export SCITT_GO_CLI=/custom/path/to/scitt
export SCITT_TS_CLI="bun run /custom/path/to/cli.ts"

# Timeouts
export SCITT_TEST_TIMEOUT=120  # seconds

# Ports
export SCITT_TEST_PORT_MIN=20000
export SCITT_TEST_PORT_MAX=30000

# Temp directory
export TMPDIR=$HOME/scitt-tests

# Debug output
export SCITT_TEST_DEBUG=1
```

## Test Execution Tips

### Run Specific Test Pattern

```bash
# Run all tests matching pattern
go test -v -run TestPost ./http/

# Run specific subtest
go test -v -run TestPostEntries/Go ./http/
```

### Generate Test Report

```bash
# JSON output for CI
go test -json ./... > results.json

# Parse with jq
cat results.json | jq -r 'select(.Action=="fail") | .Test'
```

### Profile Test Performance

```bash
# CPU profile
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof

# Memory profile
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

### Test Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

## Getting Help

### Check Documentation

1. **ARCHITECTURE.md** - Test suite design and principles
2. **README.md** - Quick start and running tests
3. **SPEC-003-COMPLETION-PLAN.md** - What's been tested
4. **specs/003-create-an-integration/** - Full specification

### Common Commands Reference

```bash
# Full test suite
cd tests/interop && go test -v ./...

# Specific category
go test -v ./http/      # HTTP API tests
go test -v ./cli/       # CLI tests
go test -v ./crypto/    # Crypto tests

# With parallelism
go test -v -parallel 10 ./...

# Rebuild implementations
./scripts/build_impls.sh

# Generate fixtures
cd tools && go run generate_keypair.go
cd tools && go run generate_cose_statement.go
```

### Report Issues

If you encounter issues not covered here:

1. Check GitHub issues: https://github.com/tradeverifyd/transparency-service/issues
2. Include:
   - Test command used
   - Full error output
   - Go version (`go version`)
   - Bun version (`bun --version`)
   - OS and version
3. Attach test output: `go test -v ./... > test-output.txt 2>&1`

## Summary

Most issues stem from:
- ✅ Binaries not built → Run `./scripts/build_impls.sh`
- ✅ Wrong directory → Run tests from `tests/interop/`
- ✅ Port conflicts → Use parallel tests with unique ports
- ✅ Missing fixtures → Regenerate with tools
- ✅ Module issues → Run `go mod tidy`

For persistent issues, consult ARCHITECTURE.md or open a GitHub issue with full diagnostics.
