# WebLLM Chat

Chat interface with a language model (SmolLM2-135M) running entirely in the browser via WebLLM. No server, no API keys — the model downloads and runs in WebGPU/WASM. Shows loading progress, chat history with user/assistant bubbles, and a typing indicator during generation.

## Run it

```
make serve
```

Open http://localhost:8082. The model (~70MB) downloads on first load (cached by the browser afterward). Once loaded, type a message and press Enter or click Send.

**Requirements:** A browser with WebGPU support (Chrome 113+, Edge 113+). The first load takes 30-60 seconds for model download.

## How it works

### Architecture

```
index.html                   main.go
  |                            |
  | imports @mlc-ai/web-llm    | builds chat UI with el.*
  | exposes window.App         | calls App.initEngine() on start
  |   .initEngine() -> Promise |   via goroutine + jsutil.Await
  |   .chat(json) -> Promise   | calls App.chat(json) per message
  |   .onProgress callback     |   via goroutine + jsutil.Await
  |                            | manages messages, input, loading
  |                            |   state with signals
```

JavaScript handles the WebLLM engine (model loading, inference). Go handles all UI state and rendering.

### gu concepts demonstrated

**Multiple signals for complex state** (`main.go:20-29`). The chat UI has several independent pieces of state, each a separate signal:

```go
input, setInput := reactive.NewSignal("")
loading, setLoading := reactive.NewSignal(true)
progress, setProgress := reactive.NewSignal("Initializing WebLLM...")
engineReady, setEngineReady := reactive.NewSignal(false)
generating, setGenerating := reactive.NewSignal(false)
errMsg, setErrMsg := reactive.NewSignal("")
```

Each signal is independent — changing `generating` doesn't re-render elements that only read `input`.

**Version-counter pattern for message history** (`main.go:28-29`). Same pattern as the DuckDB example. The `[]message` slice lives outside the reactive system; an integer signal notifies when it changes:

```go
msgVer, setMsgVer := reactive.NewSignal(0)
var messages []message

// After adding a message:
messages = append(messages, message{Role: "user", Content: text})
setMsgVer(msgVer() + 1)
```

**Setting a JS callback from Go** (`main.go:32-37`). Go sets a progress callback on `window.App` that JavaScript calls during model download. `js.FuncOf` wraps a Go function as a JS callback:

```go
progressCb := js.FuncOf(func(this js.Value, args []js.Value) any {
    setProgress(args[0].String())
    return nil
})
js.Global().Get("App").Set("onProgress", progressCb)
```

This bridges JS events into gu's reactive system — every time JS calls `onProgress`, the `progress` signal updates and the progress text re-renders.

**Goroutines for async operations** (`main.go:32-50, 65-82`). Both engine initialization and chat completion are async JS operations. Each runs in a goroutine to avoid blocking:

```go
go func() {
    promise := js.Global().Get("App").Call("initEngine")
    _, err := jsutil.Await(promise)
    // update signals...
}()
```

**Dynamic list rendering with `el.Dynamic`** (`main.go:153-160`). The message list rebuilds when `msgVer()` changes. A `[]any` slice is built with a loop and spread into `el.Div`:

```go
el.Dynamic(func() el.Node {
    _ = msgVer()
    args := []any{el.Class("space-y-4")}
    for _, msg := range messages {
        args = append(args, messageBubble(msg.Role, msg.Content))
    }
    return el.Div(args...)
})
```

**Conditional UI sections with `el.Show`** (`main.go:116-132, 144-224`). The loading screen, error display, and chat area are each wrapped in `el.Show` with a condition function. Only the relevant section renders:

```go
el.Show(
    func() bool { return loading() },
    // loading progress UI...
)
el.Show(
    func() bool { return !loading() && errMsg() == "" },
    // chat UI...
)
```

**Auto-scroll with `el.OnMount` + `reactive.CreateEffect`** (`main.go:180-186`). After the message container mounts, an effect watches `msgVer()` and `generating()` and scrolls to the bottom whenever either changes:

```go
el.OnMount(func(element dom.Element) {
    reactive.CreateEffect(func() {
        _ = msgVer()
        _ = generating()
        element.SetProperty("scrollTop", element.GetProperty("scrollHeight"))
    })
})
```

This shows how `el.OnMount` gives you the raw `dom.Element` to do imperative DOM operations, and `reactive.CreateEffect` re-runs automatically when its tracked signals change.

**Reactive disabled attribute** (`main.go:211-216`). The send button disables itself based on three signals:

```go
el.DynAttr("disabled", func() string {
    if generating() || !engineReady() || input() == "" {
        return "true"
    }
    return ""
})
```

Returning `""` from `DynAttr` removes the attribute entirely.

**Component extraction** (`main.go:228-247`). The `messageBubble` function takes role and content and returns a styled bubble. User messages are right-aligned blue, assistant messages are left-aligned gray — all via Tailwind classes.

### Controlled input with `el.DynProp` (`main.go:198`)

The input field uses `DynProp` (not `DynAttr`) to bind the `value` *property* to the `input` signal. This is important because the HTML `value` attribute only sets the initial value, while the JS `.value` property reflects the current value:

```go
el.DynProp("value", func() any { return input() })
```

When `sendMessage()` calls `setInput("")`, the input field clears because `DynProp` updates the property reactively.

## Files

| File | Purpose |
|---|---|
| `main.go` | All Go/gu code — signals, chat logic, UI tree |
| `index.html` | WebLLM import, `App.initEngine()`/`App.chat()`, Tailwind CDN |
| `Makefile` | Build WASM + copy assets to `dist/`, serve locally |
