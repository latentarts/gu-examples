//go:build js && wasm

package state

import (
	"testing"

	"github.com/latentart/gu/reactive"
)

func TestCounterState(t *testing.T) {
	reactive.CreateRoot(func(dispose func()) {
		defer dispose()
		s := NewCounterState()

		if s.Count() != 0 {
			t.Errorf("expected initial count 0, got %d", s.Count())
		}

		s.SetCount(5)
		if s.Count() != 5 {
			t.Errorf("expected count 5, got %d", s.Count())
		}

		if s.Doubled() != 10 {
			t.Errorf("expected doubled 10, got %d", s.Doubled())
		}

		if s.IsDark() != false {
			t.Error("expected initial isDark false")
		}

		s.SetIsDark(true)
		if s.IsDark() != true {
			t.Error("expected isDark true")
		}
	})
}
