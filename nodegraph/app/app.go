package app

import (
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/nodegraph/components"
	"github.com/latentarts/gu-examples/nodegraph/state"
)

func App(styles el.Node) el.Node {
	s := state.NewEditorState()
	return components.Editor(styles, s)
}
