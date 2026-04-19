package app

import (
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/reporting/components"
	"github.com/latentarts/gu-examples/reporting/state"
)

// App is the root component that composes the reporting application.
func App(styles el.Node) el.Node {
	s := state.NewReportingState()

	return el.Div(
		el.Class("app"),
		styles,
		el.H1(el.Text("Minimalist Reporter")),
		components.Uploader(s),
		el.Show(func() bool {
			return len(s.Columns()) > 0
		},
			components.Table(s),
		),
	)
}
