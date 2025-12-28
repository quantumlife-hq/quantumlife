#!/bin/bash
# Run all tests (unit, integration, contract)
# Does NOT run E2E tests - use run-e2e.sh for those

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "========================================="
echo "Running all tests"
echo "========================================="
echo ""

echo ">>> Unit Tests"
"$SCRIPT_DIR/run-unit.sh"
echo ""

echo ">>> Contract Tests"
"$SCRIPT_DIR/run-contract.sh"
echo ""

echo ">>> Integration Tests"
"$SCRIPT_DIR/run-integration.sh"
echo ""

echo "========================================="
echo "All tests passed!"
echo "========================================="
