# gu Examples

This repository contains a collection of example applications built using the **gu** framework. `gu` is a reactive Go framework designed for building modern web applications with WebAssembly (WASM), providing a declarative API for DOM manipulation and efficient reactive state management.

## 🚀 Getting Started

Each example is a standalone Go WASM application. To run an example:

1. Navigate to the example directory: `cd <example-name>`
2. Build and serve: `make serve`
3. Open [http://localhost:8086](http://localhost:8086) in your browser.

Alternatively, just build the project with `make build` and use your preferred static file server in the `dist/` directory.

---

## 📚 Example Showcase

The following examples demonstrate various technical aspects and integration capabilities of the `gu` framework:

### 1. Counter (`/counter`)
*   **Purpose:** A classic reactive counter.
*   **Technical Highlights:**
    *   **Reactive Primitives:** Demonstrates `Signals` for state and `Memos` for derived calculations.
    *   **Theming:** Showcases the built-in `theme` system for dynamic light/dark mode switching.
    *   **Conditional Rendering:** Uses `el.Show` for declarative visibility toggling.

### 2. DuckDB Explorer (`/duckdb`)
*   **Purpose:** An in-browser SQL explorer powered by DuckDB WASM.
*   **Technical Highlights:**
    *   **External WASM Interop:** Demonstrates calling into other WASM modules from Go.
    *   **Async Operations:** Uses `jsutil.Await` to handle JavaScript Promises as Go channels/errors.
    *   **Dynamic Data Tables:** Efficiently rendering query results using reactive loops.

### 3. Logging & Observability (`/logging`)
*   **Purpose:** Demonstrates the framework's logging and debugging features.
*   **Technical Highlights:**
    *   **Structured Logging:** Showcases `jsutil.LogInfo`, `LogWarn`, and `LogError` with field support.
    *   **Debug Console:** Integration with the `gu` debug console for real-time WASM observability.
    *   **Stack Traces:** Demonstrates capturing and displaying meaningful Go stack traces in the browser.

### 4. Node Graph (`/nodegraph`)
*   **Purpose:** A visual, interactive node-based graph editor.
*   **Technical Highlights:**
    *   **Complex Interactions:** Drag-and-drop, zooming, and panning implemented with `gu` events.
    *   **SVG Integration:** Dynamic rendering of connection lines using SVG elements.
    *   **Coordinate Mapping:** Converting between screen space and world space in a reactive environment.

### 5. OpenAI Chat (`/openai-chat`)
*   **Purpose:** A streaming chat interface for OpenAI.
*   **Technical Highlights:**
    *   **Streaming API:** Handling Server-Sent Events (SSE) and partial updates reactively.
    *   **History Management:** Managing complex nested state for chat messages.

### 6. Reporting Dashboard (`/reporting`)
*   **Purpose:** A data-heavy dashboard with tables and file uploads.
*   **Technical Highlights:**
    *   **Table Patterns:** Demonstrates efficient rendering of large datasets in tables.
    *   **File API:** Handling browser file uploads (`input[type=file]`) via Go/WASM.

### 7. shadcn/ui Inspired Components (`/shadcn`)
*   **Purpose:** Implementation of modern UI components (Buttons, Cards, Inputs).
*   **Technical Highlights:**
    *   **Component Composition:** Building reusable UI primitives using `gu`'s functional approach.
    *   **Styling Patterns:** Combining utility-first CSS with declarative Go component props.

### 8. Stocks Dashboard (`/stocks`)
*   **Purpose:** Real-time (simulated) financial data visualization.
*   **Technical Highlights:**
    *   **Charting:** Integrating with canvas or SVG for high-frequency updates.
    *   **Periodic Updates:** Using timers to update reactive signals and trigger efficient DOM diffing.

### 9. Tailwind CSS Integration (`/tailwind`)
*   **Purpose:** Showcases seamless use of Tailwind CSS with `gu`.
*   **Technical Highlights:**
    *   **Class Management:** Using `el.Class` and `el.DynClass` to apply utility styles reactively.

### 10. WebGPU Triangle (`/webgpu`)
*   **Purpose:** Low-level 3D graphics rendering.
*   **Technical Highlights:**
    *   **Modern Browser APIs:** Direct interop with the WebGPU API for high-performance graphics.
    *   **Frame Loops:** Managing animation frames (`requestAnimationFrame`) within the Go lifecycle.

### 11. WebLLM Chat (`/webllm`)
*   **Purpose:** Fully local LLM chat running in the browser.
*   **Technical Highlights:**
    *   **Heavy Workloads:** Managing large WASM memory and GPU resources.
    *   **Markdown Rendering:** Integrating third-party JS libraries for rich text output.

---

## 🛠 Framework Core Concepts

These examples leverage several core `gu` packages:
-   `github.com/latentart/gu/el`: The declarative element API.
-   `github.com/latentart/gu/reactive`: Signals, Memos, and Effects.
-   `github.com/latentart/gu/dom`: Low-level DOM and Event access.
-   `github.com/latentart/gu/jsutil`: Helpers for JS interop, logging, and storage.
