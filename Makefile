
# Path to the Go WebAssembly JS support file
WASM_EXEC_JS := $(shell go env GOROOT)/misc/wasm/wasm_exec.js

# Default target executed when no arguments are given to make.
default: golancenter testserver wasm

# Target for building the server
golancenter: ./cmd/golancenter/main.go wasm
	go build -o golancenter ./cmd/golancenter

# Target for building the mtls test server
testserver: ./cmd/testserver/main.go
	go build -o testserver ./cmd/testserver

certs:
	make -C certs certs

# Target for building the WebAssembly module (replace with your actual build command)
wasm: ./misc/wasm/main.go
	mkdir -p ./misc/wasm/certs/
	cp ./certs/root-ca-cert.pem ./misc/wasm/certs/root-ca-cert.pem
	cp ./certs/client-ca-cert.pem ./misc/wasm/certs/client-ca-cert.pem
	cp ./certs/server-ca-cert.pem ./misc/wasm/certs/server-ca-cert.pem
	cp ./certs/client-cert.pem ./misc/wasm/certs/client-cert.pem
	cp ./certs/client-key.pem ./misc/wasm/certs/client-key.pem
	mkdir -p ./cmd/golancenter/web/
	cp ./certs/root-ca-cert.pem ./cmd/golancenter/web/root-ca-cert.pem
	cp $(WASM_EXEC_JS) ./cmd/golancenter/web/
	GOOS=js GOARCH=wasm go build -o ./cmd/golancenter/web/myapp.wasm ./misc/wasm/main.go ./misc/wasm/conn.go

# Target for running the golancenter
run: golancenter
	./golancenter

# Target for cleaning up build artifacts
clean:
	make -C certs clean
	rm -f ./golancenter ./testserver \
		  ./cmd/golancenter/web/myapp.wasm \
	      ./cmd/golancenter/web/wasm_exec.js \
	      ./misc/wasm/certs/*.pem \
	      ./cmd/golancenter/web/root-ca-cert.pem
	test -d ./misc/wasm/certs && rmdir ./misc/wasm/certs || true

.PHONY: default wasm run clean
