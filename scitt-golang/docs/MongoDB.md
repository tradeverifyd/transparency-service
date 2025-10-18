# MongoDB Setup for SCITT Golang

This guide covers setting up MongoDB for local development and testing.

## Quick Start (Docker)

### Start MongoDB Test Container

```bash
./scripts/test-mongodb.sh start
```

This starts a MongoDB 7.0 container with:
- Port: `27017`
- Username: `scitt_test`
- Password: `scitt_test_password`
- Database: `scitt_test`

### Run MongoDB Tests

```bash
./scripts/test-mongodb.sh test
```

### Stop MongoDB

```bash
./scripts/test-mongodb.sh stop
```

### Clean MongoDB (Remove Data)

```bash
./scripts/test-mongodb.sh clean
```

## Environment Configuration

### Local Development (Docker)

Create or update `.env`:

```bash
MONGODB_URI=mongodb://scitt_test:scitt_test_password@localhost:27017
MONGODB_DATABASE=scitt_test
```

### Production (Azure Cosmos DB)

```bash
MONGODB_URI=mongodb+srv://username:password@host/?tls=true&authMechanism=SCRAM-SHA-256&retrywrites=false&maxIdleTimeMS=120000
MONGODB_DATABASE=transparency_service
```

## Manual Docker Setup

If you prefer to run Docker commands directly:

```bash
# Start
docker-compose -f docker-compose.test.yml up -d

# Stop
docker-compose -f docker-compose.test.yml down

# Clean (remove volumes)
docker-compose -f docker-compose.test.yml down -v

# View logs
docker-compose -f docker-compose.test.yml logs -f
```

## Running Tests

### All Repository Tests

```bash
go test -v ./pkg/repository/...
```

### Only MongoDB Tests

```bash
export MONGODB_URI='mongodb://scitt_test:scitt_test_password@localhost:27017'
export MONGODB_DATABASE='scitt_test'
go test -v ./pkg/repository -run "TestMongoDB"
```

### Only SQLite Tests

```bash
go test -v ./pkg/repository -run "TestSQLite"
```

### Only In-Memory Tests

```bash
go test -v ./pkg/repository -run "TestMemory"
```

## Database Collections

### `statements`
Stores statement metadata for efficient querying:
- `entry_id`: Unique auto-incrementing ID
- `leaf_hash`: Hash stored in Merkle tree tiles (unique)
- `iss`, `sub`, `cty`, `typ`: Query fields
- `payload_hash_alg`, `payload_hash`: Payload identifiers
- `registered_at`: Timestamp
- `tree_size_at_registration`: Tree size when registered
- `entry_tile_key`, `entry_tile_offset`: Tile storage location

### `tree_size`
Singleton collection tracking current tree size:
- `_id`: Always "current"
- `tree_size`: Current Merkle tree size
- `entry_id_counter`: Auto-increment counter for statements
- `last_updated`: Last update timestamp

### `tree_states`
Checkpoint history:
- `tree_size`: Tree size at checkpoint (unique)
- `root_hash`: Merkle root hash
- `updated_at`: Checkpoint timestamp

## Indexes

Automatically created for efficient queries:
- `statements.leaf_hash`: Unique index
- `statements.iss`: Query by issuer
- `statements.sub`: Query by subject
- `statements.cty`: Query by content type
- `statements.typ`: Query by type
- `statements.registered_at`: Query by date (descending)
- `tree_states.tree_size`: Unique checkpoint index

## Connection String Formats

### Local MongoDB (No Auth)
```
mongodb://localhost:27017
```

### Local MongoDB (With Auth)
```
mongodb://username:password@localhost:27017
```

### MongoDB Atlas
```
mongodb+srv://username:password@cluster.mongodb.net
```

### Azure Cosmos DB
```
mongodb+srv://username:password@host/?tls=true&authMechanism=SCRAM-SHA-256&retrywrites=false&maxIdleTimeMS=120000
```

## Troubleshooting

### Connection Timeout

If tests fail with timeout, ensure MongoDB is running:

```bash
docker ps | grep mongodb
```

### Port Already in Use

If port 27017 is in use, modify `docker-compose.test.yml`:

```yaml
ports:
  - "27018:27017"  # Use different local port
```

Then update `.env`:

```bash
MONGODB_URI=mongodb://scitt_test:scitt_test_password@localhost:27018
```

### Clear Test Data

```bash
./scripts/test-mongodb.sh clean
./scripts/test-mongodb.sh start
```

## Production Deployment

In production, set environment variables directly (don't use `.env` file):

```bash
export MONGODB_URI="mongodb+srv://..."
export MONGODB_DATABASE="transparency_service"
```

For Kubernetes, use secrets:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mongodb-secret
type: Opaque
stringData:
  uri: mongodb+srv://...
  database: transparency_service
```
