#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "============================================"
echo "  Restarting Development Environment"
echo "============================================"

# Stop everything
./scripts/dev-stop.sh

# Start everything
./scripts/dev-start.sh

echo "============================================"
echo "  Restart Complete"
echo "============================================"