package components

import (
	"testing"

	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/logging/state"
)

func TestLoggingComponent(t *testing.T) {
	reactive.CreateRoot(func(dispose func()) {
		defer dispose()
		s := state.NewLoggingState()
		if Logging(s) == nil {
			t.Fatal("expected Logging component to return a node")
		}
	})
}
