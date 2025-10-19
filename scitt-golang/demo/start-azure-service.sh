#!/bin/bash
# Start SCITT service with Azure Blob Storage + MongoDB configuration

set -e

# Load environment variables
if [ -f .env ]; then
    set -a
    source .env
    set +a
else
    echo "Error: .env file not found"
    exit 1
fi

# Start service
exec ./scitt service start --definition ./demo/scitt-azure.yaml --verbose
