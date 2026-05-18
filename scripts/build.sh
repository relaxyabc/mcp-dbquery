#!/bin/bash

# MCP Database Query Tool - Build Script

set -e

echo "Building MCP Database Query Tool..."

# Build binary
go build -o bin/db-query-server ./src/server

echo "Build complete: bin/db-query-server"

# Optional: Run tests
if [ "$1" == "test" ]; then
    echo "Running tests..."
    go test ./tests/... -v
fi

# Optional: Run with Docker test containers
if [ "$1" == "test-integration" ]; then
    echo "Running integration tests..."
    ./scripts/docker-test.sh up
    go test ./tests/integration/... -v
    ./scripts/docker-test.sh down
fi