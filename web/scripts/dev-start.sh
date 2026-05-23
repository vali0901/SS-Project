#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "${ROOT_DIR}/.." && pwd)"
README_FILE="${REPO_ROOT}/README.md"

START_MARKER="<!-- SBOM:START -->"
END_MARKER="<!-- SBOM:END -->"

update_sbom_readme() {
  local tmp_server tmp_ocr tmp_section tmp_readme

  tmp_server="$(mktemp)"
  tmp_ocr="$(mktemp)"
  tmp_section="$(mktemp)"
  tmp_readme="$(mktemp)"

  cleanup() {
    rm -f "${tmp_server}" "${tmp_ocr}" "${tmp_section}" "${tmp_readme}"
  }
  trap cleanup RETURN

  run_go_list() {
    local dir="$1"

    if command -v go >/dev/null 2>&1; then
      (
        cd "${dir}"
        go list -m -f '{{if not .Main}}{{.Path}}|{{.Version}}{{end}}' all
      )
      return
    fi

    docker run --rm \
      -v "${dir}:/src" \
      -w /src \
      golang:1.25-alpine \
      sh -c "go list -m -f '{{if not .Main}}{{.Path}}|{{.Version}}{{end}}' all"
  }

  write_table() {
    local title="$1"
    local path_label="$2"
    local input_file="$3"
    local count

    count="$(grep -c '.' "${input_file}" || true)"

    {
      echo "### ${title}"
      echo
      echo "Source: ${path_label}"
      echo
      echo "Total dependencies: ${count}"
      echo
      echo "| Dependency | Version |"
      echo "|---|---|"
      if [[ "${count}" -eq 0 ]]; then
        echo "| (none) | - |"
      else
        awk -F'|' '{printf "| %s | %s |\n", $1, $2}' "${input_file}"
      fi
      echo
    } >> "${tmp_section}"
  }

  run_go_list "${ROOT_DIR}/server" | sort > "${tmp_server}"
  run_go_list "${ROOT_DIR}/ocr-service" | sort > "${tmp_ocr}"

  {
    echo "${START_MARKER}"
    echo "## SBOM Snapshot"
    echo
    echo "This section is auto-generated during build by web/scripts/dev-start.sh."
    echo
    echo "Last updated: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
    echo
  } > "${tmp_section}"

  write_table "Go Backend Server" "web/server/go.mod" "${tmp_server}"
  write_table "OCR Service" "web/ocr-service/go.mod" "${tmp_ocr}"

  echo "${END_MARKER}" >> "${tmp_section}"

  if grep -q "${START_MARKER}" "${README_FILE}" && grep -q "${END_MARKER}" "${README_FILE}"; then
    awk -v start="${START_MARKER}" -v end="${END_MARKER}" -v repl="${tmp_section}" '
      BEGIN {
        while ((getline line < repl) > 0) {
          block = block line "\n"
        }
        close(repl)
        in_block = 0
        replaced = 0
      }
      $0 == start {
        if (!replaced) {
          printf "%s", block
          replaced = 1
        }
        in_block = 1
        next
      }
      $0 == end {
        in_block = 0
        next
      }
      !in_block {
        print
      }
      END {
        if (!replaced) {
          printf "\n%s", block
        }
      }
    ' "${README_FILE}" > "${tmp_readme}"
  else
    cat "${README_FILE}" > "${tmp_readme}"
    echo >> "${tmp_readme}"
    cat "${tmp_section}" >> "${tmp_readme}"
  fi

  mv "${tmp_readme}" "${README_FILE}"
}

if [[ ! -f "${ROOT_DIR}/.env" ]]; then
  echo "Missing .env in ${ROOT_DIR}. Copy or create it before starting the stack." >&2
  exit 1
fi

echo "Updating SBOM snapshot in root README..."
update_sbom_readme

echo "Starting docker compose services..."
cd "${ROOT_DIR}"

docker compose up -d --build

echo "------------------------------------------------"
echo "Services are starting..."
echo "Vite Dev Server: http://localhost:5173"
echo "Go API: http://localhost:8080"
echo "------------------------------------------------"
echo "To view logs, run: docker compose logs -f"
