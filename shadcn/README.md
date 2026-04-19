# shadcn/ui Component Migration

Five interactive components from [shadcn/ui](https://ui.shadcn.com) faithfully reproduced in pure Go using the gu framework. No React, no virtual DOM, no JavaScript framework — every component is implemented entirely with gu's reactive signals and direct DOM manipulation. Tailwind CSS handles styling, loaded from a CDN with zero build step.

This example demonstrates that sophisticated React component patterns (drag-to-dismiss, calendar date picking, carousel transitions, tabbed interfaces, resizable split panes) can be fully replicated in Go WASM without the original framework.

## Run it

```
make serve
```

Open http://localhost:8084.

## Components

### 1. Drawer

A bottom sheet panel that slides up from the screen edge, matching [shadcn's Drawer](https://ui.shadcn.com/docs/components/drawer) (powered by Vaul in React). Includes a drag handle that lets you swipe down to dismiss.

**Try it:** Click "Open Drawer", then drag the gray handle bar downward. Release past the threshold to dismiss, or release early to snap back. The +/- buttons inside the drawer update the goal value reactively.

#### gu concepts

**Always-in-DOM with style-controlled visibility** (`main.go:119-160`). Unlike `el.Show` which adds/removes elements (breaking CSS transitions), the overlay and panel are always rendered. Their visibility is controlled entirely through `el.DynStyle`:

```go
// Overlay fades in/out with CSS transition
el.DynStyle("opacity", func() string {
    if !open() { return "0" }
    return fmt.Sprintf("%.2f", 1.0 - offsetY()/500)
})
el.DynStyle("pointer-events", func() string {
    if open() { return "auto" }
    return "none"
})

// Panel slides with CSS transition, follows finger during drag
el.DynStyle("transform", func() string {
    if !open() { return "translateY(100%)" }
    if dragging() && offsetY() > 0 {
        return fmt.Sprintf("translateY(%.0fpx)", offsetY())
    }
    return "translateY(0)"
})
```

When `dragging()` is true, the `transition` style is set to `"none"` so the panel follows the pointer immediately. When dragging stops, the transition re-enables and the panel animates to its final position.

**Pointer event handling with `js.FuncOf`** (`main.go:83-114`). The drag gesture requires window-level `pointermove`/`pointerup` listeners that are added on `pointerdown` and removed on release. This pattern properly cleans up JS function references:

```go
var moveFn, upFn js.Func
moveFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
    dy := args[0].Get("clientY").Float() - startY
    if dy < 0 { dy = 0 }
    setOffsetY(dy)
    return nil
})
upFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
    js.Global().Call("removeEventListener", "pointermove", moveFn)
    js.Global().Call("removeEventListener", "pointerup", upFn)
    moveFn.Release()
    upFn.Release()
    // decide: dismiss or snap back
    return nil
})
js.Global().Call("addEventListener", "pointermove", moveFn)
js.Global().Call("addEventListener", "pointerup", upFn)
```

**`reactive.Batch` for atomic updates** (`main.go:72-77`). When dismissing, three signals change at once (`dragging`, `open`, `offsetY`). Without batching, each set would trigger effects independently. `Batch` groups them so effects run once after all three update:

```go
reactive.Batch(func() {
    setDragging(false)
    setOpen(false)
    setOffsetY(0)
})
```

---

### 2. Date Picker

A calendar popup matching [shadcn's DatePicker](https://ui.shadcn.com/docs/components/date-picker). Click the trigger to open a dropdown calendar with month navigation. Select a date to close the popover and update the display.

**Try it:** Click the date button, navigate months with the arrows, click a day to select it.

#### gu concepts

**`el.Show` for popover toggle** (`main.go:270-322`). Unlike the drawer (which needs CSS transitions), the calendar popover can pop in/out instantly. `el.Show` adds the element to the DOM when `open()` is true and removes it when false:

```go
el.Show(
    func() bool { return open() },
    el.Div(
        // invisible backdrop
        el.Div(
            el.Class("fixed inset-0 z-40"),
            el.OnClick(func(e dom.Event) { setOpen(false) }),
        ),
        // calendar card
        el.Div(el.Class("absolute top-full ..."), ...),
    ),
)
```

The transparent fixed backdrop catches clicks outside the calendar to close it.

**`el.Dynamic` for the calendar grid** (`main.go:316-321`). The day grid rebuilds whenever the viewed month or selected date changes. The `Dynamic` callback reads five signals, so it re-runs when any of them change:

```go
el.Dynamic(func() el.Node {
    return calendarGrid(
        viewYear(), viewMonth(),
        selYear(), selMonth(), selDay(),
        selectDay,
    )
})
```

**Go's `time` package for calendar math** (`main.go:339-345`). Computing the first weekday and days-in-month uses standard library time arithmetic:

```go
first := time.Date(vy, time.Month(vm), 1, 0, 0, 0, 0, time.UTC)
offset := int(first.Weekday())
daysInMonth := time.Date(vy, time.Month(vm+1), 0, 0, 0, 0, 0, time.UTC).Day()
```

Day 0 of the next month gives the last day of the current month — a standard Go idiom.

**Building dynamic children with `[]any` slices** (`main.go:350-374`). Since `el.Div` takes `...any`, and we can't spread a slice after another argument, we prepend the class modifier into the slice:

```go
args := []any{el.Class("grid grid-cols-7 gap-0.5")}
for d := 1; d <= daysInMonth; d++ {
    day := d
    args = append(args, el.Div(
        el.Class(cls),
        el.Text(fmt.Sprintf("%d", day)),
        el.OnClick(func(e dom.Event) { onSelect(day) }),
    ))
}
return el.Div(args...)
```

---

### 3. Carousel

A horizontal slideshow matching [shadcn's Carousel](https://ui.shadcn.com/docs/components/carousel) (powered by Embla in React). Navigate with arrow buttons or click the dot indicators. Slides animate with CSS transitions.

**Try it:** Click the arrows or dots to navigate between slides.

#### gu concepts

**CSS `transform` driven by a signal** (`main.go:399-402`). The slide strip position is a pure function of the `current` signal:

```go
el.DynStyle("transform", func() string {
    return fmt.Sprintf("translateX(-%d%%)", current()*100)
})
```

Combined with `transition-transform duration-300 ease-out` on the element's class, this produces smooth animated slides with zero imperative animation code.

**`el.Show` for conditional arrow buttons** (`main.go:418-444`). The previous arrow only appears when `current() > 0`, and the next arrow hides at the last slide:

```go
el.Show(
    func() bool { return current() > 0 },
    el.Button(el.Class("absolute left-3 ..."), ...),
)
```

**Single `el.Dynamic` for all dot indicators** (`main.go:448-467`). Rather than wrapping each dot in its own Dynamic, one Dynamic rebuilds all dots when `current()` changes. The active dot gets `bg-zinc-900`, others get `bg-zinc-300`:

```go
el.Dynamic(func() el.Node {
    cur := current()
    args := []any{el.Class("flex justify-center gap-1.5 mt-4")}
    for i := 0; i < total; i++ {
        idx := i
        cls := "w-2 h-2 rounded-full transition-colors cursor-pointer "
        if cur == idx { cls += "bg-zinc-900" } else { cls += "bg-zinc-300" }
        args = append(args, el.Div(
            el.Class(cls),
            el.OnClick(func(e dom.Event) { setCurrent(idx) }),
        ))
    }
    return el.Div(args...)
})
```

---

### 4. Button Group

A tabbed interface matching [shadcn's Tabs](https://ui.shadcn.com/docs/components/tabs). One option is active at a time. The active tab has a white background with shadow, inactive tabs are transparent. Content panel switches with the active tab.

**Try it:** Click the tab buttons to switch between Account, Password, and Settings.

#### gu concepts

**Two `el.Dynamic` blocks sharing one signal** (`main.go:486-523`). The tab bar and content panel each read `selected()`. When the signal changes, both Dynamic callbacks re-run independently:

```go
// Tab bar
el.Dynamic(func() el.Node {
    sel := selected()
    // ... build tab buttons with active/inactive styles based on sel
})
// Content panel
el.Dynamic(func() el.Node {
    t := tabs[selected()]
    // ... build content card for selected tab
})
```

This is gu's equivalent of React's `useState` — the signal is the single source of truth, and any reactive context that reads it automatically re-runs when it changes.

**Loop variable capture** (`main.go:497`). Each button's click handler needs to capture the correct index. The `idx := i` pattern creates a new variable scoped to each iteration:

```go
for i, t := range tabs {
    idx := i
    args = append(args, el.Button(
        el.OnClick(func(e dom.Event) { setSelected(idx) }),
    ))
}
```

---

### 5. Resizable Panels

A split pane with a draggable divider matching [shadcn's Resizable](https://ui.shadcn.com/docs/components/resizable). The left panel shows source code, the right shows a preview. Panels enforce a 15-85% size range.

**Try it:** Drag the thin bar between the panels left and right. Watch the split percentage update in real time.

#### gu concepts

**`el.Ref` for DOM measurements** (`main.go:541`). The resize calculation needs the container's pixel width. `el.Ref` captures the underlying `dom.Element` so we can query `offsetWidth` during drag:

```go
var containerEl dom.Element
el.Div(
    el.Ref(&containerEl),
    // ...
)
// Later, during pointermove:
w := containerEl.GetProperty("offsetWidth").Float()
pct := startPct + (dx/w)*100
```

**Percentage-based layout with `el.DynStyle`** (`main.go:551-553`). The left panel's width is a reactive style driven by the `splitPct` signal. The right panel uses `flex-1` to fill the remainder:

```go
el.DynStyle("width", func() string {
    return fmt.Sprintf("calc(%.1f%% - 4px)", splitPct())
})
```

The `- 4px` accounts for the 8px divider handle (half on each side).

**Body-level cursor override during drag** (`main.go:533-536`). Setting `cursor: col-resize` and `user-select: none` on `document.body` during drag ensures the cursor doesn't flicker when the pointer moves fast and temporarily leaves the handle:

```go
bodyStyle := js.Global().Get("document").Get("body").Get("style")
bodyStyle.Set("cursor", "col-resize")
bodyStyle.Set("userSelect", "none")
```

These are cleaned up in the `pointerup` handler.

## Key patterns vs React

| Pattern | React (shadcn/ui) | gu |
|---|---|---|
| State | `useState` hook | `reactive.NewSignal` |
| Derived state | `useMemo` hook | `reactive.CreateMemo` |
| Side effects | `useEffect` hook | `reactive.CreateEffect` |
| Conditional render | `{condition && <Component>}` | `el.Show(func() bool, node)` |
| Dynamic content | JSX expressions | `el.Dynamic(func() el.Node)` |
| Event handlers | `onClick={handler}` | `el.OnClick(handler)` |
| Refs | `useRef` hook | `el.Ref(&element)` |
| CSS classes | `className={cn(...)}` | `el.Class(...)` / `el.DynClass(...)` |
| Inline styles | `style={{prop: value}}` | `el.Style(p, v)` / `el.DynStyle(p, fn)` |
| Batched updates | Automatic in React 18+ | `reactive.Batch(func())` |
| Lifecycle | `useEffect` with `[]` deps | `el.OnMount(func(dom.Element))` |
| Cleanup | `useEffect` return function | `reactive.OnCleanup(func())` |

The core difference: React re-renders entire component trees through a virtual DOM diff. gu runs component functions **once** and uses fine-grained signals to update only the specific DOM nodes that changed.

## Files

| File | Purpose |
|---|---|
| `main.go` | All Go/gu code — five components, ~470 lines, zero JS interop libraries |
| `index.html` | Tailwind CDN + WASM bootstrap (no JS helpers needed) |
| `Makefile` | Build WASM + copy assets to `dist/`, serve locally |
