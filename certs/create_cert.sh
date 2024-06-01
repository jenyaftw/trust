#!/bin/bash

read -p "Certificate name: " cert_name
read -p "IP address: " ip

openssl genrsa -out $cert_name.key 2048

openssl req -new -key $cert_name.key -out $cert_name.csr

echo -e "[v3_req]\nsubjectAltName=IP:$ip" > $cert_name.ext

openssl x509 -req -days 3650 -in $cert_name.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out $cert_name.crt -extensions v3_req -extfile $cert_name.ext

rm $cert_name.csr $cert_name.ext
