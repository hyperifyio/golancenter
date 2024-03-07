
# Path to the Go WebAssembly JS support file
WASM_EXEC_JS := $(shell go env GOROOT)/misc/wasm/wasm_exec.js

# Default target executed when no arguments are given to make.
default: golancenter wasm

# Target for building the server
golancenter: ./cmd/golancenter/main.go wasm
	go build -o golancenter ./cmd/golancenter

# Target for building the WebAssembly module (replace with your actual build command)
wasm: ./misc/wasm/main.go
	mkdir -p ./cmd/golancenter/web/
	cp $(WASM_EXEC_JS) ./cmd/golancenter/web/
	GOOS=js GOARCH=wasm go build -o ./cmd/golancenter/web/myapp.wasm ./misc/wasm/main.go ./misc/wasm/conn.go

# Target for running the golancenter
run: golancenter
	./golancenter

# Target for cleaning up build artifacts
clean:
	rm -f ./golancenter ./cmd/golancenter/web/myapp.wasm ./cmd/golancenter/web/wasm_exec.js

.PHONY: default wasm run clean
