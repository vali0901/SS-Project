#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ ! -f "${ROOT_DIR}/.env" ]]; then
  echo "Missing .env in ${ROOT_DIR}. Copy or create it before starting the stack." >&2
  exit 1
fi

echo "Starting docker compose services..."
cd "${ROOT_DIR}"

docker compose up -d --build

echo "------------------------------------------------"
echo "Services are starting..."
echo "Vite Dev Server: http://localhost:5173"
echo "Go API: http://localhost:8080"
echo "------------------------------------------------"
echo "To view logs, run: docker compose logs -f"
