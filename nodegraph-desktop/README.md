# Node Graph Desktop

Desktop wrapper for the `nodegraph` example using `webview_go`. The UI remains the same Go WASM node graph editor, but it is embedded inside a native desktop window that serves `index.html`, `wasm_exec.js`, and `main.wasm` from the executable.

## Purpose

This sample demonstrates a minimal native shell around a `gu` Wasm app. It is useful for validating whether a browser-hosted example can be promoted into a desktop experience without rewriting the UI.

## How It Works

- `frontend/` contains the node graph Wasm frontend.
- `cmd/desktop/` starts a loopback HTTP server, serves the embedded frontend assets, and opens a `webview_go` window.
- `cmd/desktop/assets/` is populated during `make wasm` and embedded into the final binary.

## gu Implementation Details

The frontend is the same reactive node graph editor used by the browser sample. It still relies on signals, direct DOM updates during drag, and SVG rendering for edges. The native layer only hosts the page and does not change the `gu` architecture.

## Developer Guidance

- `make build` builds both the Wasm frontend and the native host.
- `make run` launches the desktop app.
- `make test` runs native host tests and the frontend Wasm tests.
- Linux builds depend on `webkit2gtk`/GTK development packages through `pkg-config`.

## Run It

```sh
make run
```
