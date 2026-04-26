//go:build js && wasm

package components

import (
	"fmt"
	"syscall/js"
	"time"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/shadcn/state"
)

func Showcase(styles el.Node) el.Node {
	jsutil.LogInfo("shadcn/ui showcase mounted")
	return el.Div(
		styles,
		el.Class("min-h-screen bg-white text-zinc-900 antialiased"),
		el.Div(
			el.Class("border-b border-zinc-200"),
			el.Div(
				el.Class("max-w-4xl mx-auto px-6 py-10"),
				el.H1(el.Class("text-3xl font-bold tracking-tight"), el.Text("shadcn/ui → gu")),
				el.P(
					el.Class("text-zinc-500 mt-2 text-sm leading-relaxed max-w-2xl"),
					el.Text("Five React components from shadcn/ui faithfully reproduced in pure Go. No JavaScript framework, no virtual DOM — just gu’s reactive signals and direct DOM updates."),
				),
			),
		),
		el.Div(
			el.Class("max-w-4xl mx-auto px-6 py-12 space-y-20"),
			demoSection("Drawer", "A panel that slides up from the bottom. Drag the handle down to dismiss.", DrawerDemo()),
			demoSection("Date Picker", "A calendar popup for selecting dates with month navigation.", DatePickerDemo()),
			demoSection("Carousel", "A slideshow with prev/next arrows and dot indicators.", CarouselDemo()),
			demoSection("Button Group", "Toggle between options — one active at a time.", ButtonGroupDemo()),
			demoSection("Resizable Panels", "Drag the divider to resize. Panels enforce 15–85% bounds.", ResizableDemo()),
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

func DrawerDemo() el.Node {
	s := state.NewDrawerState()
	body := js.Global().Get("document").Get("body")

	closeDrawer := func() {
		reactive.Batch(func() {
			s.SetDragging(false)
			s.SetOpen(false)
			s.SetOffsetY(0)
		})
		body.Get("classList").Call("remove", "drawer-open")
		jsutil.LogDebug("drawer closed")
	}

	openDrawer := func() {
		s.SetOpen(true)
		body.Get("classList").Call("add", "drawer-open")
		jsutil.LogDebug("drawer opened")
	}

	handlePointerDown := func(e dom.Event) {
		e.PreventDefault()
		startY := e.Value.Get("clientY").Float()
		s.SetDragging(true)

		var moveFn, upFn js.Func
		moveFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			dy := args[0].Get("clientY").Float() - startY
			if dy < 0 {
				dy = 0
			}
			s.SetOffsetY(dy)
			return nil
		})
		upFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			js.Global().Call("removeEventListener", "pointermove", moveFn)
			js.Global().Call("removeEventListener", "pointerup", upFn)
			moveFn.Release()
			upFn.Release()
			if s.OffsetY() > 120 {
				closeDrawer()
			} else {
				reactive.Batch(func() {
					s.SetDragging(false)
					s.SetOffsetY(0)
				})
			}
			return nil
		})
		js.Global().Call("addEventListener", "pointermove", moveFn)
		js.Global().Call("addEventListener", "pointerup", upFn)
	}

	return el.Div(
		el.Button(
			el.Class("px-4 py-2.5 bg-zinc-900 text-white rounded-lg text-sm font-medium hover:bg-zinc-800 transition-colors"),
			el.Text("Open Drawer"),
			el.OnClick(func(e dom.Event) { openDrawer() }),
		),
		el.Div(
			el.Class("fixed inset-0 z-40 bg-black/80"),
			el.DynStyle("opacity", func() string {
				if !s.Open() {
					return "0"
				}
				o := 1.0 - s.OffsetY()/500
				if o < 0 {
					o = 0
				}
				return fmt.Sprintf("%.2f", o)
			}),
			el.DynStyle("pointer-events", func() string {
				if s.Open() {
					return "auto"
				}
				return "none"
			}),
			el.DynStyle("transition", func() string {
				if s.Dragging() {
					return "none"
				}
				return "opacity 0.3s"
			}),
			el.OnClick(func(e dom.Event) { closeDrawer() }),
		),
		el.Div(
			el.Class("fixed inset-x-0 bottom-0 z-50 bg-white rounded-t-2xl"),
			el.DynStyle("transform", func() string {
				if !s.Open() {
					return "translateY(100%)"
				}
				if s.Dragging() && s.OffsetY() > 0 {
					return fmt.Sprintf("translateY(%.0fpx)", s.OffsetY())
				}
				return "translateY(0)"
			}),
			el.DynStyle("transition", func() string {
				if s.Dragging() {
					return "none"
				}
				return "transform 0.3s cubic-bezier(0.32, 0.72, 0, 1)"
			}),
			el.Div(
				el.Class("pt-4 pb-2 cursor-grab active:cursor-grabbing"),
				el.Style("touch-action", "none"),
				el.On("pointerdown", handlePointerDown),
				el.Div(el.Class("w-12 h-1.5 rounded-full bg-zinc-300 mx-auto")),
			),
			el.Div(
				el.Class("px-6 pb-8 pt-2"),
				el.H3(el.Class("text-lg font-semibold"), el.Text("Move Goal")),
				el.P(el.Class("text-sm text-zinc-500 mt-1 mb-8"), el.Text("Set your daily activity goal.")),
				el.Div(
					el.Class("flex items-center justify-center gap-8 mb-8"),
					el.Button(
						el.Class("w-12 h-12 rounded-full border border-zinc-200 text-xl flex items-center justify-center hover:bg-zinc-50 transition-colors"),
						el.Text("−"),
						el.OnClick(func(e dom.Event) {
							if g := s.Goal(); g > 100 {
								s.SetGoal(g - 10)
							}
						}),
					),
					el.Div(
						el.Class("text-center"),
						el.Div(
							el.Class("text-5xl font-bold tabular-nums"),
							el.DynText(func() string { return fmt.Sprintf("%d", s.Goal()) }),
						),
						el.Div(el.Class("text-xs text-zinc-500 mt-1"), el.Text("calories / day")),
					),
					el.Button(
						el.Class("w-12 h-12 rounded-full border border-zinc-200 text-xl flex items-center justify-center hover:bg-zinc-50 transition-colors"),
						el.Text("+"),
						el.OnClick(func(e dom.Event) { s.SetGoal(s.Goal() + 10) }),
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

func DatePickerDemo() el.Node {
	s := state.NewDatePickerState(time.Now())

	navigateMonth := func(delta int) {
		d := time.Date(s.ViewYear(), time.Month(s.ViewMonth()), 1, 0, 0, 0, 0, time.UTC).AddDate(0, delta, 0)
		reactive.Batch(func() {
			s.SetViewYear(d.Year())
			s.SetViewMonth(int(d.Month()))
		})
	}

	selectDay := func(day int) {
		reactive.Batch(func() {
			s.SetSelYear(s.ViewYear())
			s.SetSelMonth(s.ViewMonth())
			s.SetSelDay(day)
			s.SetOpen(false)
		})
		jsutil.LogDebug("date selected: %d-%02d-%02d", s.ViewYear(), s.ViewMonth(), day)
	}

	return el.Div(
		el.Class("relative inline-block"),
		el.Button(
			el.Class("h-10 px-4 border border-zinc-200 rounded-lg text-sm flex items-center gap-2 hover:bg-zinc-50 transition-colors min-w-[220px]"),
			el.Span(el.Class("text-zinc-400"), el.Text("📅")),
			el.DynText(func() string {
				d := time.Date(s.SelYear(), time.Month(s.SelMonth()), s.SelDay(), 0, 0, 0, 0, time.UTC)
				return d.Format("January 2, 2006")
			}),
			el.OnClick(func(e dom.Event) {
				if !s.Open() {
					reactive.Batch(func() {
						s.SetViewYear(s.SelYear())
						s.SetViewMonth(s.SelMonth())
						s.SetOpen(true)
					})
				} else {
					s.SetOpen(false)
				}
			}),
		),
		el.Show(
			func() bool { return s.Open() },
			el.Div(
				el.Div(el.Class("fixed inset-0 z-40"), el.OnClick(func(e dom.Event) { s.SetOpen(false) })),
				el.Div(
					el.Class("absolute top-full left-0 mt-2 z-50 bg-white border border-zinc-200 rounded-xl shadow-lg p-4 w-[280px]"),
					el.Div(
						el.Class("flex items-center justify-between mb-3"),
						el.Button(el.Class("w-8 h-8 rounded-lg flex items-center justify-center hover:bg-zinc-100 text-zinc-500 transition-colors"), el.Text("‹"), el.OnClick(func(e dom.Event) { navigateMonth(-1) })),
						el.Span(el.Class("text-sm font-medium"), el.DynText(func() string {
							return fmt.Sprintf("%s %d", time.Month(s.ViewMonth()), s.ViewYear())
						})),
						el.Button(el.Class("w-8 h-8 rounded-lg flex items-center justify-center hover:bg-zinc-100 text-zinc-500 transition-colors"), el.Text("›"), el.OnClick(func(e dom.Event) { navigateMonth(1) })),
					),
					weekdayHeaders(),
					el.Dynamic(func() el.Node {
						return calendarGrid(s.ViewYear(), s.ViewMonth(), s.SelYear(), s.SelMonth(), s.SelDay(), selectDay)
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
		args = append(args, el.Div(el.Class("text-center text-xs text-zinc-400 font-medium py-1.5"), el.Text(n)))
	}
	return el.Div(args...)
}

func calendarGrid(vy, vm, sy, sm, sd int, onSelect func(int)) el.Node {
	first := time.Date(vy, time.Month(vm), 1, 0, 0, 0, 0, time.UTC)
	offset := int(first.Weekday())
	daysInMonth := time.Date(vy, time.Month(vm+1), 0, 0, 0, 0, 0, time.UTC).Day()
	today := time.Now()
	args := []any{el.Class("grid grid-cols-7 gap-0.5")}
	for i := 0; i < offset; i++ {
		args = append(args, el.Div(el.Class("h-9")))
	}
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
		args = append(args, el.Div(el.Class(cls), el.Text(fmt.Sprintf("%d", day)), el.OnClick(func(e dom.Event) { onSelect(day) })))
	}
	return el.Div(args...)
}

func CarouselDemo() el.Node {
	s := state.NewCarouselState()
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
	stripArgs := []any{
		el.Class("flex transition-transform duration-300 ease-out"),
		el.DynStyle("transform", func() string {
			return fmt.Sprintf("translateX(-%d%%)", s.Current()*100)
		}),
	}
	for i := 0; i < total; i++ {
		slide := slides[i]
		stripArgs = append(stripArgs, el.Div(
			el.Class("min-w-full p-1"),
			el.Div(
				el.Class("bg-gradient-to-br "+slide.gradient+" rounded-xl h-52 flex flex-col items-center justify-center gap-3"),
				el.Span(el.Class("text-6xl font-bold text-white/25"), el.Text(fmt.Sprintf("%d", i+1))),
				el.Span(el.Class("text-lg font-semibold text-white"), el.Text(slide.label)),
			),
		))
	}
	return el.Div(
		el.Div(
			el.Class("relative"),
			el.Div(el.Class("overflow-hidden rounded-xl"), el.Div(stripArgs...)),
			el.Show(func() bool { return s.Current() > 0 },
				el.Button(el.Class("absolute left-3 top-1/2 -translate-y-1/2 w-9 h-9 bg-white/90 border border-zinc-200 rounded-full flex items-center justify-center shadow-sm hover:bg-white transition-colors text-zinc-600"), el.Text("‹"), el.OnClick(func(e dom.Event) {
					if c := s.Current(); c > 0 {
						s.SetCurrent(c - 1)
					}
				}))),
			el.Show(func() bool { return s.Current() < total-1 },
				el.Button(el.Class("absolute right-3 top-1/2 -translate-y-1/2 w-9 h-9 bg-white/90 border border-zinc-200 rounded-full flex items-center justify-center shadow-sm hover:bg-white transition-colors text-zinc-600"), el.Text("›"), el.OnClick(func(e dom.Event) {
					if c := s.Current(); c < total-1 {
						s.SetCurrent(c + 1)
					}
				}))),
		),
		el.Dynamic(func() el.Node {
			cur := s.Current()
			args := []any{el.Class("flex justify-center gap-1.5 mt-4")}
			for i := 0; i < total; i++ {
				idx := i
				cls := "w-2 h-2 rounded-full transition-colors cursor-pointer "
				if cur == idx {
					cls += "bg-zinc-900"
				} else {
					cls += "bg-zinc-300 hover:bg-zinc-400"
				}
				args = append(args, el.Div(el.Class(cls), el.OnClick(func(e dom.Event) { s.SetCurrent(idx) })))
			}
			return el.Div(args...)
		}),
	)
}

func ButtonGroupDemo() el.Node {
	s := state.NewButtonGroupState()
	type tab struct {
		label string
		title string
		desc  string
	}
	tabs := []tab{
		{"Account", "Account Settings", "Make changes to your account here. Click save when you’re done."},
		{"Password", "Password", "Change your password here. After saving, you’ll be logged out."},
		{"Settings", "Preferences", "Manage your notification preferences and display settings."},
	}
	return el.Div(
		el.Dynamic(func() el.Node {
			sel := s.Selected()
			args := []any{el.Class("inline-flex rounded-lg border border-zinc-200 p-1 bg-zinc-100/80")}
			for i, tab := range tabs {
				idx := i
				cls := "px-4 py-1.5 text-sm font-medium rounded-md transition-all duration-200 "
				if sel == idx {
					cls += "bg-white text-zinc-900 shadow-sm"
				} else {
					cls += "text-zinc-500 hover:text-zinc-700"
				}
				args = append(args, el.Button(el.Class(cls), el.Text(tab.label), el.OnClick(func(e dom.Event) { s.SetSelected(idx) })))
			}
			return el.Div(args...)
		}),
		el.Dynamic(func() el.Node {
			tab := tabs[s.Selected()]
			return el.Div(
				el.Class("mt-4 p-6 border border-zinc-200 rounded-xl"),
				el.H3(el.Class("font-semibold"), el.Text(tab.title)),
				el.P(el.Class("text-sm text-zinc-500 mt-2 leading-relaxed"), el.Text(tab.desc)),
				el.Div(el.Class("mt-4"),
					el.Button(el.Class("px-4 py-2 bg-zinc-900 text-white text-sm font-medium rounded-lg hover:bg-zinc-800 transition-colors"), el.Text("Save changes")),
				),
			)
		}),
	)
}

func ResizableDemo() el.Node {
	s := state.NewResizableState()
	startResize := func(e dom.Event) {
		e.PreventDefault()
		startX := e.Value.Get("clientX").Float()
		startPct := s.SplitPct()

		bodyStyle := js.Global().Get("document").Get("body").Get("style")
		bodyStyle.Set("cursor", "col-resize")
		bodyStyle.Set("userSelect", "none")

		var moveFn, upFn js.Func
		moveFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
			dx := args[0].Get("clientX").Float() - startX
			w := s.ContainerEl.GetProperty("offsetWidth").Float()
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
			s.SetSplitPct(pct)
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
		el.Ref(&s.ContainerEl),
		el.Div(
			el.Class("overflow-auto p-5 bg-zinc-50/50"),
			el.DynStyle("width", func() string { return fmt.Sprintf("calc(%.1f%% - 4px)", s.SplitPct()) }),
			el.H4(el.Class("text-xs font-semibold mb-3 text-zinc-400 uppercase tracking-widest"), el.Text("Source")),
			el.Pre(el.Class("text-xs leading-relaxed text-zinc-600 font-mono whitespace-pre-wrap"), el.Text("func App() el.Node {\n  count, setCount :=\n    reactive.NewSignal(0)\n\n  return el.Div(\n    el.DynText(func() string {\n      return fmt.Sprintf(\n        \"Count: %d\", count())\n    }),\n    el.Button(\n      el.Text(\"+1\"),\n      el.OnClick(func(e dom.Event) {\n        setCount(count() + 1)\n      }),\n    ),\n  )\n}")),
		),
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
		el.Div(
			el.Class("flex-1 overflow-auto p-5"),
			el.H4(el.Class("text-xs font-semibold mb-3 text-zinc-400 uppercase tracking-widest"), el.Text("Preview")),
			el.P(el.Class("text-sm text-zinc-600 leading-relaxed"), el.Text("Drag the divider left and right. Each panel enforces a minimum width of 15% to prevent collapsing, matching the behavior of shadcn’s resizable component.")),
			el.Div(
				el.Class("mt-4 text-xs text-zinc-400 tabular-nums"),
				el.DynText(func() string { return fmt.Sprintf("Split: %.0f%% / %.0f%%", s.SplitPct(), 100-s.SplitPct()) }),
			),
		),
	)
}
