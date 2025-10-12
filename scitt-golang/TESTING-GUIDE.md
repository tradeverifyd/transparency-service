# End-to-End Testing Guide

Complete guide for testing the SCITT transparency service Go implementation.

## Quick Start

```bash
# Build the CLI tool
cd scitt-golang
go build -o scitt ./cmd/scitt

# Initialize a new transparency service
./scitt init --origin https://transparency.example.com

# Start the HTTP server
./scitt serve --config scitt.yaml
```

## Test Scenarios

### Scenario 1: Complete Statement Registration Flow

This scenario tests the full workflow: sign statement → register → retrieve receipt → verify.

#### Step 1: Generate Test Keys

```bash
# Create a test issuer key pair
./scitt statement sign \
  --input <(echo '{"test": "data"}') \
  --key /dev/null \
  --output /dev/null \
  --generate-key issuer-key.pem

# The init command already created service keys:
# - service-key.pem (private)
# - service-key.jwk (public)
```

#### Step 2: Create and Sign a Statement

```bash
# Create test payload
echo '{"artifact": "my-app:v1.0.0", "timestamp": "2025-10-12T00:00:00Z"}' > payload.json

# Sign the statement
./scitt statement sign \
  --input payload.json \
  --key issuer-key.pem \
  --output statement.cbor \
  --issuer "https://issuer.example.com" \
  --subject "my-app:v1.0.0" \
  --content-type "application/json"

echo "✓ Statement signed and saved to statement.cbor"
```

#### Step 3: Compute Statement Hash

```bash
# Get the hash that will be used as entry ID
./scitt statement hash --input statement.cbor

# Output example:
# Statement Hash: a1b2c3d4e5f6...
```

#### Step 4: Start the Transparency Service

```bash
# In a separate terminal, start the server
./scitt serve --config scitt.yaml

# Server should start on http://127.0.0.1:8080
# Wait for "Server listening on 127.0.0.1:8080" message
```

#### Step 5: Register the Statement

```bash
# Register via HTTP API
curl -X POST http://127.0.0.1:8080/entries \
  -H "Content-Type: application/cose" \
  --data-binary @statement.cbor \
  -v

# Expected response (202 Accepted):
# {
#   "entryId": 0,
#   "statementHash": "a1b2c3d4e5f6..."
# }

# Save the entryId for next steps
ENTRY_ID=0
```

#### Step 6: Retrieve the Receipt

```bash
# Get receipt by entry ID
curl http://127.0.0.1:8080/entries/$ENTRY_ID \
  -H "Accept: application/json" \
  -o receipt.json

# View receipt
cat receipt.json

# Expected structure:
# {
#   "entryId": 0,
#   "statementHash": "a1b2c3d4e5f6...",
#   "treeSize": 1,
#   "timestamp": 1728691200
# }
```

#### Step 7: Get Current Checkpoint

```bash
# Retrieve signed tree head
curl http://127.0.0.1:8080/checkpoint \
  -o checkpoint.txt

# View checkpoint
cat checkpoint.txt

# Expected format (signed note):
# https://transparency.example.com
# 1
# <base64-root-hash>
# <timestamp>
#
# — https://transparency.example.com <base64-signature>
```

#### Step 8: Query Service Configuration

```bash
# Get transparency configuration
curl http://127.0.0.1:8080/.well-known/transparency-configuration

# Expected response:
# {
#   "origin": "https://transparency.example.com",
#   "supported_algorithms": ["ES256"],
#   "supported_hash_algorithms": ["SHA-256"],
#   "registration_policy": {
#     "type": "open"
#   }
# }
```

### Scenario 2: Multiple Statement Registration

Test tree growth and checkpoint updates.

```bash
# Register 10 statements
for i in {1..10}; do
  echo "{\"artifact\": \"app-$i:v1.0.0\"}" > payload-$i.json

  ./scitt statement sign \
    --input payload-$i.json \
    --key issuer-key.pem \
    --output statement-$i.cbor \
    --issuer "https://issuer.example.com" \
    --subject "app-$i:v1.0.0"

  curl -X POST http://127.0.0.1:8080/entries \
    -H "Content-Type: application/cose" \
    --data-binary @statement-$i.cbor

  echo "✓ Registered statement $i"
done

# Verify tree size
curl http://127.0.0.1:8080/checkpoint | head -2

# Should show:
# https://transparency.example.com
# 10
```

### Scenario 3: Statement Verification

Verify statement signatures before registration.

```bash
# Verify with CLI
./scitt statement verify \
  --input statement.cbor \
  --key issuer-key.pem

# Expected output:
# ✓ Signature valid
# ✓ Statement verified successfully

# Verify with wrong key (should fail)
./scitt statement verify \
  --input statement.cbor \
  --key service-key.pem

# Expected:
# ✗ Signature verification failed
```

### Scenario 4: Database Queries

Test database operations directly.

```bash
# Connect to SQLite database
sqlite3 scitt.db

# Check current tree size
SELECT * FROM current_tree_size;

# Query all statements
SELECT entry_id, statement_hash, iss, sub, registered_at
FROM statements
ORDER BY entry_id;

# Query by issuer
SELECT COUNT(*) FROM statements
WHERE iss = 'https://issuer.example.com';

# View tree state history
SELECT tree_size, root_hash, updated_at
FROM tree_state
ORDER BY tree_size DESC
LIMIT 5;

# Exit
.quit
```

### Scenario 5: Storage Verification

Verify tile storage on disk.

```bash
# List storage directory
ls -la storage/

# Check for entry tiles
find storage/ -name "*.tile" -o -type f

# Verify tile contents (binary)
hexdump -C storage/tile/entries/0 | head
```

### Scenario 6: Configuration Management

Test configuration loading and validation.

```bash
# Test with custom config
cat > test-config.yaml <<EOF
origin: https://test.example.com
database:
  path: test.db
  enable_wal: true
storage:
  type: local
  path: ./test-storage
keys:
  private: service-key.pem
  public: service-key.jwk
server:
  host: localhost
  port: 9000
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:3000"
EOF

# Initialize with custom config
./scitt init --origin https://test.example.com --config test-config.yaml

# Start server with custom config
./scitt serve --config test-config.yaml --port 9001

# Test on custom port
curl http://localhost:9001/health
```

### Scenario 7: Error Handling

Test error responses and validation.

```bash
# Test invalid COSE Sign1 (should fail)
echo "invalid data" > invalid.cbor
curl -X POST http://127.0.0.1:8080/entries \
  -H "Content-Type: application/cose" \
  --data-binary @invalid.cbor \
  -v

# Expected: 400 Bad Request

# Test non-existent entry
curl http://127.0.0.1:8080/entries/999999 -v

# Expected: 404 Not Found

# Test invalid content type
curl -X POST http://127.0.0.1:8080/entries \
  -H "Content-Type: text/plain" \
  --data "invalid" \
  -v

# Expected: 400 Bad Request
```

### Scenario 8: CORS Testing

Test Cross-Origin Resource Sharing.

```bash
# Preflight request
curl -X OPTIONS http://127.0.0.1:8080/entries \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: POST" \
  -v

# Check for CORS headers:
# Access-Control-Allow-Origin: *
# Access-Control-Allow-Methods: GET, POST, OPTIONS

# Actual request with origin
curl http://127.0.0.1:8080/checkpoint \
  -H "Origin: http://localhost:3000" \
  -v

# Should include Access-Control-Allow-Origin header
```

## Automated Testing

### Unit Tests

```bash
# Run all unit tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Run specific package tests
go test -v ./pkg/cose
go test -v ./pkg/database
go test -v ./pkg/merkle
go test -v ./internal/cli

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

### Integration Tests

```bash
# Run integration tests (when implemented)
go test ./tests/integration/... -v

# Run with race detection
go test ./... -race

# Run with timeout
go test ./... -timeout 30s
```

## Performance Testing

### Load Testing with Apache Bench

```bash
# Install Apache Bench (if not installed)
# macOS: brew install httpd
# Ubuntu: sudo apt-get install apache2-utils

# Create test statement
./scitt statement sign \
  --input <(echo '{"test": "load"}') \
  --key issuer-key.pem \
  --output load-test.cbor \
  --issuer "https://test.com" \
  --subject "load-test"

# Load test: 1000 requests, 10 concurrent
ab -n 1000 -c 10 -p load-test.cbor \
  -T application/cose \
  http://127.0.0.1:8080/entries

# Load test checkpoints
ab -n 1000 -c 10 http://127.0.0.1:8080/checkpoint
```

### Memory and CPU Profiling

```bash
# Run server with profiling
go run ./cmd/scitt serve --config scitt.yaml &
SERVER_PID=$!

# Generate some load
for i in {1..100}; do
  curl -X POST http://127.0.0.1:8080/entries \
    -H "Content-Type: application/cose" \
    --data-binary @statement.cbor > /dev/null 2>&1
done

# Get memory profile
curl http://127.0.0.1:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Stop server
kill $SERVER_PID
```

## Debugging

### Enable Verbose Logging

```bash
# Run with verbose flag
./scitt serve --config scitt.yaml --verbose

# Check logs for:
# - Request details
# - Database operations
# - Storage operations
# - Error traces
```

### Database Inspection

```bash
# Check database integrity
sqlite3 scitt.db "PRAGMA integrity_check;"

# View schema
sqlite3 scitt.db ".schema"

# Check table sizes
sqlite3 scitt.db <<EOF
SELECT 'statements' as table_name, COUNT(*) as count FROM statements
UNION ALL
SELECT 'receipts', COUNT(*) FROM receipts
UNION ALL
SELECT 'tree_state', COUNT(*) FROM tree_state;
EOF
```

### Storage Debugging

```bash
# Check storage directory size
du -sh storage/

# Count stored tiles
find storage/ -type f | wc -l

# Verify tile integrity
for file in storage/tile/entries/*; do
  if [ -f "$file" ]; then
    size=$(wc -c < "$file")
    echo "$file: $size bytes"
  fi
done
```

## Troubleshooting

### Server won't start

```bash
# Check if port is in use
lsof -i :8080

# Check config file
./scitt serve --config scitt.yaml --verbose

# Verify database
sqlite3 scitt.db "SELECT * FROM current_tree_size;"

# Check permissions
ls -la scitt.db storage/
```

### Statement registration fails

```bash
# Verify COSE Sign1 structure
./scitt statement verify --input statement.cbor --key issuer-key.pem

# Check statement hash
./scitt statement hash --input statement.cbor

# Verify server logs
# Look for error messages in server output
```

### Database errors

```bash
# Check WAL mode
sqlite3 scitt.db "PRAGMA journal_mode;"

# Rebuild database
rm scitt.db
./scitt init --origin https://transparency.example.com

# Verify schema version
sqlite3 scitt.db "SELECT version FROM schema_version;"
```

## Test Data Cleanup

```bash
# Remove test files
rm -f statement*.cbor payload*.json receipt.json checkpoint.txt
rm -f issuer-key.pem load-test.cbor
rm -f test-config.yaml

# Reset service (CAUTION: deletes all data)
rm -f scitt.db
rm -rf storage/
./scitt init --origin https://transparency.example.com
```

## Expected Test Results

### Success Criteria

- ✅ All unit tests pass (76+ test suites)
- ✅ Statement registration returns 202 Accepted with valid entry ID
- ✅ Receipt retrieval returns valid JSON with correct fields
- ✅ Checkpoint returns signed note format with valid signature
- ✅ Configuration loading works with default and custom paths
- ✅ CORS headers present when enabled
- ✅ Error responses include appropriate status codes
- ✅ Database operations maintain ACID properties
- ✅ Storage tiles created and readable
- ✅ Tree size increments correctly

### Known Limitations

1. **Receipt Generation**: Currently simplified, does not include full Merkle inclusion proof (T027)
2. **Tree Root Computation**: Uses placeholder root hash instead of actual Merkle tree computation
3. **Entry Tile Storage**: Stores individual leaves, not full tile format yet
4. **Verification**: CLI verification only checks signature, not tree inclusion

## Next Steps

1. Implement full Merkle inclusion proofs (T027)
2. Add contract tests for SCRAPI compliance (T026)
3. Implement complete receipt generation with proofs
4. Add integration tests for multi-component workflows
5. Add performance benchmarks

## Reference

- SCITT Architecture: [draft-ietf-scitt-architecture](https://datatracker.ietf.org/doc/draft-ietf-scitt-architecture/)
- SCRAPI: Supply Chain Repository API specification
- RFC 6962: Certificate Transparency
- RFC 9052: CBOR Object Signing and Encryption (COSE)
- C2SP tlog-tiles: https://c2sp.org/tlog-tiles
