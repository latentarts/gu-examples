//go:build js && wasm

package app

import (
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/launcher/components"
	"github.com/latentarts/gu-examples/launcher/state"
)

// App is the root component that composes the launcher application.
func App(styles el.Node) el.Node {
	s := state.NewLauncherState()

	return el.Div(
		el.Class("launcher"),
		styles,
		el.Div(
			el.Class("launcher__sidebar"),
			components.Sidebar(s),
		),
		el.Div(
			el.Class("launcher__main"),
			components.MobileToggle(s),
			components.Viewer(s),
		),
	)
}