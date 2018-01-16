#!/bin/bash
set -eux

ca_subject='/CN=Test CA'
domains=(
    'producer.example.com'
    'consumer.example.com'
)

# create the CA keypair and a self-signed certificate.
openssl genrsa -out ca-key.pem 2048
openssl req -new \
    -sha256 \
    -subj "$ca_subject" \
    -key ca-key.pem \
    -out ca-csr.pem
openssl x509 -req \
    -sha256 \
    -signkey ca-key.pem \
    -extensions a \
    -extfile <(echo '[a]
        basicConstraints=critical,CA:TRUE,pathlen:0
    ') \
    -days 3650 \
    -in  ca-csr.pem \
    -out ca-crt.pem
rm ca-csr.pem
# create the domains keypairs and their certificates signed by the test CA.
for domain in "${domains[@]}"; do
    openssl genrsa \
        -out "$domain-key.pem" \
        2048 \
        2>/dev/null
    openssl req -new \
        -sha256 \
        -subj "/CN=$domain" \
        -key "$domain-key.pem" \
        -out "$domain-csr.pem"
    openssl x509 -req -sha256 \
        -CA ca-crt.pem \
        -CAkey ca-key.pem \
        -set_serial 1 \
        -extensions a \
        -extfile <(echo "[a]
            subjectAltName=DNS:$domain
            extendedKeyUsage=serverAuth
            ") \
        -days 3650 \
        -in  "$domain-csr.pem" \
        -out "$domain-crt.pem"
    rm "$domain-csr.pem"
    openssl x509 -noout -text -in "$domain-crt.pem"
    short=${domain%\.example\.com}
    cat "$domain-key.pem" "$domain-crt.pem" > "$short.pem"
done
set +x
for domain in "${domains[@]}"; do
    openssl verify -CAfile ca-crt.pem "$domain-crt.pem"
done
