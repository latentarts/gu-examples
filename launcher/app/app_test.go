//go:build js && wasm

package app

import (
	"testing"

	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/launcher/state"
)

func TestAppInit(t *testing.T) {
	// Simple smoke test for App component
	styles := el.Tag("style")
	node := App(styles)
	if node == nil {
		t.Fatal("App component returned nil node")
	}
}

func TestState(t *testing.T) {
	s := state.NewLauncherState()
	if s.Selected() != "" {
		t.Errorf("expected initial selected empty, got %s", s.Selected())
	}
	s.SetSelected("counter")
	if s.Selected() != "counter" {
		t.Errorf("expected selected counter, got %s", s.Selected())
	}
}
