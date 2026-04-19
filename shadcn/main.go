//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"
	"time"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
)

// ---------------------------------------------------------------------------
// App — showcase page
// ---------------------------------------------------------------------------

func App() el.Node {
	jsutil.LogInfo("shadcn/ui showcase mounted")
	return el.Div(
		el.Class("min-h-screen bg-white text-zinc-900 antialiased"),

		// Header
		el.Div(
			el.Class("border-b border-zinc-200"),
			el.Div(
				el.Class("max-w-4xl mx-auto px-6 py-10"),
				el.H1(el.Class("text-3xl font-bold tracking-tight"), el.Text("shadcn/ui \u2192 gu")),
				el.P(
					el.Class("text-zinc-500 mt-2 text-sm leading-relaxed max-w-2xl"),
					el.Text("Five React components from shadcn/ui faithfully reproduced in pure Go. No JavaScript framework, no virtual DOM \u2014 just gu\u2019s reactive signals and direct DOM updates."),
				),
			),
		),

		// Component demos
		el.Div(
			el.Class("max-w-4xl mx-auto px-6 py-12 space-y-20"),
			demoSection("Drawer", "A panel that slides up from the bottom. Drag the handle down to dismiss.", DrawerDemo()),
			demoSection("Date Picker", "A calendar popup for selecting dates with month navigation.", DatePickerDemo()),
			demoSection("Carousel", "A slideshow with prev/next arrows and dot indicators.", CarouselDemo()),
			demoSection("Button Group", "Toggle between options \u2014 one active at a time.", ButtonGroupDemo()),
			demoSection("Resizable Panels", "Drag the divider to resize. Panels enforce 15\u201385% bounds.", ResizableDemo()),
		),
	)
}

func demoSection(title, desc string, content el.Node) el.Node {
	return el.Div(
		el.H2(el.Class("text-lg font-semibold tracking-tight mb-1"), el.Text(title)),
		el.P(el.Class("text-sm text-zinc-500 mb-6"), el.Text(desc)),
		content,
	)
}

// ---------------------------------------------------------------------------
// Drawer
// ---------------------------------------------------------------------------

func DrawerDemo() el.Node {
	open, setOpen := reactive.NewSignal(false)
	offsetY, setOffsetY := reactive.NewSignal(0.0)
	dragging, setDragging := reactive.NewSignal(false)
	goal, setGoal := reactive.NewSignal(350)

	body := js.Global().Get("document").Get("body")

	closeDrawer := func() {
		reactive.Batch(func() {
			setDragging(false)
			setOpen(false)
			setOffsetY(0)
		})
		body.Get("classList").Call("remove", "drawer-open")
		jsutil.LogDebug("drawer closed")
	}

	openDrawer := func() {
		setOpen(true)
		body.Get("classList").Call("add", "drawer-open")
		jsutil.LogDebug("drawer opened")
	}

	handlePointerDown := func(e dom.Event) {
		e.PreventDefault()
		startY := e.Value.Get("clientY").Float()
		setDragging(true)

		var moveFn, upFn js.Func
		moveFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			dy := args[0].Get("clientY").Float() - startY
			if dy < 0 {
				dy = 0
			}
			setOffsetY(dy)
			return nil
		})
		upFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			js.Global().Call("removeEventListener", "pointermove", moveFn)
			js.Global().Call("removeEventListener", "pointerup", upFn)
			moveFn.Release()
			upFn.Release()
			if offsetY() > 120 {
				closeDrawer()
			} else {
				reactive.Batch(func() {
					setDragging(false)
					setOffsetY(0)
				})
			}
			return nil
		})
		js.Global().Call("addEventListener", "pointermove", moveFn)
		js.Global().Call("addEventListener", "pointerup", upFn)
	}

	return el.Div(
		// Trigger
		el.Button(
			el.Class("px-4 py-2.5 bg-zinc-900 text-white rounded-lg text-sm font-medium hover:bg-zinc-800 transition-colors"),
			el.Text("Open Drawer"),
			el.OnClick(func(e dom.Event) { openDrawer() }),
		),

		// Overlay — always in DOM, opacity-controlled for transitions
		el.Div(
			el.Class("fixed inset-0 z-40 bg-black/80"),
			el.DynStyle("opacity", func() string {
				if !open() {
					return "0"
				}
				o := 1.0 - offsetY()/500
				if o < 0 {
					o = 0
				}
				return fmt.Sprintf("%.2f", o)
			}),
			el.DynStyle("pointer-events", func() string {
				if open() {
					return "auto"
				}
				return "none"
			}),
			el.DynStyle("transition", func() string {
				if dragging() {
					return "none"
				}
				return "opacity 0.3s"
			}),
			el.OnClick(func(e dom.Event) { closeDrawer() }),
		),

		// Panel — always in DOM, transform-controlled for transitions
		el.Div(
			el.Class("fixed inset-x-0 bottom-0 z-50 bg-white rounded-t-2xl"),
			el.DynStyle("transform", func() string {
				if !open() {
					return "translateY(100%)"
				}
				if dragging() && offsetY() > 0 {
					return fmt.Sprintf("translateY(%.0fpx)", offsetY())
				}
				return "translateY(0)"
			}),
			el.DynStyle("transition", func() string {
				if dragging() {
					return "none"
				}
				return "transform 0.3s cubic-bezier(0.32, 0.72, 0, 1)"
			}),

			// Drag handle
			el.Div(
				el.Class("pt-4 pb-2 cursor-grab active:cursor-grabbing"),
				el.Style("touch-action", "none"),
				el.On("pointerdown", handlePointerDown),
				el.Div(el.Class("w-12 h-1.5 rounded-full bg-zinc-300 mx-auto")),
			),

			// Drawer content
			el.Div(
				el.Class("px-6 pb-8 pt-2"),
				el.H3(el.Class("text-lg font-semibold"), el.Text("Move Goal")),
				el.P(el.Class("text-sm text-zinc-500 mt-1 mb-8"), el.Text("Set your daily activity goal.")),

				// Goal adjuster
				el.Div(
					el.Class("flex items-center justify-center gap-8 mb-8"),
					el.Button(
						el.Class("w-12 h-12 rounded-full border border-zinc-200 text-xl flex items-center justify-center hover:bg-zinc-50 transition-colors"),
						el.Text("\u2212"), // minus
						el.OnClick(func(e dom.Event) {
							if g := goal(); g > 100 {
								setGoal(g - 10)
							}
						}),
					),
					el.Div(
						el.Class("text-center"),
						el.Div(
							el.Class("text-5xl font-bold tabular-nums"),
							el.DynText(func() string { return fmt.Sprintf("%d", goal()) }),
						),
						el.Div(el.Class("text-xs text-zinc-500 mt-1"), el.Text("calories / day")),
					),
					el.Button(
						el.Class("w-12 h-12 rounded-full border border-zinc-200 text-xl flex items-center justify-center hover:bg-zinc-50 transition-colors"),
						el.Text("+"),
						el.OnClick(func(e dom.Event) { setGoal(goal() + 10) }),
					),
				),

				el.Button(
					el.Class("w-full py-2.5 bg-zinc-900 text-white rounded-lg text-sm font-medium hover:bg-zinc-800 transition-colors"),
					el.Text("Submit"),
					el.OnClick(func(e dom.Event) { closeDrawer() }),
				),
			),
		),
	)
}

// ---------------------------------------------------------------------------
// Date Picker
// ---------------------------------------------------------------------------

func DatePickerDemo() el.Node {
	now := time.Now()
	selYear, setSelYear := reactive.NewSignal(now.Year())
	selMonth, setSelMonth := reactive.NewSignal(int(now.Month()))
	selDay, setSelDay := reactive.NewSignal(now.Day())
	viewYear, setViewYear := reactive.NewSignal(now.Year())
	viewMonth, setViewMonth := reactive.NewSignal(int(now.Month()))
	open, setOpen := reactive.NewSignal(false)

	navigateMonth := func(delta int) {
		d := time.Date(viewYear(), time.Month(viewMonth()), 1, 0, 0, 0, 0, time.UTC).AddDate(0, delta, 0)
		reactive.Batch(func() {
			setViewYear(d.Year())
			setViewMonth(int(d.Month()))
		})
	}

	selectDay := func(day int) {
		reactive.Batch(func() {
			setSelYear(viewYear())
			setSelMonth(viewMonth())
			setSelDay(day)
			setOpen(false)
		})
		jsutil.LogDebug("date selected: %d-%02d-%02d", viewYear(), viewMonth(), day)
	}

	return el.Div(
		el.Class("relative inline-block"),

		// Trigger button
		el.Button(
			el.Class("h-10 px-4 border border-zinc-200 rounded-lg text-sm flex items-center gap-2 hover:bg-zinc-50 transition-colors min-w-[220px]"),
			el.Span(el.Class("text-zinc-400"), el.Text("\U0001F4C5")), // calendar emoji
			el.DynText(func() string {
				d := time.Date(selYear(), time.Month(selMonth()), selDay(), 0, 0, 0, 0, time.UTC)
				return d.Format("January 2, 2006")
			}),
			el.OnClick(func(e dom.Event) {
				if !open() {
					reactive.Batch(func() {
						setViewYear(selYear())
						setViewMonth(selMonth())
						setOpen(true)
					})
				} else {
					setOpen(false)
				}
			}),
		),

		// Popover
		el.Show(
			func() bool { return open() },
			el.Div(
				// Transparent backdrop for click-outside-to-close
				el.Div(
					el.Class("fixed inset-0 z-40"),
					el.OnClick(func(e dom.Event) { setOpen(false) }),
				),

				// Calendar card
				el.Div(
					el.Class("absolute top-full left-0 mt-2 z-50 bg-white border border-zinc-200 rounded-xl shadow-lg p-4 w-[280px]"),

					// Month/year nav
					el.Div(
						el.Class("flex items-center justify-between mb-3"),
						el.Button(
							el.Class("w-8 h-8 rounded-lg flex items-center justify-center hover:bg-zinc-100 text-zinc-500 transition-colors"),
							el.Text("\u2039"), // ‹
							el.OnClick(func(e dom.Event) { navigateMonth(-1) }),
						),
						el.Span(
							el.Class("text-sm font-medium"),
							el.DynText(func() string {
								return fmt.Sprintf("%s %d", time.Month(viewMonth()), viewYear())
							}),
						),
						el.Button(
							el.Class("w-8 h-8 rounded-lg flex items-center justify-center hover:bg-zinc-100 text-zinc-500 transition-colors"),
							el.Text("\u203a"), // ›
							el.OnClick(func(e dom.Event) { navigateMonth(1) }),
						),
					),

					// Day-of-week headers
					weekdayHeaders(),

					// Day grid
					el.Dynamic(func() el.Node {
						return calendarGrid(
							viewYear(), viewMonth(),
							selYear(), selMonth(), selDay(),
							selectDay,
						)
					}),
				),
			),
		),
	)
}

func weekdayHeaders() el.Node {
	names := []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
	args := []any{el.Class("grid grid-cols-7 mb-1")}
	for _, n := range names {
		args = append(args, el.Div(
			el.Class("text-center text-xs text-zinc-400 font-medium py-1.5"),
			el.Text(n),
		))
	}
	return el.Div(args...)
}

func calendarGrid(vy, vm, sy, sm, sd int, onSelect func(int)) el.Node {
	first := time.Date(vy, time.Month(vm), 1, 0, 0, 0, 0, time.UTC)
	offset := int(first.Weekday())
	daysInMonth := time.Date(vy, time.Month(vm+1), 0, 0, 0, 0, 0, time.UTC).Day()
	today := time.Now()

	args := []any{el.Class("grid grid-cols-7 gap-0.5")}

	// Blank cells before the 1st
	for i := 0; i < offset; i++ {
		args = append(args, el.Div(el.Class("h-9")))
	}

	// Day cells
	for d := 1; d <= daysInMonth; d++ {
		day := d
		isSel := vy == sy && vm == sm && day == sd
		isToday := vy == today.Year() && vm == int(today.Month()) && day == today.Day()

		cls := "h-9 w-full flex items-center justify-center rounded-lg text-sm cursor-pointer transition-colors "
		switch {
		case isSel:
			cls += "bg-zinc-900 text-white font-medium"
		case isToday:
			cls += "ring-1 ring-zinc-300 hover:bg-zinc-100"
		default:
			cls += "hover:bg-zinc-100 text-zinc-700"
		}

		args = append(args, el.Div(
			el.Class(cls),
			el.Text(fmt.Sprintf("%d", day)),
			el.OnClick(func(e dom.Event) { onSelect(day) }),
		))
	}

	return el.Div(args...)
}

// ---------------------------------------------------------------------------
// Carousel
// ---------------------------------------------------------------------------

func CarouselDemo() el.Node {
	current, setCurrent := reactive.NewSignal(0)
	const total = 5

	type slide struct {
		gradient string
		label    string
	}
	slides := [total]slide{
		{"from-rose-500 to-pink-600", "Beautiful UI"},
		{"from-sky-500 to-blue-600", "Reactive Signals"},
		{"from-emerald-500 to-green-600", "Zero JavaScript"},
		{"from-amber-500 to-orange-600", "Type Safe"},
		{"from-violet-500 to-purple-600", "WebAssembly"},
	}

	// Build inner slide strip
	stripArgs := []any{
		el.Class("flex transition-transform duration-300 ease-out"),
		el.DynStyle("transform", func() string {
			return fmt.Sprintf("translateX(-%d%%)", current()*100)
		}),
	}
	for i := 0; i < total; i++ {
		s := slides[i]
		stripArgs = append(stripArgs, el.Div(
			el.Class("min-w-full p-1"),
			el.Div(
				el.Class("bg-gradient-to-br "+s.gradient+" rounded-xl h-52 flex flex-col items-center justify-center gap-3"),
				el.Span(el.Class("text-6xl font-bold text-white/25"), el.Text(fmt.Sprintf("%d", i+1))),
				el.Span(el.Class("text-lg font-semibold text-white"), el.Text(s.label)),
			),
		))
	}

	return el.Div(
		// Slides + arrows
		el.Div(
			el.Class("relative"),
			el.Div(
				el.Class("overflow-hidden rounded-xl"),
				el.Div(stripArgs...),
			),

			// Previous arrow
			el.Show(
				func() bool { return current() > 0 },
				el.Button(
					el.Class("absolute left-3 top-1/2 -translate-y-1/2 w-9 h-9 bg-white/90 border border-zinc-200 rounded-full flex items-center justify-center shadow-sm hover:bg-white transition-colors text-zinc-600"),
					el.Text("\u2039"),
					el.OnClick(func(e dom.Event) {
						if c := current(); c > 0 {
							setCurrent(c - 1)
						}
					}),
				),
			),

			// Next arrow
			el.Show(
				func() bool { return current() < total-1 },
				el.Button(
					el.Class("absolute right-3 top-1/2 -translate-y-1/2 w-9 h-9 bg-white/90 border border-zinc-200 rounded-full flex items-center justify-center shadow-sm hover:bg-white transition-colors text-zinc-600"),
					el.Text("\u203a"),
					el.OnClick(func(e dom.Event) {
						if c := current(); c < total-1 {
							setCurrent(c + 1)
						}
					}),
				),
			),
		),

		// Dot indicators
		el.Dynamic(func() el.Node {
			cur := current()
			args := []any{el.Class("flex justify-center gap-1.5 mt-4")}
			for i := 0; i < total; i++ {
				idx := i
				cls := "w-2 h-2 rounded-full transition-colors cursor-pointer "
				if cur == idx {
					cls += "bg-zinc-900"
				} else {
					cls += "bg-zinc-300 hover:bg-zinc-400"
				}
				args = append(args, el.Div(
					el.Class(cls),
					el.OnClick(func(e dom.Event) { setCurrent(idx) }),
				))
			}
			return el.Div(args...)
		}),
	)
}

// ---------------------------------------------------------------------------
// Button Group (tabs)
// ---------------------------------------------------------------------------

func ButtonGroupDemo() el.Node {
	selected, setSelected := reactive.NewSignal(0)

	type tab struct {
		label string
		title string
		desc  string
	}
	tabs := []tab{
		{"Account", "Account Settings", "Make changes to your account here. Click save when you\u2019re done."},
		{"Password", "Password", "Change your password here. After saving, you\u2019ll be logged out."},
		{"Settings", "Preferences", "Manage your notification preferences and display settings."},
	}

	return el.Div(
		// Tab bar
		el.Dynamic(func() el.Node {
			sel := selected()
			args := []any{el.Class("inline-flex rounded-lg border border-zinc-200 p-1 bg-zinc-100/80")}
			for i, t := range tabs {
				idx := i
				cls := "px-4 py-1.5 text-sm font-medium rounded-md transition-all duration-200 "
				if sel == idx {
					cls += "bg-white text-zinc-900 shadow-sm"
				} else {
					cls += "text-zinc-500 hover:text-zinc-700"
				}
				args = append(args, el.Button(
					el.Class(cls),
					el.Text(t.label),
					el.OnClick(func(e dom.Event) { setSelected(idx) }),
				))
			}
			return el.Div(args...)
		}),

		// Tab content
		el.Dynamic(func() el.Node {
			t := tabs[selected()]
			return el.Div(
				el.Class("mt-4 p-6 border border-zinc-200 rounded-xl"),
				el.H3(el.Class("font-semibold"), el.Text(t.title)),
				el.P(el.Class("text-sm text-zinc-500 mt-2 leading-relaxed"), el.Text(t.desc)),
				el.Div(
					el.Class("mt-4"),
					el.Button(
						el.Class("px-4 py-2 bg-zinc-900 text-white text-sm font-medium rounded-lg hover:bg-zinc-800 transition-colors"),
						el.Text("Save changes"),
					),
				),
			)
		}),
	)
}

// ---------------------------------------------------------------------------
// Resizable Panels
// ---------------------------------------------------------------------------

func ResizableDemo() el.Node {
	splitPct, setSplitPct := reactive.NewSignal(50.0)
	var containerEl dom.Element

	startResize := func(e dom.Event) {
		e.PreventDefault()
		startX := e.Value.Get("clientX").Float()
		startPct := splitPct()

		bodyStyle := js.Global().Get("document").Get("body").Get("style")
		bodyStyle.Set("cursor", "col-resize")
		bodyStyle.Set("userSelect", "none")

		var moveFn, upFn js.Func
		moveFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			dx := args[0].Get("clientX").Float() - startX
			w := containerEl.GetProperty("offsetWidth").Float()
			if w == 0 {
				return nil
			}
			pct := startPct + (dx/w)*100
			if pct < 15 {
				pct = 15
			}
			if pct > 85 {
				pct = 85
			}
			setSplitPct(pct)
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
		el.Class("flex border border-zinc-200 rounded-xl overflow-hidden h-72"),
		el.Ref(&containerEl),

		// Left panel
		el.Div(
			el.Class("overflow-auto p-5 bg-zinc-50/50"),
			el.DynStyle("width", func() string {
				return fmt.Sprintf("calc(%.1f%% - 4px)", splitPct())
			}),
			el.H4(el.Class("text-xs font-semibold mb-3 text-zinc-400 uppercase tracking-widest"), el.Text("Source")),
			el.Pre(
				el.Class("text-xs leading-relaxed text-zinc-600 font-mono whitespace-pre-wrap"),
				el.Text("func App() el.Node {\n  count, setCount :=\n    reactive.NewSignal(0)\n\n  return el.Div(\n    el.DynText(func() string {\n      return fmt.Sprintf(\n        \"Count: %d\", count())\n    }),\n    el.Button(\n      el.Text(\"+1\"),\n      el.OnClick(func(e dom.Event) {\n        setCount(count() + 1)\n      }),\n    ),\n  )\n}"),
			),
		),

		// Resize handle
		el.Div(
			el.Class("w-2 bg-zinc-100 hover:bg-zinc-200 transition-colors cursor-col-resize flex items-center justify-center flex-shrink-0 group"),
			el.Style("touch-action", "none"),
			el.On("pointerdown", startResize),
			el.Div(
				el.Class("flex flex-col gap-0.5"),
				el.Div(el.Class("w-0.5 h-1 rounded-full bg-zinc-400 group-hover:bg-zinc-500")),
				el.Div(el.Class("w-0.5 h-1 rounded-full bg-zinc-400 group-hover:bg-zinc-500")),
				el.Div(el.Class("w-0.5 h-1 rounded-full bg-zinc-400 group-hover:bg-zinc-500")),
			),
		),

		// Right panel
		el.Div(
			el.Class("flex-1 overflow-auto p-5"),
			el.H4(el.Class("text-xs font-semibold mb-3 text-zinc-400 uppercase tracking-widest"), el.Text("Preview")),
			el.P(
				el.Class("text-sm text-zinc-600 leading-relaxed"),
				el.Text("Drag the divider left and right. Each panel enforces a minimum width of 15% to prevent collapsing, matching the behavior of shadcn\u2019s resizable component."),
			),
			el.Div(
				el.Class("mt-4 text-xs text-zinc-400 tabular-nums"),
				el.DynText(func() string {
					return fmt.Sprintf("Split: %.0f%% / %.0f%%", splitPct(), 100-splitPct())
				}),
			),
		),
	)
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

func main() {
	el.Mount("#app", App)
	select {}
}
