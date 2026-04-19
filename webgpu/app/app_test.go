//go:build js && wasm

package app

import (
	"testing"

	"github.com/latentart/gu/el"
	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/webgpu/components"
	"github.com/latentarts/gu-examples/webgpu/state"
)

func TestMatrixHelpers(t *testing.T) {
	identity := state.Mat4Identity()
	product := state.Mat4Multiply(identity, identity)
	if product != identity {
		t.Fatal("expected identity multiplied by identity to remain unchanged")
	}

	perspective := state.Mat4Perspective(1.0, 1.5, 0.1, 100.0)
	if perspective[11] != -1 {
		t.Fatalf("expected perspective matrix w term to be -1, got %v", perspective[11])
	}

	view := state.Mat4LookAt([3]float32{0, 0, 5}, [3]float32{0, 0, 0}, [3]float32{0, 1, 0})
	if view[15] != 1 {
		t.Fatalf("expected look-at matrix bottom-right value to be 1, got %v", view[15])
	}
}

func TestCubeDataAndAppRender(t *testing.T) {
	verts, indices := state.BuildCubeDataForTest()
	if len(verts) == 0 || len(indices) == 0 {
		t.Fatal("expected cube data to be populated")
	}

	reactive.CreateRoot(func(dispose func()) {
		defer dispose()
		s := state.NewAppState()
		if App(el.Tag("style")) == nil {
			t.Fatal("App returned nil")
		}
		if components.Screen(el.Tag("style"), s) == nil {
			t.Fatal("Screen returned nil")
		}
	})
}
