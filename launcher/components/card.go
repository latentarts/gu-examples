//go:build js && wasm

package components

import (
	"fmt"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/launcher/registry"
	"github.com/latentarts/gu-examples/launcher/state"
)

// Card renders a single example card in the sidebar.
func Card(example registry.ExampleData, s *state.LauncherState) el.Node {
	isSelected := func() bool { return s.Selected() == example.ID }

	return el.Div(
		el.DynClass(func() string {
			base := "example-card"
			if isSelected() {
				base += " example-card--selected"
			}
			return base
		}),
		el.OnClick(func(e dom.Event) {
			s.SetSelected(example.ID)
		}),
		el.Img(
			el.Class("example-card__thumb"),
			el.Src(example.Thumbnail),
			el.Attr("alt", example.Name),
			el.Attr("loading", "lazy"),
		),
		el.Div(
			el.Class("example-card__info"),
			el.H3(el.Class("example-card__name"), el.Text(example.Name)),
			el.P(el.Class("example-card__desc"), el.Text(truncate(example.Description, 100))),
		),
	)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// Welcome renders the welcome screen when no example is selected.
func Welcome(s *state.LauncherState) el.Node {
	return el.Div(
		el.Class("welcome"),
		el.Div(
			el.Class("welcome__content"),
			el.H1(el.Class("welcome__title"), el.Text("gu Examples")),
			el.P(el.Class("welcome__subtitle"), el.Text("Select an example from the sidebar to get started.")),
			el.Div(
				el.Class("welcome__stats"),
				el.Span(el.Class("welcome__stat"), el.Text(fmt.Sprintf("%d examples available", len(registry.Examples())))),
			),
		),
	)
}