#!/bin/bash

set -e

CERT_DIR="certs"
CA_KEY="$CERT_DIR/ca.key"
CA_CERT="$CERT_DIR/ca.pem"
CERT_CSR="$CERT_DIR/cert.csr"
PRIVATE_KEY="$CERT_DIR/private.key"
CERT="$CERT_DIR/cert.pem"

mkdir -p "$CERT_DIR"

# Generate a CA key and self-signed certificate
openssl req -x509 -sha256 -nodes -days 3650 -newkey rsa:2048 \
    -keyout "$CA_KEY" -out "$CA_CERT" \
    -subj "/CN=Fake CA For Testing/"

# Generate a certificate signing request (CSR) and private key
openssl req -out "$CERT_CSR" -new -newkey rsa:2048 -nodes -keyout "$PRIVATE_KEY" \
    -subj "/CN=Localhost Testing Certificate/" -addext "subjectAltName=DNS:localhost"

# Sign the CSR with the CA to generate the final certificate
openssl x509 -req -sha256 -days 3650 -in "$CERT_CSR" -out "$CERT" \
    -CA "$CA_CERT" -CAkey "$CA_KEY" -CAcreateserial \
    -extfile <(printf "subjectAltName=DNS:localhost")

# Display certificate information
openssl x509 -noout -text -in "$CERT"

echo "Certificate generation completed successfully!"
