//go:build js && wasm

package components

import (
	"testing"

	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/duckdb/state"
)

func TestExplorer(t *testing.T) {
	reactive.CreateRoot(func(dispose func()) {
		defer dispose()
		s := state.NewDuckDBState()
		node := Explorer(s)
		if node == nil {
			t.Fatal("Explorer returned nil node")
		}
	})
}
