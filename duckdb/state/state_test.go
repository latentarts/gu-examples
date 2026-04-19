//go:build js && wasm

package state

import (
	"testing"

	"github.com/latentart/gu/reactive"
)

func TestDuckDBState(t *testing.T) {
	reactive.CreateRoot(func(dispose func()) {
		defer dispose()
		s := NewDuckDBState()

		if s.Phase() != PhaseInit {
			t.Errorf("expected initial phase PhaseInit, got %v", s.Phase())
		}

		s.SetPhase(PhaseReady)
		if s.Phase() != PhaseReady {
			t.Errorf("expected phase PhaseReady, got %v", s.Phase())
		}

		newSQL := "SELECT 1"
		s.SetSQL(newSQL)
		if s.SQL() != newSQL {
			t.Errorf("expected SQL %s, got %s", newSQL, s.SQL())
		}

		if s.Results() != nil {
			t.Error("expected initial results to be nil")
		}

		res := &QueryResultData{Columns: []string{"col1"}, Rows: [][]string{{"val1"}}}
		s.SetResults(res)
		if s.Results() != res {
			t.Error("failed to set results")
		}
	})
}
