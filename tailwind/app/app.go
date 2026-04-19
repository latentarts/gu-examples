package app

import (
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/tailwind/components"
	"github.com/latentarts/gu-examples/tailwind/state"
)

func App(styles el.Node) el.Node {
	s := state.NewShowcaseState()
	return components.Showcase(styles, s)
}
