#!/bin/bash
# Run contract tests

set -e

echo "Running contract tests..."

go test -v ./test/contract/...

echo ""
echo "Contract tests complete!"
