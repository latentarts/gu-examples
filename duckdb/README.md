# DuckDB SQL Explorer

Interactive SQL query editor running entirely in the browser. Type SQL, execute it against an in-memory DuckDB database pre-loaded with 1,000 rows of sample data, and see results in a styled table.

## Run it

```
make serve
```

Open http://localhost:8080. DuckDB loads in a few seconds, then type SQL and click "Run Query".

Try these queries:

```sql
SELECT category, COUNT(*) AS cnt, ROUND(AVG(price), 2) AS avg_price
FROM sales GROUP BY category ORDER BY cnt DESC
```

```sql
SELECT product, SUM(quantity) AS total_qty FROM sales
GROUP BY product ORDER BY total_qty DESC LIMIT 10
```

## How it works

### Architecture

```
index.html              main.go
  |                       |
  | loads DuckDB WASM     | builds UI with el.*
  | via ES module import  | reads signals for state
  | exposes window.App    | calls window.App.query()
  |   .query(sql)         |   via jsutil.Await()
  |   .ready              | updates signals with results
  |   .onReady            |
```

The heavy lifting (DuckDB initialization, Arrow-to-array conversion) lives in JavaScript inside `index.html`. Go only sees a simple `App.query(sql) -> Promise<string>` interface.

### gu concepts demonstrated

**Signals for all UI state** (`main.go:20-28`). Every piece of state is a signal — the SQL text, loading flag, error message, and whether the database is ready:

```go
sql, setSQL := reactive.NewSignal("SELECT ...")
loading, setLoading := reactive.NewSignal(false)
errMsg, setErrMsg := reactive.NewSignal("")
dbReady, setDBReady := reactive.NewSignal(false)
```

**Version-counter pattern** (`main.go:26-28`). Signals require `comparable` types, so slices can't be signals directly. Instead, store the slice in a plain variable and bump an integer signal to notify the reactive system:

```go
resultVer, setResultVer := reactive.NewSignal(0)
var columns []string
var rows [][]string

// After getting results:
columns = result.Columns
rows = result.Rows
setResultVer(resultVer() + 1) // triggers re-render
```

Any `DynText` or `Dynamic` that reads `resultVer()` will re-run when results change.

**Async JS interop with goroutines** (`main.go:60-78`). Calling a JavaScript promise from Go uses a goroutine + `jsutil.Await` so the main thread isn't blocked:

```go
go func() {
    promise := js.Global().Get("App").Call("query", sql())
    val, err := jsutil.Await(promise)
    // ... parse JSON, update signals
}()
```

**Conditional rendering with `el.Show`** (`main.go:128-131`). Error messages only appear when the error signal is non-empty:

```go
el.Show(
    func() bool { return errMsg() != "" },
    el.Div(el.Class("error"), el.DynText(errMsg)),
)
```

**Dynamic DOM with `el.Dynamic`** (`main.go:140-164`). The results table rebuilds entirely when `resultVer()` changes. Inside the callback, plain Go loops build the `[]any` slices of `el.Th`/`el.Td`/`el.Tr` nodes:

```go
el.Dynamic(func() el.Node {
    _ = resultVer()
    headerCells := make([]any, len(columns))
    for i, col := range columns {
        headerCells[i] = el.Th(el.Text(col))
    }
    // ... same for rows
    return el.Table(
        el.Thead(el.Tr(headerCells...)),
        el.Tbody(tableRows...),
    )
})
```

**Reactive button text and disabled state** (`main.go:97-114`). `el.DynText` and `el.DynAttr` read multiple signals to derive the button label and `disabled` attribute reactively — no manual DOM manipulation needed:

```go
el.DynText(func() string {
    if loading() { return "Running..." }
    if !dbReady() { return "Loading DuckDB..." }
    return "Run Query"
})
```

### JS callback bridge (`main.go:31-52`)

On startup, a goroutine registers Go callbacks (`onReady`, `onError`) on `window.App` so JavaScript can notify Go when DuckDB finishes initializing. This uses `js.FuncOf` with a channel to bridge async JS events into Go's goroutine model:

```go
ch := make(chan struct{}, 1)
cb := js.FuncOf(func(this js.Value, args []js.Value) any {
    ch <- struct{}{}
    return nil
})
app.Set("onReady", cb)
<-ch          // blocks goroutine until JS calls onReady
cb.Release()  // free the Go->JS reference
```

## Files

| File | Purpose |
|---|---|
| `main.go` | All Go/gu code — signals, UI tree, query execution |
| `index.html` | DuckDB WASM bootstrap, `App.query()` helper, CSS |
| `Makefile` | Build WASM + copy assets to `dist/`, serve locally |
