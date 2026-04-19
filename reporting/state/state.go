package state

import (
	"github.com/latentart/gu/reactive"
)

// ReportingState centralizes the application's reactive state and data management.
type ReportingState struct {
	Columns    func() []string
	SetColumns func([]string)
	ColumnsVer func() int

	Rows    func() [][]string
	SetRows func([][]string)
	RowsVer func() int

	FoundCount    func() int
	SetFoundCount func(int)

	SortCol    func() int
	SetSortCol func(int)
	SortAsc    func() bool
	SetSortAsc func(bool)
}

// NewReportingState initializes a new state container with default values.
func NewReportingState() *ReportingState {
	cv, setCv := reactive.NewSignal(0)
	var cols []string

	rv, setRv := reactive.NewSignal(0)
	var rows [][]string

	fc, setFc := reactive.NewSignal(0)

	sc, setSc := reactive.NewSignal(-1)
	sa, setSa := reactive.NewSignal(true)

	return &ReportingState{
		Columns: func() []string { _ = cv(); return cols },
		SetColumns: func(c []string) {
			cols = c
			setCv(cv() + 1)
		},
		ColumnsVer: cv,

		Rows: func() [][]string { _ = rv(); return rows },
		SetRows: func(r [][]string) {
			rows = r
			setRv(rv() + 1)
		},
		RowsVer: rv,

		FoundCount: fc,
		SetFoundCount: setFc,

		SortCol:    sc,
		SetSortCol: setSc,
		SortAsc:    sa,
		SetSortAsc: setSa,
	}
}

// GetRowCount returns the current number of rows or the reported count if loading.
func (s *ReportingState) GetRowCount() int {
	f := s.FoundCount()
	r := len(s.Rows())
	if f > r {
		return f
	}
	return r
}
