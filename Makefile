APPS = counter duckdb logging nodegraph openai-chat reporting shadcn stocks tailwind webgpu webllm
DIST = dist

.PHONY: build-all build-launcher copy-apps copy-assets serve clean

build-all:
	@echo "Building all examples..."
	@for app in $(APPS); do \
		echo "  Building $$app..."; \
		cd $$app && make build && cd ..; \
	done

build-launcher:
	@echo "Building launcher..."
	cd launcher && make build

copy-apps: build-all
	@mkdir -p $(DIST)/apps
	@for app in $(APPS); do \
		echo "  Copying $$app..."; \
		mkdir -p $(DIST)/apps/$$app; \
		cp $$app/dist/* $(DIST)/apps/$$app/; \
	done

copy-assets:
	@mkdir -p $(DIST)/assets
	cp assets/*.png $(DIST)/assets/
	cp assets/*.gif $(DIST)/assets/

serve: build-launcher copy-apps copy-assets
	cp launcher/dist/index.html $(DIST)/index.html
	cp launcher/dist/main.wasm $(DIST)/main.wasm
	cp launcher/dist/wasm_exec.js $(DIST)/wasm_exec.js
	@echo ""
	@echo "======================================="
	@echo "  gu Examples Launcher"
	@echo "  http://localhost:8080"
	@echo "======================================="
	@echo ""
	@cd $(DIST) && python3 -m http.server 8080

test:
	@echo "Testing all examples..."
	@for app in $(APPS); do \
		echo "  Testing $$app..."; \
		cd $$app && make test && cd ..; \
	done
	@echo "  Testing launcher..."
	@cd launcher && make test && cd ..

clean:
	@for app in $(APPS); do \
		cd $$app && make clean && cd ..; \
	done
	cd launcher && make clean && cd ..
	rm -rf $(DIST)