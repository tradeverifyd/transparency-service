# Azure Blob Storage Configuration for SCITT

This guide explains how to configure SCITT to use Azure Blob Storage for storing Merkle tree tiles.

## Overview

The SCITT transparency service can store Merkle tree tiles in Azure Blob Storage, enabling:
- **Scalable storage**: Leverage Azure's cloud infrastructure for tile storage
- **Geo-redundancy**: Benefit from Azure's built-in redundancy and replication
- **Cost-effective**: Pay only for storage used
- **Secure access**: Use SAS tokens for time-limited, permission-scoped access

## Prerequisites

1. **Azure Storage Account**: Create a storage account in Azure Portal
2. **Blob Container**: Create a container within your storage account (e.g., `scitt-tiles`)
3. **Authentication**: Choose either SAS URL (recommended) or Account Key

## Authentication Methods

### Option 1: SAS URL (Recommended)

Shared Access Signature (SAS) provides secure, time-limited access without exposing account keys.

**Generate SAS Token via Azure Portal:**
1. Navigate to your Storage Account → Container
2. Click "Shared access tokens"
3. Set permissions: **Read**, **Write**, **List**, **Add**
4. Set expiration time (e.g., 1 year)
5. Click "Generate SAS token and URL"
6. Copy the **Blob SAS URL**

**Generate SAS Token via Azure CLI:**
```bash
# Set variables
STORAGE_ACCOUNT="youraccountname"
CONTAINER="scitt-tiles"
EXPIRY=$(date -u -d "1 year" '+%Y-%m-%dT%H:%MZ')

# Generate SAS token
az storage container generate-sas \
  --account-name $STORAGE_ACCOUNT \
  --name $CONTAINER \
  --permissions rwla \
  --expiry $EXPIRY \
  --https-only \
  --output tsv

# Get full SAS URL
az storage container generate-sas \
  --account-name $STORAGE_ACCOUNT \
  --name $CONTAINER \
  --permissions rwla \
  --expiry $EXPIRY \
  --https-only \
  --as-user \
  --auth-mode login \
  --output tsv
```

### Option 2: Account Name + Key

Less secure but simpler for development environments.

**Get Account Key:**
```bash
az storage account keys list \
  --account-name youraccountname \
  --query '[0].value' \
  --output tsv
```

## Configuration

### Environment Variables (.env file)

**Using SAS URL (Recommended):**
```bash
# Azure Blob Storage - SAS URL format
SCITT_AZURE_SAS_URL=https://youraccountname.blob.core.windows.net/scitt-tiles?sp=rwla&st=2025-01-01T00:00:00Z&se=2026-01-01T00:00:00Z&sv=2023-01-03&sr=c&sig=YOUR_SIGNATURE_HERE
SCITT_AZURE_CONTAINER=scitt-tiles

# Azure Cosmos DB (MongoDB API)
SCITT_MONGODB_URI=mongodb+srv://username:password@your-cosmos-account.mongo.cosmos.azure.com/?tls=true&authMechanism=SCRAM-SHA-256&retrywrites=false&maxIdleTimeMS=120000
SCITT_MONGODB_DATABASE=scitt_demo

# Service API Key
SCITT_API_KEY=your-generated-api-key-here
```

**Using Account Key:**
```bash
# Azure Blob Storage - Account Key format
SCITT_AZURE_ACCOUNT_NAME=youraccountname
SCITT_AZURE_ACCOUNT_KEY=your-storage-account-key-here
SCITT_AZURE_CONTAINER=scitt-tiles

# Azure Cosmos DB (MongoDB API)
SCITT_MONGODB_URI=mongodb+srv://username:password@your-cosmos-account.mongo.cosmos.azure.com/?tls=true&authMechanism=SCRAM-SHA-256&retrywrites=false&maxIdleTimeMS=120000
SCITT_MONGODB_DATABASE=scitt_demo

# Service API Key
SCITT_API_KEY=your-generated-api-key-here
```

### YAML Configuration (scitt-azure-cosmos.yaml)

```yaml
issuer: http://127.0.0.1:56177

database:
  type: mongodb
  mongodb:
    uri: ${SCITT_MONGODB_URI}
    database: ${SCITT_MONGODB_DATABASE}

storage:
  type: azure
  azure:
    container: ${SCITT_AZURE_CONTAINER}
    sas_url: ${SCITT_AZURE_SAS_URL}

    # OR use account key (less secure):
    # account_name: ${SCITT_AZURE_ACCOUNT_NAME}
    # account_key: ${SCITT_AZURE_ACCOUNT_KEY}

keys:
  private: ./demo/priv.cbor
  public: ./demo/pub.cbor

server:
  host: 127.0.0.1
  port: 56177
  api_key: ${SCITT_API_KEY}
```

## Usage

### Start Service

```bash
# Load environment variables and start service
./scitt service start --definition ./demo/scitt-azure-cosmos.yaml --verbose
```

### Register a Statement

```bash
# Create and sign a statement
./scitt statement sign \
  --content ./demo/test.parquet \
  --content-type application/vnd.apache.parquet \
  --content-location https://datasets.example.com/test.parquet \
  --issuer "https://example.com" \
  --subject "urn:example:dataset:2025" \
  --signing-key ./demo/priv.cbor \
  --signed-statement ./demo/statement.cbor

# Register with transparency service
./scitt statement register \
  --service http://127.0.0.1:56177 \
  --api-key "${SCITT_API_KEY}" \
  --statement ./demo/statement.cbor \
  --receipt ./demo/receipt.cbor
```

## Verify Setup

### Check Azure Blob Storage

After registering statements, verify tiles are being stored:

```bash
# List blobs in container
az storage blob list \
  --account-name youraccountname \
  --container-name scitt-tiles \
  --output table

# Expected output: tile/entries/x00/000 (and similar tile files)
```

### Check Azure Cosmos DB

Verify statement metadata is persisted:

```bash
# Using mongosh
mongosh "${SCITT_MONGODB_URI}"

# List collections
use scitt_demo
show collections

# Query statements
db.statements.find().pretty()
```

## Tile Storage Format

SCITT uses the [C2SP tlog-tiles](https://c2sp.org/tlog-tiles) format:
- **Entry Tiles**: Store leaf hashes at `tile/entries/x{level}/{index}`
- **Data Tiles**: Store intermediate nodes at `tile/data/x{level}/{index}`

Example tile paths in Azure Blob Storage:
```
tile/entries/x00/000    # First entry tile (entries 0-255)
tile/entries/x00/001    # Second entry tile (entries 256-511)
tile/data/x01/000       # First data tile (level 1)
```

## Performance Considerations

1. **Region**: Use same Azure region for Storage Account and Cosmos DB to minimize latency
2. **Redundancy**: Choose appropriate redundancy level (LRS, GRS, RA-GRS)
3. **Performance Tier**: Use Standard tier for tiles (Premium not necessary)
4. **SAS Token Expiration**: Set reasonable expiration time and rotate tokens regularly

## Security Best Practices

1. **Use SAS URLs**: Prefer SAS tokens over account keys
2. **Minimal Permissions**: Grant only required permissions (Read, Write, List, Add)
3. **Short Expiration**: Set appropriate expiration times for SAS tokens
4. **Rotate Credentials**: Regularly rotate SAS tokens and account keys
5. **Network Access**: Consider using Azure Private Link for enhanced security
6. **Monitoring**: Enable Azure Storage Analytics and monitoring

## Troubleshooting

### Error: "failed to access container"

**Possible causes:**
- Container doesn't exist
- SAS token expired
- Insufficient permissions on SAS token
- Account key is incorrect

**Solution:**
```bash
# Verify container exists
az storage container exists \
  --account-name youraccountname \
  --name scitt-tiles

# Create container if missing
az storage container create \
  --account-name youraccountname \
  --name scitt-tiles \
  --public-access off
```

### Error: "BlobNotFound" or "404"

**Cause:** Trying to read a tile that doesn't exist yet (normal for new service)

**Solution:** This is expected behavior - tiles are created on-demand as statements are registered.

## Cost Estimation

**Storage Costs** (Example using Azure Blob Storage Standard LRS):
- Storage: ~$0.018 per GB/month
- Write operations: ~$0.05 per 10,000 operations
- Read operations: ~$0.004 per 10,000 operations

**Example:**
- 10,000 statements/month
- ~32 bytes per tile entry
- ~10 KB metadata per statement in Cosmos DB
- Estimated cost: < $1/month for storage + operations

## Additional Resources

- [Azure Blob Storage Documentation](https://learn.microsoft.com/en-us/azure/storage/blobs/)
- [Azure Cosmos DB for MongoDB](https://learn.microsoft.com/en-us/azure/cosmos-db/mongodb/)
- [C2SP tlog-tiles Specification](https://c2sp.org/tlog-tiles)
- [SCITT Architecture RFC 9597](https://datatracker.ietf.org/doc/rfc9597/)
