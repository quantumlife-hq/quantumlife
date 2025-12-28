#!/bin/bash
# Run unit tests with coverage

set -e

echo "Running unit tests..."
go test -v -race -coverprofile=coverage.out \
    ./internal/mcp/server/... \
    ./internal/mcp/servers/... \
    ./internal/api/... \
    ./internal/learning/... \
    ./internal/proactive/... \
    ./test/...

echo ""
echo "Coverage summary:"
go tool cover -func=coverage.out | tail -1

echo ""
echo "To view detailed coverage: go tool cover -html=coverage.out"
