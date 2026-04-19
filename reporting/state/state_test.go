//go:build js && wasm

package state

import (
	"testing"
)

func TestFormatCount(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1234567, "1,234,567"},
		{1000000000, "1,000,000,000"},
	}

	for _, tt := range tests {
		got := FormatCount(tt.input)
		if got != tt.expected {
			t.Errorf("FormatCount(%d) = %s; want %s", tt.input, got, tt.expected)
		}
	}
}

func TestReorder(t *testing.T) {
	input := []string{"a", "b", "c", "d"}
	
	// Move "b" (idx 1) to end (idx 3)
	got := Reorder(input, 1, 3)
	expected := []string{"a", "c", "d", "b"}
	for i, v := range got {
		if v != expected[i] {
			t.Errorf("Reorder failed: got %v, want %v", got, expected)
			break
		}
	}

	// Move "d" (idx 3) to start (idx 0)
	got = Reorder(input, 3, 0)
	expected = []string{"d", "a", "b", "c"}
	for i, v := range got {
		if v != expected[i] {
			t.Errorf("Reorder failed: got %v, want %v", got, expected)
			break
		}
	}

	// Move to same position
	got = Reorder(input, 2, 2)
	for i, v := range got {
		if v != input[i] {
			t.Errorf("Reorder failed: got %v, want %v", got, input)
			break
		}
	}
}

func TestReportingState(t *testing.T) {
	s := NewReportingState()

	if s.GetRowCount() != 0 {
		t.Errorf("expected initial row count 0, got %d", s.GetRowCount())
	}

	cols := []string{"Name", "Age"}
	s.SetColumns(cols)
	if len(s.Columns()) != 2 {
		t.Errorf("expected 2 columns, got %d", len(s.Columns()))
	}

	rows := [][]string{{"Alice", "30"}, {"Bob", "25"}}
	s.SetRows(rows)
	if len(s.Rows()) != 2 {
		t.Errorf("expected 2 rows, got %d", len(s.Rows()))
	}

	if s.GetRowCount() != 2 {
		t.Errorf("expected row count 2, got %d", s.GetRowCount())
	}

	s.SetFoundCount(100)
	if s.GetRowCount() != 100 {
		t.Errorf("expected row count 100 (from found count), got %d", s.GetRowCount())
	}

	s.SetSortCol(1)
	if s.SortCol() != 1 {
		t.Errorf("expected sort col 1, got %d", s.SortCol())
	}

	s.SetSortAsc(false)
	if s.SortAsc() != false {
		t.Error("expected sort asc false")
	}
}
