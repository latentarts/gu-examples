package main

import (
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/reporting/app"
)

func main() {
	el.Mount("#app", func() el.Node {
		return app.App(GlobalStyles())
	})
	select {}
}
