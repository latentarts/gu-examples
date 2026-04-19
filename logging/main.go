//go:build js && wasm

package main

import (
	"github.com/latentart/gu/debugutil"
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/logging/app"
)

func main() {
	_ = debugutil.WithOp("main.mount", func() error {
		el.Mount("#app", func() el.Node {
			return app.App(Styles())
		})
		return nil
	})
	select {}
}
