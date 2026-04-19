package app

import (
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/counter/components"
	"github.com/latentarts/gu-examples/counter/state"
)

// App is the root component that composes the counter application.
func App(styles el.Node) el.Node {
	s := state.NewCounterState()

	return el.Div(
		el.Class("app"),
		styles,
		el.H1(el.Text("gu Counter")),
		components.Display(s),
		components.Controls(s),
		components.ThemeToggle(s),
		el.Show(
			func() bool { return s.Count() > 10 },
			el.P(
				el.Text("Count is over 10!"),
				el.Style("color", "var(--gu-color-success)"),
				el.Style("font-weight", "bold"),
			),
		),
	)
}
