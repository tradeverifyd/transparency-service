#!/bin/bash
# Helper script to manage MongoDB test container

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

case "$1" in
  start)
    echo "Starting MongoDB test container..."
    docker-compose -f "$PROJECT_DIR/docker-compose.test.yml" up -d
    echo "Waiting for MongoDB to be healthy..."
    sleep 5
    docker ps | grep scitt-mongodb-test
    echo "✓ MongoDB is ready at mongodb://scitt_test:scitt_test_password@localhost:27017"
    ;;

  stop)
    echo "Stopping MongoDB test container..."
    docker-compose -f "$PROJECT_DIR/docker-compose.test.yml" down
    echo "✓ MongoDB stopped"
    ;;

  restart)
    echo "Restarting MongoDB test container..."
    docker-compose -f "$PROJECT_DIR/docker-compose.test.yml" restart
    echo "✓ MongoDB restarted"
    ;;

  clean)
    echo "Stopping and removing MongoDB test container and volumes..."
    docker-compose -f "$PROJECT_DIR/docker-compose.test.yml" down -v
    echo "✓ MongoDB cleaned"
    ;;

  logs)
    docker-compose -f "$PROJECT_DIR/docker-compose.test.yml" logs -f
    ;;

  test)
    echo "Running MongoDB repository tests..."
    export MONGODB_URI='mongodb://scitt_test:scitt_test_password@localhost:27017'
    export MONGODB_DATABASE='scitt_test'
    cd "$PROJECT_DIR"
    go test -v ./pkg/repository -run "TestMongoDB" -timeout 60s
    ;;

  *)
    echo "Usage: $0 {start|stop|restart|clean|logs|test}"
    echo ""
    echo "Commands:"
    echo "  start   - Start MongoDB test container"
    echo "  stop    - Stop MongoDB test container"
    echo "  restart - Restart MongoDB test container"
    echo "  clean   - Stop and remove container and volumes"
    echo "  logs    - Show MongoDB logs"
    echo "  test    - Run MongoDB repository tests"
    exit 1
    ;;
esac
