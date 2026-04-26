//go:build js && wasm

package components

import (
	"fmt"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/duckdb/state"
)

// Explorer renders the DuckDB SQL explorer UI.
func Explorer(s *state.DuckDBState) el.Node {
	return el.Div(
		el.Class("app"),
		el.OnMount(func(e dom.Element) {
			s.InitDB()
		}),
		
		header(s),
		editor(s),
		toolbar(s),
		errorDisplay(s),
		resultsTable(s),
	)
}

func header(s *state.DuckDBState) el.Node {
	return el.Div(
		el.Style("display", "flex"),
		el.Style("justify-content", "space-between"),
		el.Style("align-items", "center"),
		el.Style("margin-bottom", "1rem"),
		el.H1(el.Style("margin", "0"), el.Text("DuckDB SQL Explorer")),
		el.Show(func() bool { return s.Phase() >= state.PhaseReady },
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
		el.Show(func() bool { return s.Phase() == state.PhaseInit },
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
	)
}

func editor(s *state.DuckDBState) el.Node {
	return el.Textarea(
		el.Attr("rows", "4"),
		el.DynProp("value", func() any { return s.SQL() }),
		el.OnInput(func(e dom.Event) {
			s.SetSQL(e.TargetValue())
		}),
	)
}

func toolbar(s *state.DuckDBState) el.Node {
	return el.Div(
		el.Class("toolbar"),
		el.Button(
			el.DynText(func() string {
				p := s.Phase()
				switch p {
				case state.PhaseInit: return "Loading DuckDB..."
				case state.PhaseRunning: return "Running..."
				default: return "Run Query"
				}
			}),
			el.OnClick(func(e dom.Event) { s.RunQuery() }),
			el.DynProp("disabled", func() any {
				return s.Phase() != state.PhaseReady
			}),
		),
		el.Span(
			el.Class("status"),
			el.DynText(func() string {
				res := s.Results()
				if res != nil && res.Error == "" && len(res.Rows) > 0 {
					return fmt.Sprintf("%d rows returned", len(res.Rows))
				}
				return ""
			}),
		),
	)
}

func errorDisplay(s *state.DuckDBState) el.Node {
	return el.Show(func() bool { 
		res := s.Results()
		return res != nil && res.Error != ""
	}, el.Dynamic(func() el.Node {
		return el.Div(el.Class("error"), el.Text(s.Results().Error))
	}))
}

func resultsTable(s *state.DuckDBState) el.Node {
	return el.Show(func() bool {
		res := s.Results()
		return res != nil && res.Error == "" && len(res.Columns) > 0
	}, el.Div(
		el.Class("results"),
		el.Dynamic(func() el.Node {
			res := s.Results()
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
	))
}
