#!/bin/bash

#
# Generate test CA, Client cert, Server Cert
#

TEST_HOSTNAME=`hostname`
echo "Host name is [$TEST_HOSTNAME]"

set -x

echo "subjectAltName=DNS:$TEST_HOSTNAME" > altsubj.ext

# Generate root CA
openssl req -nodes -x509 -newkey rsa:2048 -keyout ca.key -out ca.crt -subj "/C=LV/ST=RIGA/L=Riga/O=Endurox_CA/OU=root/CN=$TEST_HOSTNAME/emailAddress=test@mavimax.com"

# Generate server cert
openssl req -nodes -newkey rsa:2048 -keyout server.key -out server.csr -subj "/C=LV/ST=RIGA/L=Riga/O=Endurox_SV/OU=root/CN=$TEST_HOSTNAME/emailAddress=test@mavimax.com"

# Sign server cert
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -extfile altsubj.ext

# Generate client cert
openssl req -nodes -newkey rsa:2048 -keyout client.key -out client.csr -subj "/C=LV/ST=RIGA/L=Riga/O=Endurox_CL/OU=root/CN=$TEST_HOSTNAME/emailAddress=test@mavimax.com"

# Sign client cert
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAserial ca.srl -out client.crt -extfile altsubj.ext

