//go:build js && wasm

package components

import (
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/launcher/registry"
	"github.com/latentarts/gu-examples/launcher/state"
)

// Viewer renders the main content area. When an example is selected,
// it shows an iframe loading that example. Otherwise, it shows a welcome screen.
func Viewer(s *state.LauncherState) el.Node {
	return el.Dynamic(func() el.Node {
		selected := s.Selected()
		if selected == "" {
			return Welcome(s)
		}

		ex := registry.FindByID(selected)
		if ex == nil {
			return Welcome(s)
		}

		return el.Div(
			el.Class("viewer"),
			el.Div(
				el.Class("viewer__header"),
				el.H2(el.Class("viewer__title"), el.Text(ex.Name)),
				el.P(el.Class("viewer__desc"), el.Text(ex.Description)),
			),
			el.Tag("iframe",
				el.Class("viewer__iframe"),
				el.Src("/apps/"+ex.ID+"/index.html"),
				el.Attr("title", ex.Name),
				el.Attr("loading", "lazy"),
			),
		)
	})
}