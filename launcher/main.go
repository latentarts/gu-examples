//go:build js && wasm

package main

import (
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/theme"
	"github.com/latentarts/gu-examples/launcher/app"
)

func main() {
	theme.SetTheme(theme.DefaultDark())
	el.Mount("#app", func() el.Node {
		return app.App(GlobalStyles())
	})
	select {}
}