package app

import (
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/duckdb/components"
	"github.com/latentarts/gu-examples/duckdb/state"
)

// App is the root component that composes the DuckDB application.
func App(styles el.Node) el.Node {
	s := state.NewDuckDBState()

	return el.Div(
		styles,
		components.Explorer(s),
	)
}
