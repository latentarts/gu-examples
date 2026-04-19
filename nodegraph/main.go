//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/reactive"
)

// ── App ────────────────────────────────────────────────────────────────

func App() el.Node {
	// -- State: nodes & connections (version-counter pattern) --
	nodes := []nodeData{
		{ID: "n1", Type: "Input", X: 80, Y: 120},
		{ID: "n2", Type: "Transform", X: 380, Y: 100},
		{ID: "n3", Type: "Output", X: 680, Y: 140},
	}
	conns := []connData{
		{FromNode: "n1", FromPort: 0, ToNode: "n2", ToPort: 0},
		{FromNode: "n2", FromPort: 0, ToNode: "n3", ToPort: 0},
	}
	nodeVer, setNodeVer := reactive.NewSignal(0)
	connVer, setConnVer := reactive.NewSignal(0)

	bumpNodes := func() { setNodeVer(nodeVer() + 1) }
	bumpConns := func() { setConnVer(connVer() + 1) }

	// -- Camera --
	panX, setPanX := reactive.NewSignal(0.0)
	panY, setPanY := reactive.NewSignal(0.0)
	zoom, setZoom := reactive.NewSignal(1.0)

	// -- Drawer --
	drawerOpen, setDrawerOpen := reactive.NewSignal(true)

	// -- Temp connection line --
	tempActive, setTempActive := reactive.NewSignal(false)
	tempX1, setTempX1 := reactive.NewSignal(0.0)
	tempY1, setTempY1 := reactive.NewSignal(0.0)
	tempX2, setTempX2 := reactive.NewSignal(0.0)
	tempY2, setTempY2 := reactive.NewSignal(0.0)

	// -- Placing node from drawer --
	placingType, setPlacingType := reactive.NewSignal("")

	// -- Refs --
	var canvasEl dom.Element
	var worldEl dom.Element
	var svgContainerEl dom.Element

	// ── SVG update ─────────────────────────────────────────────
	updateSVG := func() {
		if svgContainerEl.IsNull() {
			return
		}
		content := buildSVGContent(nodes, conns, tempX1(), tempY1(), tempX2(), tempY2(), tempActive())
		svgContainerEl.SetInnerHTML(
			`<svg xmlns="http://www.w3.org/2000/svg" style="position:absolute;top:0;left:0;width:100%;height:100%;overflow:visible;pointer-events:none">` + content + `</svg>`)
	}

	// Reactive SVG: re-render when connVer or temp signals change
	reactive.CreateEffect(func() {
		_ = connVer()
		_ = tempActive()
		_ = tempX1()
		_ = tempY1()
		_ = tempX2()
		_ = tempY2()
		updateSVG()
	})

	// ── Center graph ───────────────────────────────────────────
	centerGraph := func() {
		if canvasEl.IsNull() || len(nodes) == 0 {
			return
		}
		cw := canvasEl.GetProperty("offsetWidth").Float()
		ch := canvasEl.GetProperty("offsetHeight").Float()
		minX, minY, maxX, maxY := centerGraphBounds(nodes)
		px, py, z := calcCenterView(minX, minY, maxX, maxY, cw, ch)
		reactive.Batch(func() {
			setZoom(z)
			setPanX(px)
			setPanY(py)
		})
	}

	// ── Delete node handler ───────────────────────────────────
	deleteNodeHandler := func(id string) {
		nodes, conns = deleteNode(nodes, conns, id)
		reactive.Batch(func() {
			bumpNodes()
			bumpConns()
		})
	}

	// ── Node drag (imperative) ─────────────────────────────────
	startNodeDrag := func(id string, e dom.Event) {
		e.StopPropagation()
		e.PreventDefault()
		idx := findNodeIdx(nodes, id)
		if idx < 0 {
			return
		}
		startMX := e.Value.Get("clientX").Float()
		startMY := e.Value.Get("clientY").Float()
		startNX := nodes[idx].X
		startNY := nodes[idx].Y
		z := zoom()

		// Find the DOM element for this node
		nodeEl := js.Global().Get("document").Call("querySelector", fmt.Sprintf(`[data-node-id="%s"]`, id))
		if nodeEl.IsNull() || nodeEl.IsUndefined() {
			return
		}

		// 3D lift effect
		nodeEl.Get("style").Set("transform", "scale(1.04)")
		nodeEl.Get("style").Set("boxShadow", "0 20px 40px rgba(0,0,0,0.5)")
		nodeEl.Get("style").Set("zIndex", "1000")
		nodeEl.Get("style").Set("transition", "transform 0.15s ease, box-shadow 0.15s ease")

		bodyStyle := js.Global().Get("document").Get("body").Get("style")
		bodyStyle.Set("cursor", "grabbing")
		bodyStyle.Set("userSelect", "none")

		var moveFn, upFn js.Func
		moveFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			dx := (args[0].Get("clientX").Float() - startMX) / z
			dy := (args[0].Get("clientY").Float() - startMY) / z
			newX := startNX + dx
			newY := startNY + dy
			nodes[idx].X = newX
			nodes[idx].Y = newY
			// Imperative DOM update
			nodeEl.Get("style").Set("left", fmt.Sprintf("%.1fpx", newX))
			nodeEl.Get("style").Set("top", fmt.Sprintf("%.1fpx", newY))
			// Update connections imperatively
			updateSVG()
			return nil
		})
		upFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			js.Global().Call("removeEventListener", "pointermove", moveFn)
			js.Global().Call("removeEventListener", "pointerup", upFn)
			moveFn.Release()
			upFn.Release()
			// Remove 3D lift
			nodeEl.Get("style").Set("transform", "scale(1)")
			nodeEl.Get("style").Set("boxShadow", "0 4px 12px rgba(0,0,0,0.3)")
			nodeEl.Get("style").Set("zIndex", "")
			bodyStyle.Set("cursor", "")
			bodyStyle.Set("userSelect", "")
			// Sync signals
			bumpNodes()
			return nil
		})
		js.Global().Call("addEventListener", "pointermove", moveFn)
		js.Global().Call("addEventListener", "pointerup", upFn)
	}

	// ── Port connection drag ───────────────────────────────────
	startPortDrag := func(nodeID string, isOutput bool, portIdx int, e dom.Event) {
		e.StopPropagation()
		e.PreventDefault()

		nIdx := findNodeIdx(nodes, nodeID)
		if nIdx < 0 {
			return
		}
		ox, oy := portWorldXY(nodes[nIdx], isOutput, portIdx)
		setTempX1(ox)
		setTempY1(oy)
		setTempX2(ox)
		setTempY2(oy)
		setTempActive(true)

		z := zoom()
		px := panX()
		py := panY()

		var moveFn, upFn js.Func
		moveFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			mx := args[0].Get("clientX").Float()
			my := args[0].Get("clientY").Float()
			rect := canvasEl.Value.Call("getBoundingClientRect")
			wx := (mx - rect.Get("left").Float() - px) / z
			wy := (my - rect.Get("top").Float() - py) / z
			setTempX2(wx)
			setTempY2(wy)
			return nil
		})
		upFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			js.Global().Call("removeEventListener", "pointermove", moveFn)
			js.Global().Call("removeEventListener", "pointerup", upFn)
			moveFn.Release()
			upFn.Release()

			setTempActive(false)

			// Check if dropped on a compatible port
			mx := args[0].Get("clientX").Float()
			my := args[0].Get("clientY").Float()
			target := js.Global().Get("document").Call("elementFromPoint", mx, my)

			if !target.IsNull() && !target.IsUndefined() {
				tNodeID := target.Call("getAttribute", "data-port-node").String()
				tPortIdxStr := target.Call("getAttribute", "data-port-idx").String()
				tIsOutput := target.Call("getAttribute", "data-port-output").String()

				if tNodeID != "<null>" && tNodeID != "" && tPortIdxStr != "<null>" && tPortIdxStr != "" {
					tIdx := 0
					fmt.Sscanf(tPortIdxStr, "%d", &tIdx)
					tOut := tIsOutput == "true"

					// Must connect output→input (different nodes)
					if isOutput && !tOut {
						var ok bool
						conns, ok = addConnection(conns, nodeID, portIdx, tNodeID, tIdx, true)
						if ok {
							bumpConns()
						}
					} else if !isOutput && tOut {
						var ok bool
						conns, ok = addConnection(conns, nodeID, portIdx, tNodeID, tIdx, false)
						if ok {
							bumpConns()
						}
					}
				}
			}
			return nil
		})
		js.Global().Call("addEventListener", "pointermove", moveFn)
		js.Global().Call("addEventListener", "pointerup", upFn)
	}

	// ── Canvas pan ─────────────────────────────────────────────
	startCanvasPan := func(e dom.Event) {
		// Only pan on left-click directly on canvas/world background
		if e.Value.Get("button").Int() != 0 {
			return
		}
		e.PreventDefault()
		startMX := e.Value.Get("clientX").Float()
		startMY := e.Value.Get("clientY").Float()
		startPX := panX()
		startPY := panY()

		bodyStyle := js.Global().Get("document").Get("body").Get("style")
		bodyStyle.Set("cursor", "grabbing")
		bodyStyle.Set("userSelect", "none")

		var moveFn, upFn js.Func
		moveFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			dx := args[0].Get("clientX").Float() - startMX
			dy := args[0].Get("clientY").Float() - startMY
			reactive.Batch(func() {
				setPanX(startPX + dx)
				setPanY(startPY + dy)
			})
			return nil
		})
		upFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			js.Global().Call("removeEventListener", "pointermove", moveFn)
			js.Global().Call("removeEventListener", "pointerup", upFn)
			moveFn.Release()
			upFn.Release()
			bodyStyle.Set("cursor", "")
			bodyStyle.Set("userSelect", "")
			return nil
		})
		js.Global().Call("addEventListener", "pointermove", moveFn)
		js.Global().Call("addEventListener", "pointerup", upFn)
	}

	// ── Render a single node ───────────────────────────────────
	renderNode := func(nd nodeData) el.Node {
		nt := catalogByName(nd.Type)
		h := ntHeight(nt)
		_ = h

		portCircle := func(nodeID string, isOutput bool, idx int, label string) el.Node {
			circleStyles := map[string]string{
				"position":      "absolute",
				"top":           "50%",
				"transform":     "translateY(-50%)",
				"width":         fmt.Sprintf("%dpx", portR*2),
				"height":        fmt.Sprintf("%dpx", portR*2),
				"border-radius": "50%",
				"background":    nt.Color,
				"border":        "2px solid #1e293b",
				"cursor":        "crosshair",
				"z-index":       "10",
			}
			labelStyles := map[string]string{
				"flex":        "1",
				"font-size":   "11px",
				"color":       "#cbd5e1",
				"user-select": "none",
			}
			if isOutput {
				circleStyles["right"] = "-6px"
				labelStyles["text-align"] = "right"
				labelStyles["padding-right"] = "18px"
			} else {
				circleStyles["left"] = "-6px"
				labelStyles["text-align"] = "left"
				labelStyles["padding-left"] = "18px"
			}
			return el.Div(
				el.Style("position", "relative"),
				el.Style("height", fmt.Sprintf("%dpx", portRowH)),
				el.Style("display", "flex"),
				el.Style("align-items", "center"),
				// Port circle
				el.Div(
					el.Attr("data-port-node", nodeID),
					el.Attr("data-port-idx", fmt.Sprintf("%d", idx)),
					el.Attr("data-port-output", fmt.Sprintf("%v", isOutput)),
					el.StyleMap(circleStyles),
					el.On("pointerdown", func(e dom.Event) {
						startPortDrag(nodeID, isOutput, idx, e)
					}),
				),
				// Label
				el.Span(
					el.StyleMap(labelStyles),
					el.Text(label),
				),
			)
		}

		// Build port rows
		maxPorts := len(nt.Inputs)
		if len(nt.Outputs) > maxPorts {
			maxPorts = len(nt.Outputs)
		}
		portArgs := []any{
			el.Style("padding", fmt.Sprintf("%dpx 0", portPadY)),
		}
		for i := 0; i < maxPorts; i++ {
			rowArgs := []any{
				el.Style("display", "flex"),
				el.Style("height", fmt.Sprintf("%dpx", portRowH)),
			}
			// Input port
			if i < len(nt.Inputs) {
				rowArgs = append(rowArgs, el.Div(
					el.Style("flex", "1"),
					portCircle(nd.ID, false, i, nt.Inputs[i]),
				))
			} else {
				rowArgs = append(rowArgs, el.Div(el.Style("flex", "1")))
			}
			// Output port
			if i < len(nt.Outputs) {
				rowArgs = append(rowArgs, el.Div(
					el.Style("flex", "1"),
					portCircle(nd.ID, true, i, nt.Outputs[i]),
				))
			} else {
				rowArgs = append(rowArgs, el.Div(el.Style("flex", "1")))
			}
			portArgs = append(portArgs, el.Div(rowArgs...))
		}

		return el.Div(
			el.Attr("data-node-id", nd.ID),
			el.Style("position", "absolute"),
			el.Style("left", fmt.Sprintf("%.1fpx", nd.X)),
			el.Style("top", fmt.Sprintf("%.1fpx", nd.Y)),
			el.Style("width", fmt.Sprintf("%dpx", nodeW)),
			el.Style("background", "#1e293b"),
			el.Style("border-radius", "10px"),
			el.Style("border", fmt.Sprintf("1.5px solid %s", nt.Color)),
			el.Style("box-shadow", "0 4px 12px rgba(0,0,0,0.3)"),
			el.Style("cursor", "grab"),
			el.Style("user-select", "none"),
			el.Style("transition", "transform 0.15s ease, box-shadow 0.15s ease"),
			el.Style("overflow", "visible"),
			el.On("pointerdown", func(e dom.Event) {
				// Ignore if clicking on a port
				target := e.Value.Get("target")
				if target.Call("hasAttribute", "data-port-node").Bool() {
					return
				}
				startNodeDrag(nd.ID, e)
			}),
			// Header
			el.Div(
				el.Style("height", fmt.Sprintf("%dpx", headerH)),
				el.Style("background", nt.Color),
				el.Style("border-radius", "8px 8px 0 0"),
				el.Style("display", "flex"),
				el.Style("align-items", "center"),
				el.Style("padding", "0 10px"),
				el.Style("justify-content", "space-between"),
				el.Span(
					el.Style("font-weight", "700"),
					el.Style("font-size", "13px"),
					el.Style("color", "#0f172a"),
					el.Text(fmt.Sprintf("%s  %s", nt.Icon, nt.Name)),
				),
				// Delete button
				el.Button(
					el.Style("background", "none"),
					el.Style("border", "none"),
					el.Style("color", "#0f172a88"),
					el.Style("cursor", "pointer"),
					el.Style("font-size", "14px"),
					el.Style("font-weight", "bold"),
					el.Style("padding", "0 2px"),
					el.Style("line-height", "1"),
					el.Text("\u2715"),
					el.On("pointerdown", func(e dom.Event) {
						e.StopPropagation()
					}),
					el.OnClick(func(e dom.Event) {
						e.StopPropagation()
						deleteNodeHandler(nd.ID)
					}),
				),
			),
			// Ports area
			el.Div(portArgs...),
		)
	}

	// ── Drawer item ────────────────────────────────────────────
	renderDrawerItem := func(nt nodeTypeDef) el.Node {
		return el.Div(
			el.Class("flex items-center gap-3 p-2.5 rounded-lg cursor-grab hover:bg-slate-700/50 transition-colors"),
			el.Style("user-select", "none"),
			el.Style("touch-action", "none"),
			el.On("pointerdown", func(e dom.Event) {
				e.PreventDefault()
				setPlacingType(nt.Name)

				mx := e.Value.Get("clientX").Float()
				my := e.Value.Get("clientY").Float()

				var moveFn, upFn js.Func
				moveFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
					_ = args[0].Get("clientX").Float()
					_ = args[0].Get("clientY").Float()
					return nil
				})
				upFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
					js.Global().Call("removeEventListener", "pointermove", moveFn)
					js.Global().Call("removeEventListener", "pointerup", upFn)
					moveFn.Release()
					upFn.Release()
					setPlacingType("")

					dropX := args[0].Get("clientX").Float()
					dropY := args[0].Get("clientY").Float()

					// Check if dropped on canvas
					if canvasEl.IsNull() {
						return nil
					}
					rect := canvasEl.Value.Call("getBoundingClientRect")
					cl := rect.Get("left").Float()
					ct := rect.Get("top").Float()
					cr := rect.Get("right").Float()
					cb := rect.Get("bottom").Float()

					if dropX >= cl && dropX <= cr && dropY >= ct && dropY <= cb {
						z := zoom()
						wx := (dropX - cl - panX()) / z
						wy := (dropY - ct - panY()) / z
						nodes = addNode(nodes, nt.Name, wx-nodeW/2, wy-30)
						bumpNodes()
					}
					return nil
				})
				_ = mx
				_ = my
				js.Global().Call("addEventListener", "pointermove", moveFn)
				js.Global().Call("addEventListener", "pointerup", upFn)
			}),
			// Icon circle
			el.Div(
				el.Class("w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold"),
				el.Style("background", nt.Color),
				el.Style("color", "#0f172a"),
				el.Text(nt.Icon),
			),
			// Label
			el.Div(
				el.Span(
					el.Class("text-sm font-semibold text-slate-200"),
					el.Text(nt.Name),
				),
				el.Div(
					el.Class("text-xs text-slate-400"),
					el.Text(fmt.Sprintf("%d in / %d out", len(nt.Inputs), len(nt.Outputs))),
				),
			),
		)
	}

	// ── Build the UI ───────────────────────────────────────────
	// Drawer items
	drawerItems := []any{
		el.Div(
			el.Class("p-4 border-b border-slate-700"),
			el.Div(
				el.Class("flex items-center justify-between"),
				el.H3(el.Class("text-sm font-bold text-slate-300 uppercase tracking-wider"), el.Text("Nodes")),
				el.Button(
					el.Class("text-slate-400 hover:text-slate-200 text-lg leading-none p-1"),
					el.Text("\u2715"),
					el.OnClick(func(e dom.Event) { setDrawerOpen(false) }),
				),
			),
			el.P(el.Class("text-xs text-slate-500 mt-1"), el.Text("Drag onto canvas")),
		),
	}
	for _, nt := range catalog {
		drawerItems = append(drawerItems, el.Div(
			el.Class("px-3 py-1"),
			renderDrawerItem(nt),
		))
	}

	drawerArgs := []any{
		el.Class("h-full bg-slate-800 border-r border-slate-700 flex flex-col overflow-y-auto"),
		el.DynStyle("width", func() string {
			if drawerOpen() {
				return "220px"
			}
			return "0px"
		}),
		el.Style("transition", "width 0.2s ease"),
		el.Style("overflow", "hidden"),
		el.Style("flex-shrink", "0"),
	}
	drawerArgs = append(drawerArgs, drawerItems...)

	return el.Div(
		el.Class("flex h-screen bg-slate-900 text-slate-200 font-sans"),
		// Drawer
		el.Div(drawerArgs...),
		// Main area
		el.Div(
			el.Class("flex-1 flex flex-col"),
			// Toolbar
			el.Div(
				el.Class("h-12 bg-slate-800 border-b border-slate-700 flex items-center px-4 gap-2"),
				el.Show(func() bool { return !drawerOpen() },
					el.Button(
						el.Class("px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded text-sm font-medium transition-colors"),
						el.Text("\u2630 Nodes"),
						el.OnClick(func(e dom.Event) { setDrawerOpen(true) }),
					),
				),
				el.Button(
					el.Class("px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded text-sm font-medium transition-colors"),
					el.Text("\u2316 Center"),
					el.OnClick(func(e dom.Event) { centerGraph() }),
				),
				el.Button(
					el.Class("px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded text-sm font-medium transition-colors"),
					el.Text("+ Zoom In"),
					el.OnClick(func(e dom.Event) {
						z := zoom() * 1.25
						if z > maxZoom {
							z = maxZoom
						}
						setZoom(z)
					}),
				),
				el.Button(
					el.Class("px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded text-sm font-medium transition-colors"),
					el.Text("\u2212 Zoom Out"),
					el.OnClick(func(e dom.Event) {
						z := zoom() * 0.8
						if z < minZoom {
							z = minZoom
						}
						setZoom(z)
					}),
				),
				el.Div(el.Class("flex-1")),
				el.DynText(func() string {
					return fmt.Sprintf("%.0f%%", zoom()*100)
				}),
				el.Span(el.Class("text-slate-500 text-xs ml-2"),
					el.DynText(func() string {
						_ = nodeVer()
						return fmt.Sprintf("%d nodes", len(nodes))
					}),
				),
				el.Span(el.Class("text-slate-500 text-xs ml-2"),
					el.DynText(func() string {
						_ = connVer()
						return fmt.Sprintf("%d connections", len(conns))
					}),
				),
			),
			// Canvas area
			el.Div(
				el.Ref(&canvasEl),
				el.Class("flex-1 relative overflow-hidden"),
				el.Style("background", "#0f172a"),
				el.Style("cursor", "grab"),
				// Grid background
				el.DynStyle("background-image", func() string {
					z := zoom()
					ds := 2 * z
					if ds < 0.5 {
						ds = 0.5
					}
					return fmt.Sprintf("radial-gradient(circle, #334155 %.1fpx, transparent %.1fpx)", ds, ds)
				}),
				el.DynStyle("background-size", func() string {
					gs := 30 * zoom()
					return fmt.Sprintf("%.1fpx %.1fpx", gs, gs)
				}),
				el.DynStyle("background-position", func() string {
					return fmt.Sprintf("%.1fpx %.1fpx", panX(), panY())
				}),
				// Pan on pointerdown on canvas background
				el.On("pointerdown", func(e dom.Event) {
					target := e.Value.Get("target")
					// Only start pan if clicking directly on canvas or world (not on nodes)
					if target.Equal(canvasEl.Value) || target.Equal(worldEl.Value) {
						startCanvasPan(e)
					}
				}),
				// Zoom on wheel
				el.On("wheel", func(e dom.Event) {
					e.PreventDefault()
					dy := e.Value.Get("deltaY").Float()
					factor := 1.0
					if dy < 0 {
						factor = 1.1
					} else {
						factor = 0.9
					}
					oldZ := zoom()
					newZ := oldZ * factor
					if newZ < minZoom {
						newZ = minZoom
					}
					if newZ > maxZoom {
						newZ = maxZoom
					}
					// Zoom centered on cursor
					rect := canvasEl.Value.Call("getBoundingClientRect")
					mx := e.Value.Get("clientX").Float() - rect.Get("left").Float()
					my := e.Value.Get("clientY").Float() - rect.Get("top").Float()
					reactive.Batch(func() {
						setPanX(mx - (mx-panX())*(newZ/oldZ))
						setPanY(my - (my-panY())*(newZ/oldZ))
						setZoom(newZ)
					})
				}),
				// World container
				el.Div(
					el.Ref(&worldEl),
					el.Style("position", "absolute"),
					el.Style("top", "0"),
					el.Style("left", "0"),
					el.Style("width", "1px"),
					el.Style("height", "1px"),
					el.Style("transform-origin", "0 0"),
					el.DynStyle("transform", func() string {
						return fmt.Sprintf("translate(%.1fpx, %.1fpx) scale(%.4f)", panX(), panY(), zoom())
					}),
					// SVG connections layer
					el.Div(
						el.Ref(&svgContainerEl),
						el.Style("position", "absolute"),
						el.Style("top", "0"),
						el.Style("left", "0"),
						el.Style("width", "1px"),
						el.Style("height", "1px"),
						el.Style("overflow", "visible"),
						el.Style("pointer-events", "none"),
					),
					// Nodes layer
					el.Dynamic(func() el.Node {
						_ = nodeVer()
						args := []any{
							el.Style("position", "absolute"),
							el.Style("top", "0"),
							el.Style("left", "0"),
						}
						for _, nd := range nodes {
							args = append(args, renderNode(nd))
						}
						return el.Div(args...)
					}),
				),
			),
		),
		// Placing ghost overlay
		el.Show(func() bool { return placingType() != "" },
			el.Div(
				el.Class("fixed inset-0 z-50 pointer-events-none"),
				el.Div(
					el.Class("absolute top-4 left-1/2 -translate-x-1/2 bg-amber-500/90 text-slate-900 px-4 py-2 rounded-full text-sm font-semibold shadow-lg"),
					el.DynText(func() string {
						return fmt.Sprintf("Drop \"%s\" on canvas", placingType())
					}),
				),
			),
		),
	)
}

func main() {
	el.Mount("#app", App)
	select {}
}
