# Tailwind CSS Component Showcase

A gallery of six UI component cards built entirely in Go using gu's `el` package with Tailwind CSS utility classes. Includes a dark/light theme toggle. No JavaScript interop needed — this example is pure gu.

## Run it

```
make serve
```

Open http://localhost:8081. Click "Dark Mode" / "Light Mode" to toggle themes.

## How it works

### Architecture

```
index.html               main.go
  |                        |
  | loads Tailwind v4      | builds all UI with el.*
  | via browser CDN        | uses el.Class("...") for
  | defines @theme colors  |   Tailwind utility classes
  |                        | DynClass toggles dark mode
```

This is the simplest example — no JS interop at all. Tailwind is loaded as a CDN script in `index.html` and gu's `el.Class()` applies utility classes to every element.

### gu concepts demonstrated

**Component functions** (`main.go:57-182`). Each card is a plain Go function returning `el.Node`. This is gu's component model — no interfaces to implement, no lifecycle boilerplate, just functions:

```go
func HeroCard() el.Node {
    return el.Div(
        el.Class("rounded-2xl bg-gradient-to-br from-brand to-brand-dark p-8 text-white"),
        el.H2(el.Class("text-3xl font-bold mb-3"), el.Text("Build reactive UIs in Go")),
        // ...
    )
}
```

Components compose naturally — the grid in `App()` just calls each card function:

```go
el.Div(
    el.Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"),
    HeroCard(),
    ProfileCard(),
    PricingCard(),
    StatsCard(),
    TestimonialCard(),
    CTACard(),
)
```

**Reactive dark mode with `el.DynClass`** (`main.go:13-18`). A single boolean signal controls the entire theme. `DynClass` swaps the root element's class string reactively, and Tailwind's `dark:` variants handle the rest:

```go
isDark, setIsDark := reactive.NewSignal(false)

el.Div(
    el.DynClass(func() string {
        if isDark() {
            return "dark min-h-screen bg-gray-950 text-gray-100 transition-colors duration-300"
        }
        return "min-h-screen bg-gray-50 text-gray-900 transition-colors duration-300"
    }),
    // ...
)
```

Every child element using `dark:` prefixed classes (like `dark:bg-gray-900`, `dark:text-gray-400`) reacts automatically because Tailwind's dark mode is CSS-based, triggered by the `dark` class on an ancestor.

**Reactive button text with `el.DynText`** (`main.go:31-36`). The toggle button reads the `isDark` signal to show the right label:

```go
el.DynText(func() string {
    if isDark() { return "Light Mode" }
    return "Dark Mode"
})
```

**Helper functions for repeated patterns** (`main.go:88-93, 116-122`). Small helpers like `statItem` and `featureItem` reduce duplication while staying simple — they're just functions that return `el.Node`:

```go
func statItem(label, value string) el.Node {
    return el.Div(
        el.Span(el.Class("block font-bold"), el.Text(value)),
        el.Span(el.Class("text-gray-500"), el.Text(label)),
    )
}
```

**`el.Class` with Tailwind** (`main.go` throughout). Every element uses `el.Class("...")` with standard Tailwind utility classes. gu doesn't interfere with class names — `el.Class` sets the `class` attribute directly, so any CSS framework works:

```go
el.Div(
    el.Class("rounded-2xl bg-white dark:bg-gray-900 p-6 shadow-lg border border-gray-100"),
    // ...
)
```

### Custom Tailwind theme (`index.html`)

The `@theme` block in `index.html` defines custom colors that Tailwind utilities reference (e.g., `text-brand`, `bg-brand-dark`):

```html
<style type="text/tailwindcss">
    @theme {
        --color-brand: #6366f1;
        --color-brand-light: #818cf8;
        --color-brand-dark: #4338ca;
        --color-accent: #f59e0b;
    }
</style>
```

## Files

| File | Purpose |
|---|---|
| `main.go` | All Go/gu code — signals, component functions, UI tree |
| `index.html` | Tailwind CDN, custom theme, WASM bootstrap |
| `Makefile` | Build WASM + copy assets to `dist/`, serve locally |
