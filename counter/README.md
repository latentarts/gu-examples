# Counter

A classic reactive counter demonstrating gu's core primitives — signals for state, memos for derived values, and conditional rendering with `el.Show`.

## Run it

```
make serve
```

Open http://localhost:8086. Click the + and - buttons to change the count. Toggle between light and dark themes.

## How it works

### gu concepts demonstrated

**Signals for state** (`state/state.go`). The counter value and theme toggle are separate signals, each independently reactive:

```go
count, setCount := reactive.NewSignal(0)
isDark, setIsDark := reactive.NewSignal(false)
```

**Memos for derived values** (`state/state.go`). The doubled count is computed reactively from the count signal:

```go
doubled := reactive.CreateMemo(func() int { return count() * 2 })
```

**Conditional rendering with `el.Show`** (`app/app.go`). A message appears only when the count exceeds 10:

```go
el.Show(func() bool { return s.Count() > 10 }, el.P(el.Text("Count is over 10!")))
```

**Theme switching** (`components/counter.go`). The theme toggle button switches between light and dark palettes:

```go
theme.SetTheme(theme.DefaultDark()) // or theme.DefaultLight()
```