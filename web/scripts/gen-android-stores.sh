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
    "${SECRETS_DIR}/server.crt"
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

SERVER_PIN="$(openssl x509 -in "${SECRETS_DIR}/server.crt" -pubkey -noout \
    | openssl pkey -pubin -outform der \
    | openssl dgst -sha256 -binary \
    | openssl base64)"

cat > "${PROJECT_DIR}/mobile/app/src/main/res/xml/network_security_config.xml" <<EOF
<?xml version="1.0" encoding="utf-8"?>
<network-security-config>
    <base-config cleartextTrafficPermitted="false">
        <trust-anchors>
            <certificates src="@raw/ca" />
        </trust-anchors>
    </base-config>

    <domain-config cleartextTrafficPermitted="false">
        <domain includeSubdomains="false">10.0.2.2</domain>
        <domain includeSubdomains="false">127.0.0.1</domain>
        <domain includeSubdomains="false">localhost</domain>
        <domain includeSubdomains="false">broker</domain>
        <pin-set expiration="2036-05-21">
            <pin digest="SHA-256">${SERVER_PIN}</pin>
        </pin-set>
    </domain-config>
</network-security-config>
EOF

openssl pkcs12 -export \
    -in "${SECRETS_DIR}/web.crt" \
    -inkey "${SECRETS_DIR}/web.key" \
    -certfile "${SECRETS_DIR}/ca.crt" \
    -name android-client \
    -out "${ANDROID_RAW_DIR}/mtls_keystore.p12" \
    -passout "pass:${STORE_PASSWORD}"

echo "[SUCCES] Android PKCS#12 stores generated in ${ANDROID_RAW_DIR}"
