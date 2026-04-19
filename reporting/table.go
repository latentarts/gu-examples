package main

import (
	"sort"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/reactive"
)

func Table(columns func() []string, setColumns func([]string), rows func() [][]string, setRows func([][]string)) el.Node {
	sortCol, setSortCol := reactive.NewSignal(-1)
	sortAsc, setSortAsc := reactive.NewSignal(true)
	visibleCount, setVisibleCount := reactive.NewSignal(50)
	draggedIdx, setDraggedIdx := reactive.NewSignal(-1)

	var tbodyRef dom.Element
	tbodyReady, setTbodyReady := reactive.NewSignal(false)
	var renderedCount int
	var currentSortedRows [][]string

	// Effect 1: Handle Sorting and Initial Render
	// This only re-runs when data or sort criteria change, NOT on scroll.
	reactive.CreateEffect(func() {
		sr := rows()
		sc := sortCol()
		asc := sortAsc()
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
				cols := columns()
				ths := make([]any, len(cols))
				for i, col := range cols {
					idx := i
					ths[i] = el.Th(
						el.Text(col),
						el.Attr("draggable", "true"),
						el.OnClick(func(e dom.Event) {
							if sortCol() == idx {
								setSortAsc(!sortAsc())
							} else {
								setSortCol(idx)
								setSortAsc(true)
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
								newCols := reorder(cols, from, to)
								r := rows()
								newRows := make([][]string, len(r))
								for j, row := range r {
									newRows[j] = reorder(row, from, to)
								}
								setColumns(newCols)
								setRows(newRows)
								setSortCol(-1)
							}
							setDraggedIdx(-1)
						}),
						el.Show(func() bool { return sortCol() == idx },
							el.Span(el.Class("sort-icon"), el.DynText(func() string {
								if sortAsc() {
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

func reorder[T any](s []T, from, to int) []T {
	if from == to {
		return s
	}
	res := make([]T, len(s))
	copy(res, s)
	val := res[from]
	res = append(res[:from], res[from+1:]...)
	res = append(res[:to], append([]T{val}, res[to:]...)...)
	return res
}
