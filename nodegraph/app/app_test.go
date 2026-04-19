//go:build js && wasm

package app

import (
	"testing"

	"github.com/latentart/gu/el"
	"github.com/latentart/gu/reactive"
)

func TestAppRenders(t *testing.T) {
	reactive.CreateRoot(func(dispose func()) {
		defer dispose()
		if App(el.Tag("style")) == nil {
			t.Fatal("App returned nil")
		}
	})
}
