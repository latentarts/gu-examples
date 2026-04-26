//go:build js && wasm

package components

import (
	"sort"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/reporting/state"
)

// Table component renders a reactive, sortable, and virtualized data table.
func Table(s *state.ReportingState) el.Node {
	visibleCount, setVisibleCount := reactive.NewSignal(50)
	draggedIdx, setDraggedIdx := reactive.NewSignal(-1)

	var tbodyRef dom.Element
	tbodyReady, setTbodyReady := reactive.NewSignal(false)
	var renderedCount int
	var currentSortedRows [][]string

	// Effect 1: Handle Sorting and Initial Render
	// This only re-runs when data or sort criteria change, NOT on scroll.
	reactive.CreateEffect(func() {
		sr := s.Rows()
		sc := s.SortCol()
		asc := s.SortAsc()
		ready := tbodyReady()

		if sc != -1 {
			// Create a copy for sorting
			sorted := make([][]string, len(sr))
			copy(sorted, sr)
			sort.SliceStable(sorted, func(i, j int) bool {
				if asc {
					return sorted[i][sc] < sorted[j][sc]
				}
				return sorted[i][sc] > sorted[j][sc]
			})
			currentSortedRows = sorted
		} else {
			currentSortedRows = sr
		}

		if ready && !tbodyRef.IsNull() {
			tbodyRef.SetInnerHTML("") // Clear existing rows
			renderedCount = 0

			// Render initial batch
			limit := 50
			if limit > len(currentSortedRows) {
				limit = len(currentSortedRows)
			}
			appendRows(tbodyRef, currentSortedRows[0:limit])
			renderedCount = limit
			setVisibleCount(50) // Reset infinite scroll state
		}
	})

	// Effect 2: Handle Infinite Scroll (visibleCount)
	// This appends ONLY new rows to the existing DOM.
	reactive.CreateEffect(func() {
		limit := visibleCount()
		ready := tbodyReady()

		if ready && !tbodyRef.IsNull() && currentSortedRows != nil && limit > renderedCount {
			if limit > len(currentSortedRows) {
				limit = len(currentSortedRows)
			}
			if limit > renderedCount {
				appendRows(tbodyRef, currentSortedRows[renderedCount:limit])
				renderedCount = limit
			}
		}
	})

	return el.Div(
		el.Class("table-container"),
		el.On("scroll", func(e dom.Event) {
			target := e.Target()
			sh := target.Value.Get("scrollHeight").Int()
			st := target.Value.Get("scrollTop").Int()
			ch := target.Value.Get("clientHeight").Int()
			if sh-st-ch < 100 {
				setVisibleCount(visibleCount() + 50)
			}
		}),
		el.Table(
			el.Dynamic(func() el.Node {
				cols := s.Columns()
				ths := make([]any, len(cols))
				for i, col := range cols {
					idx := i
					ths[i] = el.Th(
						el.Text(col),
						el.Attr("draggable", "true"),
						el.OnClick(func(e dom.Event) {
							if s.SortCol() == idx {
								s.SetSortAsc(!s.SortAsc())
							} else {
								s.SetSortCol(idx)
								s.SetSortAsc(true)
							}
						}),
						el.On("dragstart", func(e dom.Event) {
							setDraggedIdx(idx)
							e.Value.Get("dataTransfer").Set("effectAllowed", "move")
						}),
						el.On("dragover", func(e dom.Event) {
							e.PreventDefault()
						}),
						el.On("drop", func(e dom.Event) {
							e.PreventDefault()
							from := draggedIdx()
							to := idx
							if from != -1 && from != to {
								newCols := state.Reorder(cols, from, to)
								r := s.Rows()
								newRows := make([][]string, len(r))
								for j, row := range r {
									newRows[j] = state.Reorder(row, from, to)
								}
								s.SetColumns(newCols)
								s.SetRows(newRows)
								s.SetSortCol(-1)
							}
							setDraggedIdx(-1)
						}),
						el.Show(func() bool { return s.SortCol() == idx },
							el.Span(el.Class("sort-icon"), el.DynText(func() string {
								if s.SortAsc() {
									return "↑"
								}
								return "↓"
							})),
						),
					)
				}
				return el.Thead(el.Tr(ths...))
			}),
			el.Tbody(el.OnMount(func(e dom.Element) {
				tbodyRef = e
				setTbodyReady(true)
			})),
		),
	)
}

func appendRows(parent dom.Element, rows [][]string) {
	for _, row := range rows {
		tr := dom.CreateElement("tr")
		for _, cell := range row {
			td := dom.CreateElement("td")
			td.SetTextContent(cell)
			tr.AppendChild(td)
		}
		parent.AppendChild(tr)
	}
}
