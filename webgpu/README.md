# WebGPU Rotating Cube

Animated 3D cube rendered with WebGPU and controlled from Go. The render loop runs via `requestAnimationFrame`, driven from a Go callback that reads reactive signals for rotation speed. UI controls let you adjust speed and change face colors in real time.

## Run it

```
make serve
```

Open http://localhost:8083. The cube starts rotating immediately. Use the speed slider and color preset buttons to interact.

**Requirements:** A browser with WebGPU support (Chrome 113+, Edge 113+).

## How it works

### Architecture

```
index.html                      main.go
  |                               |
  | defines window.App with:      | builds UI with el.*
  |   .init(canvas) -> Promise    | on mount: checks App.supported
  |   .render(angle)              |   then awaits App.init(canvas)
  |   .setColor(face,r,g,b)      | starts rAF loop calling
  |   .supported (bool)           |   App.render(angle)
  |                               | speed signal controls rotation
  | contains:                     | color buttons call App.setColor
  |   WGSL shaders               |
  |   matrix math helpers         |
  |   vertex/index buffer setup   |
```

All GPU setup (shaders, pipeline, buffers, matrix math) lives in JavaScript. Go controls the animation loop and UI.

### gu concepts demonstrated

**`el.Tag` for non-standard elements** (`main.go:77`). gu has helpers for common HTML elements (`el.Div`, `el.Button`, etc.) but `<canvas>` isn't one of them. `el.Tag` creates any element by tag name:

```go
el.Tag("canvas",
    el.Attr("width", "800"),
    el.Attr("height", "500"),
    el.Class("w-full bg-gray-900"),
    el.Ref(&canvasEl),
    el.OnMount(func(element dom.Element) { /* ... */ }),
)
```

**`el.Ref` for DOM element capture** (`main.go:81`). `el.Ref` stores the underlying `dom.Element` into a variable so you can pass it to JavaScript later:

```go
var canvasEl dom.Element
// ...
el.Ref(&canvasEl)
```

**`el.OnMount` for initialization** (`main.go:82-109`). The canvas needs to exist in the DOM before WebGPU can configure it. `el.OnMount` runs after the element is inserted, receiving the `dom.Element`:

```go
el.OnMount(func(element dom.Element) {
    app := js.Global().Get("App")
    if !app.Get("supported").Bool() {
        setSupported(false)
        return
    }
    go func() {
        promise := app.Call("init", element.Value)
        _, err := jsutil.Await(promise)
        // ...
    }()
})
```

Note `element.Value` — this passes the raw `js.Value` to JavaScript so it receives the actual canvas DOM element.

**`requestAnimationFrame` loop from Go** (`main.go:98-107`). After GPU init, the render loop uses `js.FuncOf` to create a self-scheduling callback. Each frame reads the `speed()` signal to compute the angle increment:

```go
angle := 0.0
var renderFrame js.Func
renderFrame = js.FuncOf(func(this js.Value, args []js.Value) any {
    angle += 0.016 * speed() * math.Pi
    app.Call("render", angle)
    js.Global().Call("requestAnimationFrame", renderFrame)
    return nil
})
js.Global().Call("requestAnimationFrame", renderFrame)
```

The `speed()` call inside the callback reads the signal's current value each frame. When the user moves the slider, the next frame automatically uses the new speed — no wiring needed.

**Range input with `el.OnInput`** (`main.go:131-143`). The speed slider is a standard HTML range input. `el.OnInput` fires on every drag, and `fmt.Sscanf` parses the string value to float64:

```go
el.Input(
    el.Type("range"),
    el.Attr("min", "0"), el.Attr("max", "5"), el.Attr("step", "0.1"),
    el.Value("1"),
    el.OnInput(func(e dom.Event) {
        var f float64
        fmt.Sscanf(e.TargetValue(), "%f", &f)
        setSpeed(f)
    }),
)
```

**Reactive text from signals** (`main.go:126-129`). The speed label updates as the slider moves:

```go
el.DynText(func() string {
    return fmt.Sprintf("%.1fx", speed())
})
```

**Building dynamic element lists** (`main.go:159-177`). The color buttons are built in a loop. Since `el.Div` takes `...any`, we build a `[]any` slice and spread it:

```go
func colorButtonGrid() el.Node {
    args := []any{el.Class("flex flex-wrap gap-2")}
    for _, p := range presets {
        p := p // capture loop variable
        args = append(args, el.Button(
            el.OnClick(func(e dom.Event) {
                js.Global().Get("App").Call("setColor", p.Face, p.R, p.G, p.B)
            }),
            // ...
        ))
    }
    return el.Div(args...)
}
```

The `p := p` line is important — without it, all closures would capture the same loop variable.

**Conditional sections with `el.Show`** (`main.go:53-56, 60-72, 114-154`). The "WebGPU Active" badge, error message, unsupported warning, and controls panel each appear/disappear based on signal state:

```go
el.Show(
    func() bool { return supported() && running() },
    el.Span(el.Class("..."), el.Text("WebGPU Active")),
)
```

**Inline styles with `el.Style`** (`main.go:168`). For one-off styles that aren't Tailwind utilities (like dynamic background colors), `el.Style` sets a CSS property directly:

```go
el.Span(
    el.Class("w-3 h-3 rounded-full"),
    el.Style("background-color", bg),
)
```

### JS-side details

The `index.html` contains ~120 lines of self-contained WebGPU setup:

- **Matrix math** — `mat4Perspective`, `mat4LookAt`, `mat4RotateY`, `mat4RotateX`, `mat4Multiply` (~40 lines)
- **Vertex data** — 6 faces, 4 vertices each, with position (xyz) + color (rgb) per vertex
- **WGSL shaders** — minimal vertex/fragment shaders that transform by a uniform MVP matrix
- **`App.render(angle)`** — called per frame, computes MVP matrix, writes to uniform buffer, submits a render pass
- **`App.setColor(face, r, g, b)`** — updates vertex buffer colors for a specific face

## Files

| File | Purpose |
|---|---|
| `main.go` | All Go/gu code — signals, canvas setup, render loop, controls |
| `index.html` | WebGPU pipeline, shaders, matrix math, Tailwind CDN |
| `Makefile` | Build WASM + copy assets to `dist/`, serve locally |
