package app

import (
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/nodegraph-tauri/frontend/components"
	"github.com/latentarts/gu-examples/nodegraph-tauri/frontend/state"
)

func App(styles el.Node) el.Node {
	s := state.NewEditorState()
	return components.Editor(styles, s)
}
