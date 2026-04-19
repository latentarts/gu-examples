package app

import (
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/webgpu/components"
	"github.com/latentarts/gu-examples/webgpu/state"
)

func App(styles el.Node) el.Node {
	s := state.NewAppState()
	return components.Screen(styles, s)
}
