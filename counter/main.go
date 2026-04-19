//go:build js && wasm

package main

import (
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/theme"
	"github.com/latentarts/gu-examples/counter/app"
)

func main() {
	theme.SetTheme(theme.DefaultLight())
	el.Mount("#app", func() el.Node {
		return app.App(GlobalStyles())
	})
	select {}
}
