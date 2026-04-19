package app

import (
	"testing"

	"github.com/latentart/gu/el"
	"github.com/latentart/gu/reactive"
)

func TestApp(t *testing.T) {
	reactive.CreateRoot(func(dispose func()) {
		defer dispose()
		if App(el.Div()) == nil {
			t.Fatal("expected App component to return a node")
		}
	})
}
