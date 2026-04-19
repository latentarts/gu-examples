//go:build js && wasm

package components

import (
	"testing"

	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/reporting/state"
)

func TestTableComponentInit(t *testing.T) {
	reactive.CreateRoot(func(dispose func()) {
		defer dispose()
		s := state.NewReportingState()
		node := Table(s)
		if node == nil {
			t.Fatal("Table component returned nil node")
		}
	})
}

func TestUploaderComponentInit(t *testing.T) {
	reactive.CreateRoot(func(dispose func()) {
		defer dispose()
		s := state.NewReportingState()
		node := Uploader(s)
		if node == nil {
			t.Fatal("Uploader component returned nil node")
		}
	})
}
