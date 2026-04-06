#!/bin/bash

ADRESA_IP_BROKER="0.0.0.0"

mkdir -p certs && cd certs
openssl genrsa -out ca.key 2048

openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.crt \
    -subj "/CN=SS-Project-CA"

openssl genrsa -out server.key 2048

openssl req -new -key server.key -out server.csr \
    -subj "/CN=$ADRESA_IP_BROKER" \
    -addext "subjectAltName=IP:$ADRESA_IP_BROKER"

openssl x509 -req -in server.csr \
    -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out server.crt -days 365 -sha256 \
    -copy_extensions copyall

openssl x509 -in server.crt -text -noout | grep -A1 "Subject Alternative Name"
