#!/bin/bash
# Run E2E tests with real APIs
# Requires environment variables to be set (see test/e2e/config.go)

set -e

# Check for required env file
if [ -f .env.e2e ]; then
    echo "Loading .env.e2e..."
    source .env.e2e
fi

echo "Running E2E tests..."
echo "Note: Tests will be skipped if credentials are not set"
echo ""

go test -v -tags=e2e -timeout=10m ./test/e2e/...

echo ""
echo "E2E tests complete!"
