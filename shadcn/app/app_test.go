//go:build js && wasm

package app

import (
	"testing"

	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/shadcn/components"
)

func TestSafeSectionsRender(t *testing.T) {
	reactive.CreateRoot(func(dispose func()) {
		defer dispose()

		nodes := []struct {
			name string
			node any
		}{
			{name: "DatePickerDemo", node: components.DatePickerDemo()},
			{name: "CarouselDemo", node: components.CarouselDemo()},
			{name: "ButtonGroupDemo", node: components.ButtonGroupDemo()},
		}

		for _, tc := range nodes {
			if tc.node == nil {
				t.Fatalf("%s returned nil", tc.name)
			}
		}
	})
}
