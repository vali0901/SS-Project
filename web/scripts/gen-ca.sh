#!/bin/bash

set -euo pipefail

# Setează directorul pentru secrete
SECRETS_DIR="secrets"
EMULATOR_IP="10.0.2.2"

server_cert_needs_regeneration() {
    if [ ! -f "$SECRETS_DIR/server.key" ] || [ ! -f "$SECRETS_DIR/server.crt" ]; then
        return 0
    fi

    if ! openssl x509 -in "$SECRETS_DIR/server.crt" -noout -ext subjectAltName 2>/dev/null | grep -q "IP Address:${EMULATOR_IP}"; then
        return 0
    fi

    if ! openssl x509 -in "$SECRETS_DIR/server.crt" -noout -ext subjectAltName 2>/dev/null | grep -q "DNS:go-api"; then
        return 0
    fi

    return 1
}

SERVER_CERT_NEEDS_REGENERATION=false
if server_cert_needs_regeneration; then
    SERVER_CERT_NEEDS_REGENERATION=true
fi

# Verifică dacă fișierele finale există deja și includ SAN-ul pentru emulatorul Android
if [ -f "$SECRETS_DIR/ca.crt" ] && [ -f "$SECRETS_DIR/server.crt" ] && [ -f "$SECRETS_DIR/web.crt" ] && [ -f "$SECRETS_DIR/db.crt" ] && [ "$SERVER_CERT_NEEDS_REGENERATION" = false ]; then
    echo "[INFO] Toate certificatele (CA, Server, Client, DB) există deja în '$SECRETS_DIR/' și includ SAN-urile necesare. Generarea a fost anulată."
    exit 0
fi

echo "[INFO] Certificatele lipsesc sau sunt incomplete. Se începe generarea..."

# Creează directorul dacă nu există și intră în el
mkdir -p "$SECRETS_DIR"
cd "$SECRETS_DIR" || exit 1

# ---------------------------------------------------------------------
# 1. Generarea Certificate Authority (CA)
# ---------------------------------------------------------------------
if [ ! -f "ca.key" ] || [ ! -f "ca.crt" ]; then
    echo "[+] Generare CA (Cheie și Certificat)..."
    openssl genrsa -out ca.key 4096
    openssl req -new -x509 -days 3650 -key ca.key -out ca.crt \
        -subj "/C=RO/ST=Romania/L=Bucharest/O=SS-Web/OU=Security/CN=SS-Web-CA"
else
    echo "[~] CA există deja. Se trece mai departe."
fi

# ---------------------------------------------------------------------
# 2. Generarea Certificatului pentru Server (Cu SAN)
# ---------------------------------------------------------------------
if [ ! -f "server.key" ] || [ ! -f "server.crt" ] || [ "$SERVER_CERT_NEEDS_REGENERATION" = true ]; then
    echo "[+] Generare Certificat Server (cu SAN)..."
    chmod u+w server.key server.crt 2>/dev/null || true
    openssl genrsa -out server.key 2048
    openssl req -new -key server.key -out server.csr \
        -subj "/C=RO/ST=Romania/L=Bucharest/O=SS-Web/OU=Broker/CN=broker"

    # Creare fișier temporar de configurare pentru extensia SAN
    # Adaugă aici IP-ul sau DNS-ul real dacă nu rulezi pe localhost (ex: DNS.2 = broker.local)
    cat <<EOF > server_ext.cnf
subjectAltName = @alt_names

[alt_names]
DNS.1 = broker
DNS.2 = localhost
DNS.3 = go-api
IP.1 = 127.0.0.1
IP.2 = 10.0.2.2
EOF

    # Semnarea certificatului folosind fișierul de extensie
    openssl x509 -req -days 365 -in server.csr -CA ca.crt -CAkey ca.key \
        -CAcreateserial -out server.crt -extfile server_ext.cnf

    # Curățare fișiere temporare
    rm -f server.csr server_ext.cnf
else
    echo "[~] Certificatul de server există deja."
fi

# ---------------------------------------------------------------------
# 3. Generarea Certificatului pentru Client (Web - Cu SAN)
# ---------------------------------------------------------------------
if [ ! -f "web.key" ] || [ ! -f "web.crt" ]; then
    echo "[+] Generare Certificat Client (Web cu SAN)..."
    chmod u+w web.key web.crt 2>/dev/null || true
    openssl genrsa -out web.key 2048
    openssl req -new -key web.key -out web.csr \
        -subj "/C=RO/ST=Romania/L=Bucharest/O=SS-Web/OU=WebClient/CN=web"

    # Creare fișier temporar de configurare pentru SAN client
    cat <<EOF > web_ext.cnf
subjectAltName = @alt_names

[alt_names]
DNS.1 = web
DNS.2 = localhost
IP.1 = 127.0.0.1
EOF

    # Semnarea certificatului client
    openssl x509 -req -days 365 -in web.csr -CA ca.crt -CAkey ca.key \
        -CAcreateserial -out web.crt -extfile web_ext.cnf

    # Curățare fișiere temporare
    rm -f web.csr web_ext.cnf
else
    echo "[~] Certificatul de client există deja."
fi

# ---------------------------------------------------------------------
# 4. Generarea Certificatului pentru OCR (Cu SAN)
# ---------------------------------------------------------------------
if [ ! -f "ocr.key" ] || [ ! -f "ocr.crt" ]; then
    echo "[+] Generare Certificat OCR (cu SAN)..."
    chmod u+w ocr.key ocr.crt 2>/dev/null || true
    openssl genrsa -out ocr.key 2048
    openssl req -new -key ocr.key -out ocr.csr \
        -subj "/C=RO/ST=Romania/L=Bucharest/O=SS-Web/OU=OCR/CN=ocr-service"

    # Creare fișier temporar de configurare pentru extensia SAN
    # Adaugă aici IP-ul sau DNS-ul real dacă nu rulezi pe localhost (ex: DNS.2 = broker.local)
    cat <<EOF > ocr_ext.cnf
subjectAltName = @alt_names

[alt_names]
DNS.1 = ocr-service
DNS.2 = localhost
IP.1 = 127.0.0.1
EOF

    # Semnarea certificatului folosind fișierul de extensie
    openssl x509 -req -days 365 -in ocr.csr -CA ca.crt -CAkey ca.key \
        -CAcreateserial -out ocr.crt -extfile ocr_ext.cnf

    # Curățare fișiere temporare
    rm -f ocr.csr ocr_ext.cnf
else
    echo "[~] Certificatul de OCR există deja."
fi

# ---------------------------------------------------------------------
# 5. Generarea Certificatului pentru PostgreSQL (Cu SAN)
# ---------------------------------------------------------------------
if [ ! -f "db.key" ] || [ ! -f "db.crt" ]; then
    echo "[+] Generare Certificat PostgreSQL (cu SAN)..."
    chmod u+w db.key db.crt 2>/dev/null || true
    openssl genrsa -out db.key 2048
    openssl req -new -key db.key -out db.csr \
        -subj "/C=RO/ST=Romania/L=Bucharest/O=SS-Web/OU=Database/CN=postgres-db"

    cat <<EOF > db_ext.cnf
subjectAltName = @alt_names

[alt_names]
DNS.1 = postgres-db
DNS.2 = localhost
IP.1 = 127.0.0.1
EOF

    openssl x509 -req -days 365 -in db.csr -CA ca.crt -CAkey ca.key \
        -CAcreateserial -out db.crt -extfile db_ext.cnf

    rm -f db.csr db_ext.cnf
else
    echo "[~] Certificatul PostgreSQL există deja."
fi

# ---------------------------------------------------------------------
# 6. Verificarea Certificatelor Generate
# ---------------------------------------------------------------------
echo -e "\n--- Verificare Certificate ---"
openssl verify -CAfile ca.crt server.crt
openssl verify -CAfile ca.crt web.crt
openssl verify -CAfile ca.crt ocr.crt
openssl verify -CAfile ca.crt db.crt

# pt ocr
chmod 444 *.crt
chmod 444 *.key

# generare key pt criptare in db
if [ ! -f "encryption.key" ]; then
    echo "[+] Generare cheie de criptare pentru baza de date..."
    openssl rand -out encryption.key 32
else
    echo "[~] Cheia de criptare există deja."
fi

echo "[SUCCES] Procesul a fost finalizat cu succes!"
