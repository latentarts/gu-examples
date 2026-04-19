# Node Graph Editor

A fully interactive node graph editor with infinite canvas, built entirely in Go using the gu framework. Drag nodes from a sidebar, connect ports with bezier curves, pan and zoom the canvas, and enjoy a tactile 3D lift effect when dragging nodes. Tailwind CSS handles all styling with zero build step.

## Run it

```
make serve
```

Open http://localhost:8085.

## Features

- **Infinite canvas** with pan (drag background) and zoom (scroll wheel, cursor-centered)
- **Node drawer** — drag any of 6 node types onto the canvas
- **Port connections** — drag from an output port to an input port to create bezier connections
- **3D lift effect** — nodes scale up and cast a deeper shadow while dragging, then settle back down
- **Delete nodes** — click the X on any node header
- **Center graph** — toolbar button fits all nodes into view
- **Grid background** — dot grid that scales and tracks with pan/zoom

## Architecture

```
┌──────────┐  ┌──────────────────────────────────────────┐
│  Drawer   │  │  Canvas (overflow: hidden)               │
│           │  │  ┌────────────────────────────────────┐  │
│ [Input  ] │  │  │ World div                          │  │
│ [Output ] │──▶  │   transform: translate(px,py)      │  │
│ [Math   ] │  │  │              scale(zoom)            │  │
│ [Filter ] │  │  │                                    │  │
│ [Xform  ] │  │  │   ┌──SVG──┐  ┌─Nodes─┐           │  │
│ [Merge  ] │  │  │   │beziers│  │ abs pos│           │  │
│           │  │  │   └───────┘  └────────┘           │  │
└──────────┘  │  └────────────────────────────────────┘  │
              │  Toolbar: Center | Zoom | Node count     │
              └──────────────────────────────────────────┘
```

## gu concepts

### Version-counter pattern for non-comparable state

Nodes and connections are stored as `[]nodeData` and `[]connData` slices. Since slices aren't comparable, they can't be used directly with `reactive.NewSignal`. Instead, an integer version counter triggers re-renders:

```go
nodes := []nodeData{...}
nodeVer, setNodeVer := reactive.NewSignal(0)
bumpNodes := func() { setNodeVer(nodeVer() + 1) }

// el.Dynamic reads nodeVer(), so it re-runs when bumped
el.Dynamic(func() el.Node {
    _ = nodeVer()
    for _, nd := range nodes {
        // render each node...
    }
})
```

### Imperative DOM updates during drag

For smooth 60fps node dragging, the code bypasses reactive re-rendering entirely. During `pointermove`, it sets `style.left` and `style.top` directly on the DOM element via `js.Value`, and calls `updateSVG()` imperatively to redraw connections. Only on `pointerup` does it bump `nodeVer` to sync the reactive tree:

```go
moveFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
    nodes[idx].X = newX
    nodes[idx].Y = newY
    nodeEl.Get("style").Set("left", fmt.Sprintf("%.1fpx", newX))
    nodeEl.Get("style").Set("top", fmt.Sprintf("%.1fpx", newY))
    updateSVG() // imperative SVG redraw
    return nil
})
```

### 3D lift effect with CSS transitions

When a node drag begins, the DOM element gets `transform: scale(1.04)`, a deeper `box-shadow`, and an elevated `z-index`. A CSS `transition` on the element makes this animate smoothly. On release, properties revert and the node "settles" back:

```go
// Lift on pointerdown
nodeEl.Get("style").Set("transform", "scale(1.04)")
nodeEl.Get("style").Set("boxShadow", "0 20px 40px rgba(0,0,0,0.5)")
nodeEl.Get("style").Set("zIndex", "1000")

// Settle on pointerup
nodeEl.Get("style").Set("transform", "scale(1)")
nodeEl.Get("style").Set("boxShadow", "0 4px 12px rgba(0,0,0,0.3)")
```

### SVG connections via innerHTML

Since gu's `dom.CreateElement` doesn't support SVG-namespaced elements, connections are rendered by building an SVG string and setting it via `SetInnerHTML` on a container div. A `reactive.CreateEffect` tracks `connVer` and temp-connection signals to keep the SVG in sync:

```go
reactive.CreateEffect(func() {
    _ = connVer()
    _ = tempActive()
    content := buildSVGContent(nodes, conns, ...)
    svgContainerEl.SetInnerHTML(`<svg xmlns="...">` + content + `</svg>`)
})
```

Bezier curves use a control-point offset based on horizontal distance for natural-looking connections:

```go
dx := math.Abs(x2-x1) * 0.5
fmt.Sprintf(`M%.1f %.1fC%.1f %.1f %.1f %.1f %.1f %.1f`,
    x1, y1, x1+dx, y1, x2-dx, y2, x2, y2)
```

### Cursor-centered zoom

Wheel zoom adjusts pan offset so the point under the cursor stays fixed, using the standard zoom-to-point formula:

```go
newPanX = mx - (mx - panX) * (newZoom / oldZoom)
newPanY = my - (my - panY) * (newZoom / oldZoom)
```

### Port connection via elementFromPoint

When a port drag ends, `document.elementFromPoint` finds what's under the cursor. Data attributes on port circles (`data-port-node`, `data-port-idx`, `data-port-output`) identify the target. The handler validates that connections go from output to input on different nodes:

```go
target := js.Global().Get("document").Call("elementFromPoint", mx, my)
tNodeID := target.Call("getAttribute", "data-port-node").String()
tPortIdx := target.Call("getAttribute", "data-port-idx").Int()
tIsOutput := target.Call("getAttribute", "data-port-output").String() == "true"
```

### Drawer-to-canvas drag

Dragging from the drawer uses window-level pointer listeners. On `pointerup`, it checks if the cursor is within the canvas bounding rect, converts screen coordinates to world coordinates using pan and zoom, and appends a new node:

```go
wx := (dropX - canvasLeft - panX) / zoom
wy := (dropY - canvasTop - panY) / zoom
nodes = append(nodes, nodeData{ID: genID(), Type: nt.Name, X: wx, Y: wy})
bumpNodes()
```

### reactive.Batch for atomic camera updates

When centering the graph or zooming, `panX`, `panY`, and `zoom` all change together. `reactive.Batch` ensures the world container's `transform` style only recalculates once:

```go
reactive.Batch(func() {
    setZoom(z)
    setPanX(cw/2 - cx*z)
    setPanY(ch/2 - cy*z)
})
```

## Node types

| Node | Color | Inputs | Outputs |
|------|-------|--------|---------|
| Input | Green | — | out |
| Output | Red | in | — |
| Math | Blue | a, b | result |
| Filter | Purple | data | pass, fail |
| Transform | Amber | in | out |
| Merge | Cyan | a, b, c | merged |

## Files

| File | Purpose |
|------|---------|
| `main.go` | All Go/gu code — node graph editor, ~580 lines |
| `index.html` | Tailwind CDN + WASM bootstrap |
| `Makefile` | Build WASM + copy assets to `dist/`, serve locally |
