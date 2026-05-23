#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_DIR="$(cd "${ROOT_DIR}/.." && pwd)"
SECRETS_DIR="${ROOT_DIR}/secrets"
ANDROID_RAW_DIR="${PROJECT_DIR}/mobile/app/src/main/res/raw"
STORE_PASSWORD="changeit"

mkdir -p "${ANDROID_RAW_DIR}"

required_files=(
    "${SECRETS_DIR}/ca.crt"
    "${SECRETS_DIR}/web.crt"
    "${SECRETS_DIR}/web.key"
)

for file in "${required_files[@]}"; do
    if [ ! -f "${file}" ]; then
        echo "[ERROR] Missing ${file}. Run ./scripts/gen-ca.sh first." >&2
        exit 1
    fi
done

cp "${SECRETS_DIR}/ca.crt" "${ANDROID_RAW_DIR}/ca.crt"

openssl pkcs12 -export \
    -in "${SECRETS_DIR}/web.crt" \
    -inkey "${SECRETS_DIR}/web.key" \
    -certfile "${SECRETS_DIR}/ca.crt" \
    -name android-client \
    -out "${ANDROID_RAW_DIR}/mtls_keystore.p12" \
    -passout "pass:${STORE_PASSWORD}"

echo "[SUCCES] Android PKCS#12 stores generated in ${ANDROID_RAW_DIR}"
