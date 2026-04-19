GOROOT     ?= $(shell go env GOROOT)
WASM_EXEC  := $(firstword $(wildcard $(GOROOT)/lib/wasm/wasm_exec.js $(GOROOT)/misc/wasm/wasm_exec.js))
DIST       := dist

.PHONY: build tinygo serve tinygo-serve clean

build:
	@mkdir -p $(DIST)
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o $(DIST)/main.wasm .
	cp index.html $(DIST)/index.html
	cp --remove-destination $(WASM_EXEC) $(DIST)/wasm_exec.js
	@echo "Build complete -> $(DIST)/"
	@ls -lh $(DIST)/main.wasm | awk '{print "WASM size:", $$5}'

tinygo:
	@mkdir -p $(DIST)
	tinygo build -target=wasm -opt=z -no-debug -o $(DIST)/main.wasm .
	cp index.html $(DIST)/index.html
	cp --remove-destination "$$(tinygo env TINYGOROOT)/targets/wasm_exec.js" $(DIST)/wasm_exec.js
	@echo "TinyGo build complete -> $(DIST)/"
	@ls -lh $(DIST)/main.wasm | awk '{print "WASM size:", $$5}'

serve: build
	@echo "Serving on http://localhost:8082"
	@cd $(DIST) && python3 -m http.server 8082

tinygo-serve: tinygo
	@echo "Serving on http://localhost:8082"
	@cd $(DIST) && python3 -m http.server 8082

clean:
	rm -rf $(DIST)
