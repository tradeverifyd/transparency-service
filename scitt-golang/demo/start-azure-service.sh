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

# Port to use
PORT=56177

# Kill any process using the port
echo "Checking for processes on port $PORT..."
PID=$(lsof -ti:$PORT 2>/dev/null || true)
if [ -n "$PID" ]; then
    echo "Killing process $PID on port $PORT..."
    kill -9 $PID
    sleep 1
fi

# Start service
exec ./scitt service start --definition ./demo/scitt-azure.yaml --verbose
