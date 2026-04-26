# Node Graph Tauri

Desktop wrapper for the `nodegraph` example using Tauri. The frontend remains the Go Wasm node graph app, while Tauri provides the native window and packaging layer.

## Purpose

This sample demonstrates a second desktop packaging route for `gu` apps. Compared with `nodegraph-desktop`, it swaps the Go webview host for a Rust/Tauri shell.

## How It Works

- `frontend/` contains the node graph Wasm frontend.
- `dist/` receives `index.html`, `main.wasm`, and `wasm_exec.js`.
- `src-tauri/` contains the Tauri configuration and Rust entrypoints that load `dist/` as the frontend bundle.

## gu Implementation Details

The UI is still the same reactive `gu` application. Tauri does not alter the node graph logic; it only changes how the Wasm bundle is launched and packaged for desktop distribution.

## Developer Guidance

- `make wasm` refreshes the frontend bundle consumed by Tauri.
- `make tauri-dev` launches the native shell in development mode.
- `make tauri-build` produces a packaged desktop build.
- `make test` runs the frontend Wasm tests.
- Tauri requires Rust plus the platform-specific webview toolchain. The JavaScript CLI also requires `npm install`.

## Run It

```sh
make tauri-dev
```
