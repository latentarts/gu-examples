APP_NAME   := nodegraph-tauri
DIST       := dist
CACHE_DIR  := .cache
GOCACHE    := $(abspath $(CACHE_DIR)/go-build)
FRONTEND   := ./frontend
TAURI_DIR  := ./src-tauri
GOROOT     ?= $(shell go env GOROOT)
WASM_EXEC  := $(firstword $(wildcard $(GOROOT)/lib/wasm/wasm_exec.js $(GOROOT)/misc/wasm/wasm_exec.js))

.PHONY: wasm tauri-dev tauri-build clean test

wasm:
	@mkdir -p $(DIST) $(GOCACHE)
	GOCACHE=$(GOCACHE) go mod tidy
	GOCACHE=$(GOCACHE) GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o $(DIST)/main.wasm $(FRONTEND)
	cp $(FRONTEND)/index.html $(DIST)/index.html
	cp --remove-destination $(WASM_EXEC) $(DIST)/wasm_exec.js

tauri-dev: wasm
	npm exec tauri dev -- --config $(TAURI_DIR)/tauri.conf.json

tauri-build: wasm
	npm exec tauri build -- --config $(TAURI_DIR)/tauri.conf.json

test:
	@echo "Running WASM frontend tests..."
	PATH=$(PATH):$(GOROOT)/lib/wasm:$(GOROOT)/misc/wasm GOCACHE=$(GOCACHE) GOOS=js GOARCH=wasm go test ./frontend/...

clean:
	rm -rf $(DIST) $(CACHE_DIR) $(TAURI_DIR)/target
