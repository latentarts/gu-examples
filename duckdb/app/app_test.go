//go:build js && wasm

package app

import (
        "testing"

        "github.com/latentart/gu/el"
        "github.com/latentart/gu/reactive"
)

func TestAppInit(t *testing.T) {
        reactive.CreateRoot(func(dispose func()) {
                defer dispose()
                styles := el.Tag("style")
                node := App(styles)
                if node == nil {
                        t.Fatal("App component returned nil node")
                }
        })
}
