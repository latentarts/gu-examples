APP_NAME   := nodegraph-desktop
DIST       := dist
CACHE_DIR  := .cache
GOCACHE    := $(abspath $(CACHE_DIR)/go-build)
FRONTEND   := ./frontend
DESKTOP    := ./cmd/desktop
EMBED_DIR  := ./cmd/desktop/assets
GOROOT     ?= $(shell go env GOROOT)
WASM_EXEC  := $(firstword $(wildcard $(GOROOT)/lib/wasm/wasm_exec.js $(GOROOT)/misc/wasm/wasm_exec.js))

GOOS       ?= $(shell go env GOOS)
GOARCH     ?= $(shell go env GOARCH)

ifeq ($(GOOS),windows)
BIN_EXT := .exe
else
BIN_EXT :=
endif

.PHONY: build wasm desktop run clean test

build: wasm desktop

wasm:
	@mkdir -p $(DIST) $(EMBED_DIR) $(GOCACHE)
	GOCACHE=$(GOCACHE) go mod tidy
	GOCACHE=$(GOCACHE) GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o $(DIST)/main.wasm $(FRONTEND)
	cp $(FRONTEND)/index.html $(DIST)/index.html
	cp --remove-destination $(WASM_EXEC) $(DIST)/wasm_exec.js
	cp $(DIST)/index.html $(EMBED_DIR)/index.html
	cp $(DIST)/main.wasm $(EMBED_DIR)/main.wasm
	cp $(DIST)/wasm_exec.js $(EMBED_DIR)/wasm_exec.js

desktop:
	@mkdir -p $(DIST) $(GOCACHE)
	GOCACHE=$(GOCACHE) GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(DIST)/$(APP_NAME)$(BIN_EXT) $(DESKTOP)

run: build
	./$(DIST)/$(APP_NAME)$(BIN_EXT)

test:
	@echo "Running native desktop tests..."
	GOCACHE=$(GOCACHE) go test ./cmd/desktop
	@echo "Running WASM frontend tests..."
	PATH=$(PATH):$(GOROOT)/lib/wasm:$(GOROOT)/misc/wasm GOCACHE=$(GOCACHE) GOOS=js GOARCH=wasm go test ./frontend/...

clean:
	rm -rf $(DIST) $(CACHE_DIR)
	rm -f $(EMBED_DIR)/index.html $(EMBED_DIR)/main.wasm $(EMBED_DIR)/wasm_exec.js
