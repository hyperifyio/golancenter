# Path to the Go WebAssembly JS support file from the Go installation
WASM_EXEC_JS := $(shell go env GOROOT)/misc/wasm/wasm_exec.js

# Default target executed when no arguments are given to make.
default: golancenter

# Target for building the server executable
golancenter:
	go build -o golancenter ./cmd/golancenter

# Uncomment below when you have a Go file for WASM compilation
# wasm:
#	mkdir -p ./cmd/golancenter/web/
#	cp $(WASM_EXEC_JS) ./cmd/golancenter/web/
#	GOOS=js GOARCH=wasm go build -o ./cmd/golancenter/web/app.wasm ./path/to/your/wasm/main.go

# Target for running the golancenter server
run: golancenter
	./golancenter

# Target for cleaning up build artifacts
clean:
	rm -f ./golancenter
	rm -f ./cmd/golancenter/web/app.wasm
	rm -f ./cmd/golancenter/web/wasm_exec.js

.PHONY: default golancenter run clean
