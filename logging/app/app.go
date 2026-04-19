package app

import (
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/logging/components"
	"github.com/latentarts/gu-examples/logging/state"
)

// App is the root component that composes the Logging application.
func App(styles el.Node) el.Node {
	s := state.NewLoggingState()

	return el.Div(
		styles,
		components.Logging(s),
	)
}
