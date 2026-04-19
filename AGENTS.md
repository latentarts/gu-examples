# Agent Instructions for `gu-examples`

This document provides rules and guidelines for AI agents working on the `gu-examples` repository. The goal is to ensure consistency, high-quality documentation, and proper integration of new examples into the showcase.

## Core Framework Principles (gu)

`gu` is a reactive Go framework for building web applications using WebAssembly. It uses fine-grained reactivity (signals) and a declarative UI API. For exhaustive details on how the framework works, refer to the [gu AGENTS.md](https://github.com/latentart/gu/blob/main/AGENTS.md).

### Reactive Primitives (`reactive` package)
-   **`NewSignal(initialValue)`**: Creates a getter and a setter for a reactive value.
    ```go
    count, setCount := reactive.NewSignal(0)
    ```
-   **`CreateMemo(func)`**: Creates a read-only signal that automatically recomputes when its dependencies change.
    ```go
    doubled := reactive.CreateMemo(func() int { return count() * 2 })
    ```
-   **`CreateEffect(func)`**: Runs a function and tracks its dependencies, re-running it whenever they change. Useful for side effects.

### Declarative UI (`el` package)
-   **Elements**: Standard HTML tags are available as functions: `el.Div()`, `el.Button()`, `el.Span()`, `el.P()`, `el.H1()`, etc.
-   **Text**: Use `el.Text("static")` for static text and `el.DynText(func() string)` for reactive text.
-   **Attributes & Styling**:
    -   `el.Class("static-class")` / `el.DynClass(func() string)`
    -   `el.Style("property", "value")` / `el.DynStyle("property", func() string)`
    -   `el.Attr("name", "value")` for general attributes.
-   **Events**: Use `el.OnClick(handler)`, `el.OnInput(handler)`, or the general `el.On("event", handler)`. Handlers take a `dom.Event`.
-   **Control Flow**:
    -   `el.Show(conditionFunc, node)`: Conditionally renders a node.
    -   `el.Dynamic(func() el.Node)`: Re-renders a subtree whenever any signal accessed inside the function changes.
-   **Lifecycle & Refs**:
    -   `el.OnMount(func(elem dom.Element))`: Called when the element is added to the DOM.
    -   `el.Ref(&domElementVar)`: Captures a reference to the underlying DOM element.

### Interop and Utilities (`jsutil` & `dom`)
-   **`jsutil`**: Provides logging (`LogInfo`, `LogDebug`, etc.), timing (`SetTimeout`, `SetInterval`), and JS promise helpers (`Await`).
-   **`dom`**: Low-level access to the DOM and event objects.

## Rules for Adding New Examples

1.  **Pure Go Focus:** Implement logic and UI in Go. Avoid JavaScript unless strictly required for interop or external libraries.
2.  **Isolated Directory:** Each example must reside in its own subdirectory with a `go.mod`, `index.html`, `main.go`, and `Makefile`.
3.  **Comprehensive README:** Every example MUST include a `README.md` containing:
    -   **Purpose:** What the example does.
    -   **How it Works:** Technical explanation of the implementation.
    -   **gu Implementation Details:** Code snippets showing how `gu` (signals, memos, elements) was used.
    -   **Developer Guidance:** Tips for developers learning from this example.

## Rules for Assets and Main Showcase

1.  **Video Assets:** If a video file is provided in `assets/` for the application:
    -   **Generate GIF & Thumbnail:** Create high-quality visual representations.
    -   **Update Main README:** Add the example to the "Example Showcase" section in the root `README.md` with the GIF/thumbnail and a description of features being tested.

## Maintenance and Updates

1.  **UI Changes:** If an existing example's UI or functionality changes significantly:
    -   **Request New Video:** Ask the user for a new screencapture video.
    -   **Update Documentation:** Revise both the example's `README.md` and the root `README.md` to reflect the new state.
