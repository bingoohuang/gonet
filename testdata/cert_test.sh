#!/usr/bin/env bash

cd v3

openssl genrsa -out root.key 2048
openssl req -new -nodes -x509 -days 3650 -key root.key -out root.pem -subj "/C=CN/ST=BEIJING/L=Earth/O=BJCA/OU=IT/CN=root"

openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -subj "/C=CN/ST=BEIJING/L=Earth/O=BJCA/OU=IT/CN=server"
openssl x509 -req -in server.csr -CA root.pem -CAkey root.key -CAcreateserial -out server.pem -days 3650

openssl genrsa -out client.key 2048
openssl req -new -key client.key -subj "/C=CN/ST=BEIJING/L=Earth/O=BJCA/OU=IT/CN=client" -out client.csr
echo "[ssl_client]"> openssl.cnf; echo "extendedKeyUsage = clientAuth" >> openssl.cnf;
openssl x509 -req -in client.csr -CA root.pem -CAkey root.key -CAcreateserial -extfile ./openssl.cnf -out client.pem -days 3650