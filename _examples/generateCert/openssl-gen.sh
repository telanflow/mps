#!/bin/bash
set -ex
# generate CA's  key
openssl genrsa -aes256 -passout pass:1 -out ca.key 4096
openssl rsa -passin pass:1 -in ca.key -out ca.key.tmp
mv ca.key.tmp ca.key

openssl req -config openssl.cnf -key ca.key -new -x509 -days 7300 -sha256 -extensions v3_ca -out ca.crt
