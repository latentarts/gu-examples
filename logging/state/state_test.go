package state

import (
	"testing"

	"github.com/latentart/gu/reactive"
)

func TestLoggingState(t *testing.T) {
	reactive.CreateRoot(func(dispose func()) {
		defer dispose()
		s := NewLoggingState()

		if s.ClickCount() != 0 {
			t.Errorf("expected click count 0, got %d", s.ClickCount())
		}

		s.Bump()
		if s.ClickCount() != 1 {
			t.Errorf("expected click count 1, got %d", s.ClickCount())
		}

		// Test log level change (mocking jsutil is hard in wasm tests if it calls actual JS,
		// but we can at least check if the signal updates)
		s.SetLevel("2") // Warning
		if s.CurrentLevel() != 2 {
			t.Errorf("expected level 2, got %d", s.CurrentLevel())
		}
	})
}
