//go:build js && wasm

package components

import (
	"fmt"
	"syscall/js"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/nodegraph-tauri/frontend/state"
)

func Editor(styles el.Node, s *state.EditorState) el.Node {
	updateSVG := func() {
		if s.SVGContainerEl.IsNull() {
			return
		}
		content := state.BuildSVGContent(s.Nodes, s.Conns, s.TempX1(), s.TempY1(), s.TempX2(), s.TempY2(), s.TempActive())
		s.SVGContainerEl.SetInnerHTML(
			`<svg xmlns="http://www.w3.org/2000/svg" style="position:absolute;top:0;left:0;width:100%;height:100%;overflow:visible;pointer-events:none">` + content + `</svg>`)
	}

	reactive.CreateEffect(func() {
		_ = s.ConnVer()
		_ = s.TempActive()
		_ = s.TempX1()
		_ = s.TempY1()
		_ = s.TempX2()
		_ = s.TempY2()
		updateSVG()
	})

	centerGraph := func() {
		if s.CanvasEl.IsNull() || len(s.Nodes) == 0 {
			return
		}
		cw := s.CanvasEl.GetProperty("offsetWidth").Float()
		ch := s.CanvasEl.GetProperty("offsetHeight").Float()
		minX, minY, maxX, maxY := state.CenterGraphBounds(s.Nodes)
		px, py, z := state.CalcCenterView(minX, minY, maxX, maxY, cw, ch)
		reactive.Batch(func() {
			s.SetZoom(z)
			s.SetPanX(px)
			s.SetPanY(py)
		})
	}

	deleteNodeHandler := func(id string) {
		s.Nodes, s.Conns = state.DeleteNode(s.Nodes, s.Conns, id)
		reactive.Batch(func() {
			s.BumpNodes()
			s.BumpConns()
		})
	}

	startNodeDrag := func(id string, e dom.Event) {
		e.StopPropagation()
		e.PreventDefault()
		idx := state.FindNodeIdx(s.Nodes, id)
		if idx < 0 {
			return
		}
		startMX := e.Value.Get("clientX").Float()
		startMY := e.Value.Get("clientY").Float()
		startNX := s.Nodes[idx].X
		startNY := s.Nodes[idx].Y
		z := s.Zoom()
		nodeEl := js.Global().Get("document").Call("querySelector", fmt.Sprintf(`[data-node-id="%s"]`, id))
		if nodeEl.IsNull() || nodeEl.IsUndefined() {
			return
		}
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
			s.Nodes[idx].X = newX
			s.Nodes[idx].Y = newY
			nodeEl.Get("style").Set("left", fmt.Sprintf("%.1fpx", newX))
			nodeEl.Get("style").Set("top", fmt.Sprintf("%.1fpx", newY))
			updateSVG()
			return nil
		})
		upFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			js.Global().Call("removeEventListener", "pointermove", moveFn)
			js.Global().Call("removeEventListener", "pointerup", upFn)
			moveFn.Release()
			upFn.Release()
			nodeEl.Get("style").Set("transform", "scale(1)")
			nodeEl.Get("style").Set("boxShadow", "0 4px 12px rgba(0,0,0,0.3)")
			nodeEl.Get("style").Set("zIndex", "")
			bodyStyle.Set("cursor", "")
			bodyStyle.Set("userSelect", "")
			s.BumpNodes()
			return nil
		})
		js.Global().Call("addEventListener", "pointermove", moveFn)
		js.Global().Call("addEventListener", "pointerup", upFn)
	}

	startPortDrag := func(nodeID string, isOutput bool, portIdx int, e dom.Event) {
		e.StopPropagation()
		e.PreventDefault()
		nIdx := state.FindNodeIdx(s.Nodes, nodeID)
		if nIdx < 0 {
			return
		}
		ox, oy := state.PortWorldXY(s.Nodes[nIdx], isOutput, portIdx)
		s.SetTempX1(ox)
		s.SetTempY1(oy)
		s.SetTempX2(ox)
		s.SetTempY2(oy)
		s.SetTempActive(true)

		z := s.Zoom()
		px := s.PanX()
		py := s.PanY()
		var moveFn, upFn js.Func
		moveFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			mx := args[0].Get("clientX").Float()
			my := args[0].Get("clientY").Float()
			rect := s.CanvasEl.Value.Call("getBoundingClientRect")
			wx := (mx - rect.Get("left").Float() - px) / z
			wy := (my - rect.Get("top").Float() - py) / z
			s.SetTempX2(wx)
			s.SetTempY2(wy)
			return nil
		})
		upFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			js.Global().Call("removeEventListener", "pointermove", moveFn)
			js.Global().Call("removeEventListener", "pointerup", upFn)
			moveFn.Release()
			upFn.Release()
			s.SetTempActive(false)
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
					if isOutput && !tOut {
						var ok bool
						s.Conns, ok = state.AddConnection(s.Conns, nodeID, portIdx, tNodeID, tIdx, true)
						if ok {
							s.BumpConns()
						}
					} else if !isOutput && tOut {
						var ok bool
						s.Conns, ok = state.AddConnection(s.Conns, nodeID, portIdx, tNodeID, tIdx, false)
						if ok {
							s.BumpConns()
						}
					}
				}
			}
			return nil
		})
		js.Global().Call("addEventListener", "pointermove", moveFn)
		js.Global().Call("addEventListener", "pointerup", upFn)
	}

	startCanvasPan := func(e dom.Event) {
		if e.Value.Get("button").Int() != 0 {
			return
		}
		e.PreventDefault()
		startMX := e.Value.Get("clientX").Float()
		startMY := e.Value.Get("clientY").Float()
		startPX := s.PanX()
		startPY := s.PanY()
		bodyStyle := js.Global().Get("document").Get("body").Get("style")
		bodyStyle.Set("cursor", "grabbing")
		bodyStyle.Set("userSelect", "none")
		var moveFn, upFn js.Func
		moveFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			dx := args[0].Get("clientX").Float() - startMX
			dy := args[0].Get("clientY").Float() - startMY
			reactive.Batch(func() {
				s.SetPanX(startPX + dx)
				s.SetPanY(startPY + dy)
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

	renderNode := func(nd state.NodeData) el.Node {
		nt := state.CatalogByName(nd.Type)
		portCircle := func(nodeID string, isOutput bool, idx int, label string) el.Node {
			circleStyles := map[string]string{
				"position":      "absolute",
				"top":           "50%",
				"transform":     "translateY(-50%)",
				"width":         fmt.Sprintf("%dpx", state.PortR*2),
				"height":        fmt.Sprintf("%dpx", state.PortR*2),
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
				el.Style("height", fmt.Sprintf("%dpx", state.PortRowH)),
				el.Style("display", "flex"),
				el.Style("align-items", "center"),
				el.Div(
					el.Attr("data-port-node", nodeID),
					el.Attr("data-port-idx", fmt.Sprintf("%d", idx)),
					el.Attr("data-port-output", fmt.Sprintf("%v", isOutput)),
					el.StyleMap(circleStyles),
					el.On("pointerdown", func(e dom.Event) { startPortDrag(nodeID, isOutput, idx, e) }),
				),
				el.Span(el.StyleMap(labelStyles), el.Text(label)),
			)
		}

		maxPorts := len(nt.Inputs)
		if len(nt.Outputs) > maxPorts {
			maxPorts = len(nt.Outputs)
		}
		portArgs := []any{el.Style("padding", fmt.Sprintf("%dpx 0", state.PortPadY))}
		for i := 0; i < maxPorts; i++ {
			rowArgs := []any{
				el.Style("display", "flex"),
				el.Style("height", fmt.Sprintf("%dpx", state.PortRowH)),
			}
			if i < len(nt.Inputs) {
				rowArgs = append(rowArgs, el.Div(el.Style("flex", "1"), portCircle(nd.ID, false, i, nt.Inputs[i])))
			} else {
				rowArgs = append(rowArgs, el.Div(el.Style("flex", "1")))
			}
			if i < len(nt.Outputs) {
				rowArgs = append(rowArgs, el.Div(el.Style("flex", "1"), portCircle(nd.ID, true, i, nt.Outputs[i])))
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
			el.Style("width", fmt.Sprintf("%dpx", state.NodeW)),
			el.Style("background", "#1e293b"),
			el.Style("border-radius", "10px"),
			el.Style("border", fmt.Sprintf("1.5px solid %s", nt.Color)),
			el.Style("box-shadow", "0 4px 12px rgba(0,0,0,0.3)"),
			el.Style("cursor", "grab"),
			el.Style("user-select", "none"),
			el.Style("transition", "transform 0.15s ease, box-shadow 0.15s ease"),
			el.Style("overflow", "visible"),
			el.On("pointerdown", func(e dom.Event) {
				target := e.Value.Get("target")
				if target.Call("hasAttribute", "data-port-node").Bool() {
					return
				}
				startNodeDrag(nd.ID, e)
			}),
			el.Div(
				el.Style("height", fmt.Sprintf("%dpx", state.HeaderH)),
				el.Style("background", nt.Color),
				el.Style("border-radius", "8px 8px 0 0"),
				el.Style("display", "flex"),
				el.Style("align-items", "center"),
				el.Style("padding", "0 10px"),
				el.Style("justify-content", "space-between"),
				el.Span(el.Style("font-weight", "700"), el.Style("font-size", "13px"), el.Style("color", "#0f172a"), el.Text(fmt.Sprintf("%s  %s", nt.Icon, nt.Name))),
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
					el.On("pointerdown", func(e dom.Event) { e.StopPropagation() }),
					el.OnClick(func(e dom.Event) {
						e.StopPropagation()
						deleteNodeHandler(nd.ID)
					}),
				),
			),
			el.Div(portArgs...),
		)
	}

	renderDrawerItem := func(nt state.NodeTypeDef) el.Node {
		return el.Div(
			el.Class("flex items-center gap-3 p-2.5 rounded-lg cursor-grab hover:bg-slate-700/50 transition-colors"),
			el.Style("user-select", "none"),
			el.Style("touch-action", "none"),
			el.On("pointerdown", func(e dom.Event) {
				e.PreventDefault()
				s.SetPlacingType(nt.Name)
				var moveFn, upFn js.Func
				moveFn = js.FuncOf(func(_ js.Value, args []js.Value) any { return nil })
				upFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
					js.Global().Call("removeEventListener", "pointermove", moveFn)
					js.Global().Call("removeEventListener", "pointerup", upFn)
					moveFn.Release()
					upFn.Release()
					s.SetPlacingType("")
					dropX := args[0].Get("clientX").Float()
					dropY := args[0].Get("clientY").Float()
					if s.CanvasEl.IsNull() {
						return nil
					}
					rect := s.CanvasEl.Value.Call("getBoundingClientRect")
					cl := rect.Get("left").Float()
					ct := rect.Get("top").Float()
					cr := rect.Get("right").Float()
					cb := rect.Get("bottom").Float()
					if dropX >= cl && dropX <= cr && dropY >= ct && dropY <= cb {
						z := s.Zoom()
						wx := (dropX - cl - s.PanX()) / z
						wy := (dropY - ct - s.PanY()) / z
						s.Nodes = state.AddNode(s.Nodes, nt.Name, wx-state.NodeW/2, wy-30)
						s.BumpNodes()
					}
					return nil
				})
				js.Global().Call("addEventListener", "pointermove", moveFn)
				js.Global().Call("addEventListener", "pointerup", upFn)
			}),
			el.Div(el.Class("w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold"), el.Style("background", nt.Color), el.Style("color", "#0f172a"), el.Text(nt.Icon)),
			el.Div(
				el.Span(el.Class("text-sm font-semibold text-slate-200"), el.Text(nt.Name)),
				el.Div(el.Class("text-xs text-slate-400"), el.Text(fmt.Sprintf("%d in / %d out", len(nt.Inputs), len(nt.Outputs)))),
			),
		)
	}

	drawerItems := []any{
		el.Div(
			el.Class("p-4 border-b border-slate-700"),
			el.Div(
				el.Class("flex items-center justify-between"),
				el.H3(el.Class("text-sm font-bold text-slate-300 uppercase tracking-wider"), el.Text("Nodes")),
				el.Button(el.Class("text-slate-400 hover:text-slate-200 text-lg leading-none p-1"), el.Text("\u2715"), el.OnClick(func(e dom.Event) { s.SetDrawerOpen(false) })),
			),
			el.P(el.Class("text-xs text-slate-500 mt-1"), el.Text("Drag onto canvas")),
		),
	}
	for _, nt := range state.Catalog {
		drawerItems = append(drawerItems, el.Div(el.Class("px-3 py-1"), renderDrawerItem(nt)))
	}

	drawerArgs := []any{
		el.Class("h-full bg-slate-800 border-r border-slate-700 flex flex-col overflow-y-auto"),
		el.DynStyle("width", func() string {
			if s.DrawerOpen() {
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
		styles,
		el.Class("flex h-screen bg-slate-900 text-slate-200 font-sans"),
		el.Div(drawerArgs...),
		el.Div(
			el.Class("flex-1 flex flex-col"),
			el.Div(
				el.Class("h-12 bg-slate-800 border-b border-slate-700 flex items-center px-4 gap-2"),
				el.Show(func() bool { return !s.DrawerOpen() },
					el.Button(el.Class("px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded text-sm font-medium transition-colors"), el.Text("\u2630 Nodes"), el.OnClick(func(e dom.Event) { s.SetDrawerOpen(true) })),
				),
				el.Button(el.Class("px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded text-sm font-medium transition-colors"), el.Text("\u2316 Center"), el.OnClick(func(e dom.Event) { centerGraph() })),
				el.Button(el.Class("px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded text-sm font-medium transition-colors"), el.Text("+ Zoom In"), el.OnClick(func(e dom.Event) {
					z := s.Zoom() * 1.25
					if z > state.MaxZoom {
						z = state.MaxZoom
					}
					s.SetZoom(z)
				})),
				el.Button(el.Class("px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded text-sm font-medium transition-colors"), el.Text("\u2212 Zoom Out"), el.OnClick(func(e dom.Event) {
					z := s.Zoom() * 0.8
					if z < state.MinZoom {
						z = state.MinZoom
					}
					s.SetZoom(z)
				})),
				el.Div(el.Class("flex-1")),
				el.DynText(func() string { return fmt.Sprintf("%.0f%%", s.Zoom()*100) }),
				el.Span(el.Class("text-slate-500 text-xs ml-2"), el.DynText(func() string {
					_ = s.NodeVer()
					return fmt.Sprintf("%d nodes", len(s.Nodes))
				})),
				el.Span(el.Class("text-slate-500 text-xs ml-2"), el.DynText(func() string {
					_ = s.ConnVer()
					return fmt.Sprintf("%d connections", len(s.Conns))
				})),
			),
			el.Div(
				el.Ref(&s.CanvasEl),
				el.Class("flex-1 relative overflow-hidden"),
				el.Style("background", "#0f172a"),
				el.Style("cursor", "grab"),
				el.DynStyle("background-image", func() string {
					z := s.Zoom()
					ds := 2 * z
					if ds < 0.5 {
						ds = 0.5
					}
					return fmt.Sprintf("radial-gradient(circle, #334155 %.1fpx, transparent %.1fpx)", ds, ds)
				}),
				el.DynStyle("background-size", func() string {
					gs := 30 * s.Zoom()
					return fmt.Sprintf("%.1fpx %.1fpx", gs, gs)
				}),
				el.DynStyle("background-position", func() string {
					return fmt.Sprintf("%.1fpx %.1fpx", s.PanX(), s.PanY())
				}),
				el.On("pointerdown", func(e dom.Event) {
					target := e.Value.Get("target")
					if target.Equal(s.CanvasEl.Value) || target.Equal(s.WorldEl.Value) {
						startCanvasPan(e)
					}
				}),
				el.On("wheel", func(e dom.Event) {
					e.PreventDefault()
					dy := e.Value.Get("deltaY").Float()
					factor := 1.0
					if dy < 0 {
						factor = 1.1
					} else {
						factor = 0.9
					}
					oldZ := s.Zoom()
					newZ := oldZ * factor
					if newZ < state.MinZoom {
						newZ = state.MinZoom
					}
					if newZ > state.MaxZoom {
						newZ = state.MaxZoom
					}
					rect := s.CanvasEl.Value.Call("getBoundingClientRect")
					mx := e.Value.Get("clientX").Float() - rect.Get("left").Float()
					my := e.Value.Get("clientY").Float() - rect.Get("top").Float()
					reactive.Batch(func() {
						s.SetPanX(mx - (mx-s.PanX())*(newZ/oldZ))
						s.SetPanY(my - (my-s.PanY())*(newZ/oldZ))
						s.SetZoom(newZ)
					})
				}),
				el.Div(
					el.Ref(&s.WorldEl),
					el.Style("position", "absolute"),
					el.Style("top", "0"),
					el.Style("left", "0"),
					el.Style("width", "1px"),
					el.Style("height", "1px"),
					el.Style("transform-origin", "0 0"),
					el.DynStyle("transform", func() string {
						return fmt.Sprintf("translate(%.1fpx, %.1fpx) scale(%.4f)", s.PanX(), s.PanY(), s.Zoom())
					}),
					el.Div(
						el.Ref(&s.SVGContainerEl),
						el.Style("position", "absolute"),
						el.Style("top", "0"),
						el.Style("left", "0"),
						el.Style("width", "1px"),
						el.Style("height", "1px"),
						el.Style("overflow", "visible"),
						el.Style("pointer-events", "none"),
					),
					el.Dynamic(func() el.Node {
						_ = s.NodeVer()
						args := []any{el.Style("position", "absolute"), el.Style("top", "0"), el.Style("left", "0")}
						for _, nd := range s.Nodes {
							args = append(args, renderNode(nd))
						}
						return el.Div(args...)
					}),
				),
			),
		),
		el.Show(func() bool { return s.PlacingType() != "" },
			el.Div(
				el.Class("fixed inset-0 z-50 pointer-events-none"),
				el.Div(el.Class("absolute top-4 left-1/2 -translate-x-1/2 bg-amber-500/90 text-slate-900 px-4 py-2 rounded-full text-sm font-semibold shadow-lg"),
					el.DynText(func() string { return fmt.Sprintf("Drop \"%s\" on canvas", s.PlacingType()) }),
				),
			),
		),
	)
}
