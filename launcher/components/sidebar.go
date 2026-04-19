//go:build js && wasm

package components

import (
	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/launcher/registry"
	"github.com/latentarts/gu-examples/launcher/state"
)

// Sidebar renders the scrollable list of example cards.
func Sidebar(s *state.LauncherState) el.Node {
	examples := registry.Examples()

	args := []any{el.Class("sidebar")}
	for i := range examples {
		ex := examples[i]
		args = append(args, Card(ex, s))
	}

	// Mobile toggle button (shown when sidebar is collapsed)
	args = append(args,
		el.Div(el.Class("sidebar__spacer")),
		el.Div(
			el.Class("sidebar__footer"),
			el.A(
				el.Class("sidebar__link"),
				el.Href("https://github.com/latentart/gu"),
				el.Attr("target", "_blank"),
				el.Text("gu on GitHub"),
			),
		),
	)

	return el.Div(args...)
}

// MobileToggle renders a button to toggle the sidebar on mobile.
func MobileToggle(s *state.LauncherState) el.Node {
	return el.Button(
		el.Class("mobile-toggle"),
		el.OnClick(func(e dom.Event) {
			s.SetSidebarOpen(!s.SidebarOpen())
		}),
		el.Text("☰"),
	)
}