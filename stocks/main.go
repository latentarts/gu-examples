//go:build js && wasm

package main

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
)

// ---------------------------------------------------------------------------
// Split Tree Layout
// ---------------------------------------------------------------------------

type splitDir int

const (
	dirHorizontal splitDir = iota // side by side (left | right)
	dirVertical                   // stacked (top / bottom)
)

type splitNode struct {
	isLeaf bool
	// Leaf fields
	panel *panelState
	// Split fields
	dir    splitDir
	ratio  float64 // 0.0–1.0, fraction for first child
	first  *splitNode
	second *splitNode
}

type panelState struct {
	id               int
	symbol           string
	name             string
	candles          []StockQuote
	rng              *pseudoRand
	candleVersion    func() int
	setCandleVersion func(int)
	cancelFn         func() // interval cancel
	chartEl          dom.Element // direct reference for innerHTML updates
}

// Global state
var (
	nextPanelID int
	root        *splitNode

	// Version-counter for re-rendering the tree
	treeVersion    func() int
	setTreeVersion func(int)

	// Selected panel for close
	selectedPanel    func() int
	setSelectedPanel func(int)

	// Command palette state
	paletteOpen    func() bool
	setPaletteOpen func(bool)

	// Placement mode
	placementActive    func() bool
	setPlacementActive func(bool)
	placementSymbol    string
	placementName      string
	placementDir       func() int
	setPlacementDir    func(int)
	placementTarget    *splitNode // the leaf to split
)

func bumpTree() {
	setTreeVersion(treeVersion() + 1)
}

// ---------------------------------------------------------------------------
// Panel management
// ---------------------------------------------------------------------------

func newPanel(symbol, name string) *panelState {
	jsutil.LogInfo("adding stock panel: %s (%s)", symbol, name)
	nextPanelID++
	rng := newRand(simpleHash(symbol) + uint64(nextPanelID))
	candles := GenerateCandles(symbol, 40)
	cv, setCv := reactive.NewSignal(0)
	p := &panelState{
		id:               nextPanelID,
		symbol:           symbol,
		name:             name,
		candles:          candles,
		rng:              rng,
		candleVersion:    cv,
		setCandleVersion: setCv,
	}
	startPanelUpdates(p)
	return p
}

func stopPanel(p *panelState) {
	if p != nil && p.cancelFn != nil {
		jsutil.LogInfo("removing stock panel: %s", p.symbol)
		p.cancelFn()
		p.cancelFn = nil
	}
}

func startPanelUpdates(p *panelState) {
	bump := func() { p.setCandleVersion(p.candleVersion() + 1) }

	// Fast tick (1s): update current candle's close price + header text
	cancelTick := jsutil.SetInterval(func() {
		if len(p.candles) == 0 {
			return
		}
		TickPrice(&p.candles[len(p.candles)-1], p.rng)
		bump() // triggers header price/change text (lightweight DynText)
	}, 1000)

	// Chart redraw (2s): directly set innerHTML, bypassing reactive system
	cancelChart := jsutil.SetInterval(func() {
		if len(p.candles) == 0 {
			return
		}
		if !p.chartEl.Value.IsUndefined() && !p.chartEl.Value.IsNull() && p.chartEl.Value.Truthy() {
			p.chartEl.SetInnerHTML(RenderChartHTML(p.candles))
		}
	}, 2000)

	// Slow tick (8s): finalize current candle and start a new one
	cancelCandle := jsutil.SetInterval(func() {
		if len(p.candles) == 0 {
			return
		}
		last := p.candles[len(p.candles)-1]
		next := NextCandle(last, p.rng)
		p.candles = append(p.candles, next)
		if len(p.candles) > 60 {
			p.candles = p.candles[len(p.candles)-60:]
		}
		bump()
		// Immediately redraw chart for new candle
		if !p.chartEl.Value.IsUndefined() && !p.chartEl.Value.IsNull() && p.chartEl.Value.Truthy() {
			p.chartEl.SetInnerHTML(RenderChartHTML(p.candles))
		}
	}, 8000)

	p.cancelFn = func() { cancelTick(); cancelChart(); cancelCandle() }
}

func countLeaves(node *splitNode) int {
	if node == nil {
		return 0
	}
	if node.isLeaf {
		return 1
	}
	return countLeaves(node.first) + countLeaves(node.second)
}

func findLeaf(node *splitNode, panelID int) *splitNode {
	if node == nil {
		return nil
	}
	if node.isLeaf && node.panel != nil && node.panel.id == panelID {
		return node
	}
	if r := findLeaf(node.first, panelID); r != nil {
		return r
	}
	return findLeaf(node.second, panelID)
}

func firstLeaf(node *splitNode) *splitNode {
	if node == nil {
		return nil
	}
	if node.isLeaf {
		return node
	}
	return firstLeaf(node.first)
}

// removePanel removes a leaf from the tree and collapses the parent.
func removePanel(panelID int) {
	if root == nil {
		return
	}
	if root.isLeaf {
		if root.panel != nil && root.panel.id == panelID {
			stopPanel(root.panel)
			root = nil
			bumpTree()
		}
		return
	}
	removeFromNode(nil, root, panelID)
}

func removeFromNode(parent, node *splitNode, panelID int) {
	if node == nil || node.isLeaf {
		return
	}
	// Check if first child is the target
	if node.first != nil && node.first.isLeaf && node.first.panel != nil && node.first.panel.id == panelID {
		stopPanel(node.first.panel)
		replaceNode(parent, node, node.second)
		bumpTree()
		return
	}
	// Check if second child is the target
	if node.second != nil && node.second.isLeaf && node.second.panel != nil && node.second.panel.id == panelID {
		stopPanel(node.second.panel)
		replaceNode(parent, node, node.first)
		bumpTree()
		return
	}
	removeFromNode(node, node.first, panelID)
	removeFromNode(node, node.second, panelID)
}

func replaceNode(parent, old, replacement *splitNode) {
	if parent == nil {
		// old is root
		*root = *replacement
		return
	}
	if parent.first == old {
		parent.first = replacement
	} else if parent.second == old {
		parent.second = replacement
	}
}

// addPanel inserts a new panel by splitting the target leaf.
// dir: 0=left, 1=right, 2=top, 3=bottom
func addPanel(target *splitNode, panel *panelState, dir int) {
	if target == nil || !target.isLeaf {
		return
	}
	oldPanel := target.panel

	newLeaf := &splitNode{isLeaf: true, panel: panel}
	oldLeaf := &splitNode{isLeaf: true, panel: oldPanel}

	target.isLeaf = false
	target.panel = nil
	target.ratio = 0.5

	switch dir {
	case 0: // left
		target.dir = dirHorizontal
		target.first = newLeaf
		target.second = oldLeaf
	case 1: // right
		target.dir = dirHorizontal
		target.first = oldLeaf
		target.second = newLeaf
	case 2: // top
		target.dir = dirVertical
		target.first = newLeaf
		target.second = oldLeaf
	case 3: // bottom
		target.dir = dirVertical
		target.first = oldLeaf
		target.second = newLeaf
	}
	bumpTree()
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

func renderTree() el.Node {
	_ = treeVersion() // track
	if root == nil {
		return emptyState()
	}
	return renderNode(root, 0)
}

func emptyState() el.Node {
	return el.Div(
		el.Class("flex flex-col items-center justify-center h-screen bg-zinc-950 text-zinc-400 gap-4"),
		el.Div(
			el.Class("text-6xl opacity-20"),
			el.Text("$"),
		),
		el.Div(
			el.Class("text-lg font-medium text-zinc-300"),
			el.Text("Stock Market Monitor"),
		),
		el.Div(
			el.Class("text-sm"),
			el.Text("Press Ctrl+Space to add a stock"),
		),
		el.Button(
			el.Class("mt-4 px-5 py-2.5 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-500 transition-colors cursor-pointer"),
			el.Text("+ Add Stock"),
			el.OnClick(func(e dom.Event) {
				setPaletteOpen(true)
			}),
		),
		el.Div(
			el.Class("flex gap-2 mt-2"),
			kbdBadge("Ctrl+Space"),
			el.Span(el.Class("text-zinc-600 text-xs self-center"), el.Text("open command palette")),
		),
	)
}

func kbdBadge(key string) el.Node {
	return el.Span(
		el.Class("px-2 py-0.5 bg-zinc-800 text-zinc-400 rounded text-xs font-mono border border-zinc-700"),
		el.Text(key),
	)
}

func renderNode(node *splitNode, depth int) el.Node {
	if node.isLeaf {
		return renderPanel(node)
	}

	flexDir := "flex-row"
	if node.dir == dirVertical {
		flexDir = "flex-col"
	}

	return el.Div(
		el.Class("flex "+flexDir+" w-full h-full"),
		el.Div(
			el.Class("overflow-hidden"),
			el.DynStyle("flex", func() string {
				_ = treeVersion()
				return fmt.Sprintf("%.4f 1 0%%", node.ratio)
			}),
			el.Style("transition", "flex 0.3s ease"),
			el.Style("min-width", "0"),
			el.Style("min-height", "0"),
			renderNode(node.first, depth+1),
		),
		renderSplitHandle(node),
		el.Div(
			el.Class("overflow-hidden"),
			el.DynStyle("flex", func() string {
				_ = treeVersion()
				return fmt.Sprintf("%.4f 1 0%%", 1.0-node.ratio)
			}),
			el.Style("transition", "flex 0.3s ease"),
			el.Style("min-width", "0"),
			el.Style("min-height", "0"),
			renderNode(node.second, depth+1),
		),
	)
}

func renderSplitHandle(node *splitNode) el.Node {
	isHoriz := node.dir == dirHorizontal

	cursorClass := "cursor-col-resize"
	sizeClass := "w-1 hover:w-1.5"
	if !isHoriz {
		cursorClass = "cursor-row-resize"
		sizeClass = "h-1 hover:h-1.5"
	}

	var containerEl dom.Element

	startResize := func(e dom.Event) {
		e.PreventDefault()
		var startPos float64
		if isHoriz {
			startPos = e.Value.Get("clientX").Float()
		} else {
			startPos = e.Value.Get("clientY").Float()
		}
		startRatio := node.ratio

		bodyStyle := js.Global().Get("document").Get("body").Get("style")
		if isHoriz {
			bodyStyle.Set("cursor", "col-resize")
		} else {
			bodyStyle.Set("cursor", "row-resize")
		}
		bodyStyle.Set("userSelect", "none")

		// Get parent container size
		parentEl := containerEl.Value.Get("parentElement")

		var moveFn, upFn js.Func
		moveFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			var currentPos float64
			if isHoriz {
				currentPos = args[0].Get("clientX").Float()
			} else {
				currentPos = args[0].Get("clientY").Float()
			}
			delta := currentPos - startPos

			var containerSize float64
			if isHoriz {
				containerSize = parentEl.Get("offsetWidth").Float()
			} else {
				containerSize = parentEl.Get("offsetHeight").Float()
			}
			if containerSize == 0 {
				return nil
			}

			pct := startRatio + delta/containerSize
			if pct < 0.15 {
				pct = 0.15
			}
			if pct > 0.85 {
				pct = 0.85
			}
			node.ratio = pct
			bumpTree()
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

	return el.Div(
		el.Class(sizeClass+" "+cursorClass+" bg-zinc-800 hover:bg-zinc-600 transition-colors flex-shrink-0"),
		el.Style("touch-action", "none"),
		el.Ref(&containerEl),
		el.On("pointerdown", startResize),
	)
}

func renderPanel(node *splitNode) el.Node {
	p := node.panel
	if p == nil {
		return el.Div()
	}

	hovered, setHovered := reactive.NewSignal(false)
	candleVersion := p.candleVersion

	// Price info
	priceText := reactive.CreateMemo(func() string {
		_ = candleVersion()
		if len(p.candles) == 0 {
			return "--"
		}
		return fmt.Sprintf("%.2f", p.candles[len(p.candles)-1].Close)
	})

	changeText := reactive.CreateMemo(func() string {
		_ = candleVersion()
		if len(p.candles) < 2 {
			return ""
		}
		last := p.candles[len(p.candles)-1].Close
		first := p.candles[0].Open
		pctChange := ((last - first) / first) * 100
		if pctChange >= 0 {
			return fmt.Sprintf("+%.2f%%", pctChange)
		}
		return fmt.Sprintf("%.2f%%", pctChange)
	})

	changeColor := reactive.CreateMemo(func() string {
		_ = candleVersion()
		if len(p.candles) < 2 {
			return "text-zinc-500"
		}
		last := p.candles[len(p.candles)-1].Close
		first := p.candles[0].Open
		if last >= first {
			return "text-green-500"
		}
		return "text-red-500"
	})

	return el.Div(
		el.DynClass(func() string {
			if selectedPanel() == p.id {
				return "w-full h-full flex flex-col bg-zinc-950 border-2 border-blue-500 overflow-hidden"
			}
			return "w-full h-full flex flex-col bg-zinc-950 border border-zinc-800 overflow-hidden"
		}),
		el.OnClick(func(e dom.Event) {
			e.StopPropagation()
			setSelectedPanel(p.id)
		}),
		el.On("mouseenter", func(e dom.Event) { setHovered(true) }),
		el.On("mouseleave", func(e dom.Event) { setHovered(false) }),

		// Header
		el.Div(
			el.Class("flex items-center justify-between px-3 py-2 border-b border-zinc-800 flex-shrink-0"),
			el.Div(
				el.Class("flex items-center gap-3"),
				el.Span(
					el.Class("text-sm font-bold text-zinc-100"),
					el.Text(p.symbol),
				),
				el.Span(
					el.Class("text-xs text-zinc-500 hidden sm:inline"),
					el.Text(p.name),
				),
			),
			el.Div(
				el.Class("flex items-center gap-3"),
				// Price
				el.Span(
					el.Class("text-sm font-mono font-semibold text-zinc-100"),
					el.DynText(priceText),
				),
				// Change %
				el.Span(
					el.DynClass(func() string {
						return "text-xs font-mono " + changeColor()
					}),
					el.DynText(changeText),
				),
				// Close button (visible on hover)
				el.Show(
					func() bool { return hovered() },
					el.Button(
						el.Class("w-5 h-5 flex items-center justify-center rounded text-zinc-500 hover:text-zinc-100 hover:bg-zinc-700 transition-colors text-xs"),
						el.Text("\u2715"),
						el.OnClick(func(e dom.Event) {
							e.StopPropagation()
							removePanel(p.id)
						}),
					),
				),
			),
		),

		// Chart area — stable div, innerHTML updated directly by interval timer.
		// No el.Dynamic = no DOM teardown/rebuild on each tick.
		el.Div(
			el.Class("flex-1 p-1 min-h-0"),
			el.OnMount(func(elem dom.Element) {
				p.chartEl = elem
				// Initial render
				if len(p.candles) > 0 {
					elem.SetInnerHTML(RenderChartHTML(p.candles))
				}
			}),
		),
	)
}

// ---------------------------------------------------------------------------
// Command Palette
// ---------------------------------------------------------------------------

func commandPalette() el.Node {
	query, setQuery := reactive.NewSignal("")
	selectedIdx, setSelectedIdx := reactive.NewSignal(0)

	filterStocks := func() []StockInfo {
		q := strings.ToUpper(query())
		if q == "" {
			return PopularStocks[:15] // show top 15 by default
		}
		var out []StockInfo
		for _, s := range PopularStocks {
			if strings.Contains(s.Symbol, q) || strings.Contains(strings.ToUpper(s.Name), q) {
				out = append(out, s)
			}
			if len(out) >= 15 {
				break
			}
		}
		return out
	}

	selectStock := func(s StockInfo) {
		setPaletteOpen(false)
		setQuery("")
		setSelectedIdx(0)

		if root == nil {
			// First panel — fullscreen
			p := newPanel(s.Symbol, s.Name)
			root = &splitNode{isLeaf: true, panel: p}
			bumpTree()
			return
		}

		// Enter placement mode
		placementSymbol = s.Symbol
		placementName = s.Name
		setPlacementDir(1) // default: right
		// Find target: selected panel or first leaf
		if sel := selectedPanel(); sel > 0 {
			if leaf := findLeaf(root, sel); leaf != nil {
				placementTarget = leaf
			} else {
				placementTarget = firstLeaf(root)
			}
		} else {
			placementTarget = firstLeaf(root)
		}
		setPlacementActive(true)
	}

	var inputEl dom.Element

	return el.Show(
		func() bool { return paletteOpen() },
		el.Div(
			el.Class("fixed inset-0 z-50 flex items-start justify-center pt-[20vh]"),

			// Backdrop
			el.Div(
				el.Class("absolute inset-0 bg-black/60 animate-fade-in"),
				el.OnClick(func(e dom.Event) {
					setPaletteOpen(false)
					setQuery("")
					setSelectedIdx(0)
				}),
			),

			// Card
			el.Div(
				el.Class("relative z-10 w-full max-w-md bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl animate-scale-up overflow-hidden"),

				// Input
				el.Div(
					el.Class("p-3 border-b border-zinc-800"),
					el.Tag("input",
						el.Class("w-full bg-zinc-800 text-zinc-100 text-sm rounded-lg px-3 py-2.5 outline-none placeholder-zinc-500 focus:ring-2 focus:ring-blue-500 border border-zinc-700"),
						el.Attr("type", "text"),
						el.Attr("placeholder", "Search stocks... (e.g. AAPL, Tesla)"),
						el.Attr("autofocus", "true"),
						el.Ref(&inputEl),
						el.OnMount(func(elem dom.Element) {
							elem.Focus()
						}),
						el.OnInput(func(e dom.Event) {
							setQuery(e.TargetValue())
							setSelectedIdx(0)
						}),
						el.OnKeyDown(func(e dom.Event) {
							key := e.Value.Get("key").String()
							switch key {
							case "ArrowDown":
								e.PreventDefault()
								f := filterStocks()
								if idx := selectedIdx(); idx < len(f)-1 {
									setSelectedIdx(idx + 1)
								}
							case "ArrowUp":
								e.PreventDefault()
								if idx := selectedIdx(); idx > 0 {
									setSelectedIdx(idx - 1)
								}
							case "Enter":
								e.PreventDefault()
								f := filterStocks()
								idx := selectedIdx()
								if idx < len(f) {
									selectStock(f[idx])
								}
							case "Escape":
								e.PreventDefault()
								setPaletteOpen(false)
								setQuery("")
								setSelectedIdx(0)
							}
						}),
					),
				),

				// Results
				el.Div(
					el.Class("max-h-72 overflow-y-auto"),
					el.Dynamic(func() el.Node {
						f := filterStocks()
						sel := selectedIdx()
						if len(f) == 0 {
							return el.Div(
								el.Class("px-4 py-8 text-center text-zinc-500 text-sm"),
								el.Text("No stocks found"),
							)
						}
						items := []any{el.Class("py-1")}
						for i, s := range f {
							idx := i
							stock := s
							cls := "flex items-center justify-between px-4 py-2.5 cursor-pointer transition-colors "
							if idx == sel {
								cls += "bg-zinc-800 text-zinc-100"
							} else {
								cls += "text-zinc-300 hover:bg-zinc-800/50"
							}
							items = append(items, el.Div(
								el.Class(cls),
								el.OnClick(func(e dom.Event) {
									selectStock(stock)
								}),
								el.On("mouseenter", func(e dom.Event) {
									setSelectedIdx(idx)
								}),
								el.Div(
									el.Class("flex items-center gap-3"),
									el.Span(el.Class("text-sm font-bold font-mono min-w-[55px]"), el.Text(stock.Symbol)),
									el.Span(el.Class("text-xs text-zinc-500"), el.Text(stock.Name)),
								),
								el.Span(el.Class("text-xs text-zinc-600"), el.Text(stock.Sector)),
							))
						}
						return el.Div(items...)
					}),
				),

				// Footer hint
				el.Div(
					el.Class("px-4 py-2 border-t border-zinc-800 flex items-center gap-4 text-xs text-zinc-600"),
					el.Span(el.Text("↑↓ navigate")),
					el.Span(el.Text("↵ select")),
					el.Span(el.Text("esc close")),
				),
			),
		),
	)
}

// ---------------------------------------------------------------------------
// Placement Mode Overlay
// ---------------------------------------------------------------------------

func placementOverlay() el.Node {
	dirLabels := []string{"← Left", "→ Right", "↑ Top", "↓ Bottom"}

	confirmPlacement := func() {
		if placementTarget == nil {
			setPlacementActive(false)
			return
		}
		p := newPanel(placementSymbol, placementName)
		addPanel(placementTarget, p, placementDir())
		setPlacementActive(false)
	}

	cancelPlacement := func() {
		setPlacementActive(false)
	}

	return el.Show(
		func() bool { return placementActive() },
		el.Div(
			el.Class("fixed inset-0 z-40 flex items-center justify-center"),

			// Semi-transparent overlay with placement preview
			el.Div(
				el.Class("absolute inset-0 bg-black/40 animate-fade-in"),
			),

			// Placement indicator card
			el.Div(
				el.Class("relative z-10 bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl p-6 animate-scale-up min-w-[320px]"),

				el.Div(
					el.Class("text-center mb-4"),
					el.Div(
						el.Class("text-sm font-bold text-zinc-100 mb-1"),
						el.Text(fmt.Sprintf("Placing %s", placementSymbol)),
					),
					el.Div(
						el.Class("text-xs text-zinc-500"),
						el.Text("Choose split direction"),
					),
				),

				// Visual direction preview
				el.Div(
					el.Class("mb-4"),
					el.Dynamic(func() el.Node {
						d := placementDir()
						return directionPreview(d)
					}),
				),

				// Direction label
				el.Div(
					el.Class("text-center mb-4"),
					el.DynText(func() string {
						return dirLabels[placementDir()]
					}),
					el.DynClass(func() string {
						return "text-sm font-medium text-blue-400"
					}),
				),

				// Direction buttons
				el.Div(
					el.Class("grid grid-cols-4 gap-2 mb-4"),
					dirButton("←", 0),
					dirButton("→", 1),
					dirButton("↑", 2),
					dirButton("↓", 3),
				),

				// Action buttons
				el.Div(
					el.Class("flex gap-2"),
					el.Button(
						el.Class("flex-1 py-2 bg-zinc-800 text-zinc-300 rounded-lg text-sm font-medium hover:bg-zinc-700 transition-colors"),
						el.Text("Cancel"),
						el.OnClick(func(e dom.Event) { cancelPlacement() }),
					),
					el.Button(
						el.Class("flex-1 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-500 transition-colors"),
						el.Text("Confirm"),
						el.OnClick(func(e dom.Event) { confirmPlacement() }),
					),
				),

				// Keyboard hints
				el.Div(
					el.Class("mt-3 text-center text-xs text-zinc-600"),
					el.Text("Arrow keys to pick direction · Enter to confirm · Esc to cancel"),
				),
			),
		),
	)
}

func dirButton(label string, dir int) el.Node {
	return el.Button(
		el.DynClass(func() string {
			base := "py-2 rounded-lg text-sm font-medium transition-colors "
			if placementDir() == dir {
				return base + "bg-blue-600 text-white"
			}
			return base + "bg-zinc-800 text-zinc-400 hover:bg-zinc-700"
		}),
		el.Text(label),
		el.OnClick(func(e dom.Event) {
			setPlacementDir(dir)
		}),
	)
}

func directionPreview(dir int) el.Node {
	// A small visual showing the split direction
	// Blue = new panel, gray = existing panel
	newCls := "bg-blue-500/30 border border-blue-500/50 rounded flex items-center justify-center text-xs text-blue-400"
	oldCls := "bg-zinc-800 border border-zinc-700 rounded flex items-center justify-center text-xs text-zinc-500"

	newLabel := el.Span(el.Class("text-xs"), el.Text("NEW"))
	oldLabel := el.Span(el.Class("text-xs"), el.Text("CUR"))

	switch dir {
	case 0: // left
		return el.Div(
			el.Class("flex gap-1 h-24"),
			el.Div(el.Class("flex-1 "+newCls), newLabel),
			el.Div(el.Class("flex-1 "+oldCls), oldLabel),
		)
	case 1: // right
		return el.Div(
			el.Class("flex gap-1 h-24"),
			el.Div(el.Class("flex-1 "+oldCls), oldLabel),
			el.Div(el.Class("flex-1 "+newCls), newLabel),
		)
	case 2: // top
		return el.Div(
			el.Class("flex flex-col gap-1 h-24"),
			el.Div(el.Class("flex-1 "+newCls), newLabel),
			el.Div(el.Class("flex-1 "+oldCls), oldLabel),
		)
	case 3: // bottom
		return el.Div(
			el.Class("flex flex-col gap-1 h-24"),
			el.Div(el.Class("flex-1 "+oldCls), oldLabel),
			el.Div(el.Class("flex-1 "+newCls), newLabel),
		)
	}
	return el.Div()
}

// ---------------------------------------------------------------------------
// Status Bar
// ---------------------------------------------------------------------------

func statusBar() el.Node {
	return el.Div(
		el.Class("flex items-center justify-between px-4 py-1.5 bg-zinc-900 border-t border-zinc-800 flex-shrink-0"),
		el.Div(
			el.Class("flex items-center gap-4 text-xs text-zinc-500"),
			el.Span(
				el.DynText(func() string {
					_ = treeVersion()
					n := countLeaves(root)
					if n == 0 {
						return "No panels"
					}
					if n == 1 {
						return "1 panel"
					}
					return fmt.Sprintf("%d panels", n)
				}),
			),
		),
		el.Div(
			el.Class("flex items-center gap-3 text-xs text-zinc-600"),
			el.Button(
				el.Class("px-2 py-0.5 bg-zinc-800 text-zinc-300 rounded text-xs hover:bg-zinc-700 transition-colors cursor-pointer border border-zinc-700"),
				el.Text("+ Add"),
				el.OnClick(func(e dom.Event) { setPaletteOpen(true) }),
			),
			el.Span(el.Text("Ctrl+Space add")),
			el.Span(el.Text("Ctrl+X close")),
			el.Span(el.Text("Click select")),
		),
	)
}

// ---------------------------------------------------------------------------
// App
// ---------------------------------------------------------------------------

func App() el.Node {
	jsutil.LogInfo("stock market monitor started")
	treeVersion, setTreeVersion = reactive.NewSignal(0)
	selectedPanel, setSelectedPanel = reactive.NewSignal(0)
	paletteOpen, setPaletteOpen = reactive.NewSignal(false)
	placementActive, setPlacementActive = reactive.NewSignal(false)
	placementDir, setPlacementDir = reactive.NewSignal(1)

	// Global keyboard handler
	keydownFn := js.FuncOf(func(_ js.Value, args []js.Value) any {
		evt := args[0]
		key := evt.Get("key").String()
		ctrl := evt.Get("ctrlKey").Bool() || evt.Get("metaKey").Bool()

		// Placement mode keys
		if placementActive() {
			switch key {
			case "ArrowLeft":
				evt.Call("preventDefault")
				setPlacementDir(0)
				return nil
			case "ArrowRight":
				evt.Call("preventDefault")
				setPlacementDir(1)
				return nil
			case "ArrowUp":
				evt.Call("preventDefault")
				setPlacementDir(2)
				return nil
			case "ArrowDown":
				evt.Call("preventDefault")
				setPlacementDir(3)
				return nil
			case "Enter":
				evt.Call("preventDefault")
				if placementTarget != nil {
					p := newPanel(placementSymbol, placementName)
					addPanel(placementTarget, p, placementDir())
				}
				setPlacementActive(false)
				return nil
			case "Escape":
				evt.Call("preventDefault")
				setPlacementActive(false)
				return nil
			}
			return nil
		}

		// Ctrl+Space — toggle command palette
		code := evt.Get("code").String()
		if ctrl && (key == " " || code == "Space") {
			evt.Call("preventDefault")
			setPaletteOpen(!paletteOpen())
			return nil
		}

		// Ctrl+X — close selected panel
		if ctrl && (key == "x" || key == "X") {
			if !paletteOpen() {
				evt.Call("preventDefault")
				if sel := selectedPanel(); sel > 0 {
					removePanel(sel)
					setSelectedPanel(0)
				}
			}
			return nil
		}

		// Escape — close palette
		if key == "Escape" && paletteOpen() {
			setPaletteOpen(false)
			return nil
		}

		return nil
	})
	js.Global().Get("document").Call("addEventListener", "keydown", keydownFn)

	return el.Div(
		el.Class("h-screen flex flex-col bg-zinc-950"),

		// Main content area
		el.Div(
			el.Class("flex-1 min-h-0 overflow-hidden"),
			el.Dynamic(renderTree),
		),

		// Status bar
		statusBar(),

		// Overlays
		commandPalette(),
		placementOverlay(),
	)
}

func main() {
	el.Mount("#app", App)
	select {}
}
