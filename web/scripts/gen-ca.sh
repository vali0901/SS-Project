#!/bin/bash

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

    return 1
}

SERVER_CERT_NEEDS_REGENERATION=false
if server_cert_needs_regeneration; then
    SERVER_CERT_NEEDS_REGENERATION=true
fi

# Verifică dacă fișierele finale există deja și includ SAN-ul pentru emulatorul Android
if [ -f "$SECRETS_DIR/ca.crt" ] && [ -f "$SECRETS_DIR/server.crt" ] && [ -f "$SECRETS_DIR/web.crt" ] && [ "$SERVER_CERT_NEEDS_REGENERATION" = false ]; then
    echo "[INFO] Toate certificatele (CA, Server, Client) există deja în '$SECRETS_DIR/' și includ SAN-ul pentru emulator. Generarea a fost anulată."
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
# 4. Verificarea Certificatelor Generate
# ---------------------------------------------------------------------
echo -e "\n--- Verificare Certificate ---"
openssl verify -CAfile ca.crt server.crt
openssl verify -CAfile ca.crt web.crt

echo "[SUCCES] Procesul a fost finalizat cu succes!"
