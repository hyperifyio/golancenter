.PHONY: all build certs clean verify

all: build

build: certs

certs: root-ca-cert.pem client-ca-cert.pem server-ca-cert.pem client-cert.pem server-cert.pem client-ca-chain.pem server-ca-chain.pem

# Root CA
root-ca-key.pem:
	openssl genrsa -out root-ca-key.pem 2046

root-ca-cert.pem: root-ca-key.pem root-ca-cert.conf
	openssl req -x509 -new -nodes -key root-ca-key.pem -days 3650 -out root-ca-cert.pem -subj "/C=FI/ST=Tampere/L=Tampere/O=HyperifyIO/OU=Developers/CN=HyperifyCA" -sha256 -extensions v3_ca -config root-ca-cert.conf

# Intermediate Client CA
client-ca-key.pem:
	openssl genrsa -out client-ca-key.pem 2046

client-ca-csr.pem: client-ca-key.pem
	openssl req -new -key client-ca-key.pem -out client-ca-csr.pem -subj "/C=FI/ST=Tampere/L=Tampere/O=HyperifyIO/OU=IntermediateCA/CN=HyperifyIntermediateCA" -config client-ca-cert.conf

client-ca-cert.pem: client-ca-csr.pem root-ca-key.pem root-ca-cert.pem
	openssl x509 -req -in client-ca-csr.pem -CA root-ca-cert.pem -CAkey root-ca-key.pem -CAcreateserial -out client-ca-cert.pem -days 3650 -sha256 -extfile client-ca-cert.conf -extensions v3_intermediate_ca

# Intermediate Server CA
server-ca-key.pem:
	openssl genrsa -out server-ca-key.pem 2046

server-ca-csr.pem: server-ca-key.pem
	openssl req -new -key server-ca-key.pem -out server-ca-csr.pem -subj "/C=FI/ST=Tampere/L=Tampere/O=HyperifyIO/OU=IntermediateCA/CN=HyperifyIntermediateCA"

server-ca-cert.pem: server-ca-csr.pem root-ca-key.pem root-ca-cert.pem
	openssl x509 -req -in server-ca-csr.pem -CA root-ca-cert.pem -CAkey root-ca-key.pem -CAcreateserial -out server-ca-cert.pem -days 3650 -sha256 -extfile server-ca-cert.conf -extensions v3_intermediate_ca

# Client Certificate
client-key.pem:
	openssl genrsa -out client-key.pem 2046

client-csr.pem: client-key.pem
	openssl req -new -key client-key.pem -out client-csr.pem -config client-cert.conf -extensions req_ext

client-cert.pem: root-ca-cert.pem root-ca-key.pem client-csr.pem client-ca-cert.pem client-ca-key.pem
	openssl x509 -req -in client-csr.pem -CA client-ca-cert.pem -CAkey client-ca-key.pem -CAcreateserial -out client-cert.pem -days 3650 -sha256

# Server Certificate
server-key.pem:
	openssl genpkey -algorithm RSA -out server-key.pem -pkeyopt rsa_keygen_bits:2048

server-csr.pem: server-cert.conf server-key.pem
	openssl req -new -key server-key.pem -out server-csr.pem -config server-cert.conf -extensions req_ext

server-cert.pem: server-key.pem server-cert.conf server-csr.pem server-ca-cert.pem server-ca-key.pem
	openssl x509 -req -in server-csr.pem -CA server-ca-cert.pem -CAkey server-ca-key.pem -CAcreateserial -out server-cert.pem -days 3650 -sha256 -extfile server-cert.conf -extensions req_ext

client-ca-chain.pem:./client-ca-cert.pem ./root-ca-cert.pem
	cat ./client-ca-cert.pem ./root-ca-cert.pem > ./client-ca-chain.pem

server-ca-chain.pem: ./server-ca-cert.pem ./root-ca-cert.pem
	cat ./server-ca-cert.pem ./root-ca-cert.pem > ./server-ca-chain.pem

verify: client-ca-chain.pem server-ca-chain.pem
	openssl verify -CAfile ./client-ca-chain.pem ./client-cert.pem
	openssl verify -CAfile ./server-ca-chain.pem ./server-cert.pem

clean:
	rm -f server-ca-cert.pem server-cert.pem server-key.pem server-csr.pem \
	      client-cert.pem client-csr.pem client-key.pem client-server-cert.pem \
	      client-ca-cert.pem client-ca-csr.pem client-ca-key.pem client-ca-server-cert.pem \
	      server-ca-csr.pem server-ca-key.pem server-ca-server-cert.pem \
	      root-ca-cert.pem root-ca-key.pem server-ca-chain.pem client-ca-chain.pem \
	      ca-cert.srl client-ca-cert.srl root-ca-cert.srl server-ca-cert.srl