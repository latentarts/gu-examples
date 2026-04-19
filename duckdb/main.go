//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
)

type queryResultData struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
	Error   string
}

type appPhase int

const (
	PhaseInit appPhase = iota
	PhaseReady
	PhaseRunning
)

func App() el.Node {
	jsutil.LogInfo("App initializing...")
	
	sql, setSQL := reactive.NewSignal("SELECT category, COUNT(*) AS cnt, ROUND(AVG(price), 2) AS avg_price\nFROM sales\nGROUP BY category\nORDER BY cnt DESC")
	phase, setPhase := reactive.NewSignal(PhaseInit)
	results, setResults := reactive.NewSignal[*queryResultData](nil)

	// Monitor DuckDB Readiness
	initDB := func() {
		go func() {
			for {
				app := js.Global().Get("App")
				if !app.IsUndefined() && app.Get("ready").Bool() {
					jsutil.LogInfo("DuckDB ready detected in Go")
					dom.RequestAnimationFrame(func() {
						setPhase(PhaseReady)
					})
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}

	runQuery := func() {
		if phase() != PhaseReady {
			return
		}
		
		setPhase(PhaseRunning)
		jsutil.LogDebug("Starting query execution...")
		
		go func() {
			currentSQL := sql()
			promise := js.Global().Get("App").Call("query", currentSQL)
			val, err := jsutil.Await(promise)
			
			dom.RequestAnimationFrame(func() {
				if err != nil {
					jsutil.LogError("Query failed: %v", err)
					setResults(&queryResultData{Error: fmt.Sprintf("Query error: %v", err)})
					setPhase(PhaseReady)
					return
				}

				var res queryResultData
				if jsonErr := json.Unmarshal([]byte(val.String()), &res); jsonErr != nil {
					jsutil.LogError("JSON parse failed: %v", jsonErr)
					setResults(&queryResultData{Error: fmt.Sprintf("Parse error: %v", jsonErr)})
				} else {
					jsutil.LogInfo("Query successful, %d rows", len(res.Rows))
					setResults(&res)
				}
				setPhase(PhaseReady)
			})
		}()
	}

	return el.Div(
		el.Class("app"),
		el.OnMount(func(e dom.Element) {
			initDB()
		}),
		
		// Header
		el.Div(
			el.Style("display", "flex"),
			el.Style("justify-content", "space-between"),
			el.Style("align-items", "center"),
			el.Style("margin-bottom", "1rem"),
			el.H1(el.Style("margin", "0"), el.Text("DuckDB SQL Explorer")),
			el.Show(func() bool { return phase() >= PhaseReady },
				el.Span(
					el.Style("color", "#10b981"),
					el.Style("font-size", "0.75rem"),
					el.Style("font-weight", "600"),
					el.Style("background", "#10b98122"),
					el.Style("padding", "0.25rem 0.75rem"),
					el.Style("border-radius", "1rem"),
					el.Text("● DB READY"),
				),
			),
			el.Show(func() bool { return phase() == PhaseInit },
				el.Span(
					el.Style("color", "#f59e0b"),
					el.Style("font-size", "0.75rem"),
					el.Style("font-weight", "600"),
					el.Style("background", "#f59e0b22"),
					el.Style("padding", "0.25rem 0.75rem"),
					el.Style("border-radius", "1rem"),
					el.Text("○ INITIALIZING..."),
				),
			),
		),

		el.Textarea(
			el.Attr("rows", "4"),
			el.DynProp("value", func() any { return sql() }),
			el.OnInput(func(e dom.Event) {
				setSQL(e.TargetValue())
			}),
		),

		el.Div(
			el.Class("toolbar"),
			el.Button(
				el.DynText(func() string {
					p := phase()
					switch p {
					case PhaseInit: return "Loading DuckDB..."
					case PhaseRunning: return "Running..."
					default: return "Run Query"
					}
				}),
				el.OnClick(func(e dom.Event) { runQuery() }),
				el.DynProp("disabled", func() any {
					return phase() != PhaseReady
				}),
			),
			el.Span(
				el.Class("status"),
				el.DynText(func() string {
					res := results()
					if res != nil && res.Error == "" && len(res.Rows) > 0 {
						return fmt.Sprintf("%d rows returned", len(res.Rows))
					}
					return ""
				}),
			),
		),

		// Error Display
		el.Show(func() bool { 
			res := results()
			return res != nil && res.Error != ""
		}, el.Dynamic(func() el.Node {
			return el.Div(el.Class("error"), el.Text(results().Error))
		})),

		// Results Table
		el.Show(func() bool {
			res := results()
			return res != nil && res.Error == "" && len(res.Columns) > 0
		}, el.Div(
			el.Class("results"),
			el.Dynamic(func() el.Node {
				res := results()
				if res == nil { return el.Span() }

				headerCells := make([]any, len(res.Columns))
				for i, col := range res.Columns {
					headerCells[i] = el.Th(el.Text(col))
				}

				tableRows := make([]any, len(res.Rows))
				for i, row := range res.Rows {
					cells := make([]any, len(row))
					for j, cell := range row {
						cells[j] = el.Td(el.Text(cell))
					}
					tableRows[i] = el.Tr(cells...)
				}

				return el.Table(
					el.Thead(el.Tr(headerCells...)),
					el.Tbody(tableRows...),
				)
			}),
		)),
	)
}

func main() {
	el.Mount("#app", App)
	select {}
}
