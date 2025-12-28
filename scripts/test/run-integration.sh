#!/bin/bash
# Run integration tests with mock servers

set -e

echo "Running integration tests..."
go test -v -tags=integration ./internal/mcp/servers/...

echo ""
echo "Integration tests complete!"
