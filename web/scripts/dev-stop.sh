#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "Stopping all services via docker compose..."
cd "${ROOT_DIR}"
docker compose down

echo "All services stopped."