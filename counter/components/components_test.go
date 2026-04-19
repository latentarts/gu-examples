//go:build js && wasm

package components

import (
	"testing"

	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/counter/state"
)

func TestCounterComponents(t *testing.T) {
	reactive.CreateRoot(func(dispose func()) {
		defer dispose()
		s := state.NewCounterState()

		if Display(s) == nil {
			t.Error("Display returned nil")
		}
		if Controls(s) == nil {
			t.Error("Controls returned nil")
		}
		if ThemeToggle(s) == nil {
			t.Error("ThemeToggle returned nil")
		}
	})
}
