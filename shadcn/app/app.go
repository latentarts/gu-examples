package app

import (
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/shadcn/components"
)

func App(styles el.Node) el.Node {
	return components.Showcase(styles)
}
