package components

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"syscall/js"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/jsutil"
	"github.com/latentarts/gu-examples/reporting/state"
)

// Uploader component handles CSV file selection and parsing.
func Uploader(s *state.ReportingState) el.Node {
	return el.Div(
		el.Class("uploader"),
		el.P(el.Text("Drop a CSV file here or click to upload")),
		el.Show(func() bool { return s.GetRowCount() > 0 },
			el.P(
				el.Style("color", "#38bdf8"),
				el.Style("font-weight", "500"),
				el.Style("margin-top", "0.5rem"),
				el.DynText(func() string {
					return fmt.Sprintf("✓ %s records loaded", state.FormatCount(s.GetRowCount()))
				}),
			),
		),
		el.Input(
			el.Attr("type", "file"),
			el.Attr("accept", ".csv"),
			el.OnChange(func(e dom.Event) {
				files := e.Value.Get("target").Get("files")
				if files.Length() == 0 {
					return
				}
				file := files.Index(0)
				go func() {
					promise := file.Call("arrayBuffer")
					val, err := jsutil.Await(promise)
					if err != nil {
						jsutil.LogError("failed to read file: %v", err)
						return
					}
					// Raw JS ArrayBuffer -> Go bytes
					uint8Array := js.Global().Get("Uint8Array").New(val)
					length := uint8Array.Get("length").Int()
					buf := make([]byte, length)
					js.CopyBytesToGo(buf, uint8Array)

					// 1. FAST PASS: Count records without allocations
					countReader := csv.NewReader(bytes.NewReader(buf))
					countReader.ReuseRecord = true
					count := 0
					for {
						_, err := countReader.Read()
						if err == io.EOF {
							break
						}
						if err != nil {
							jsutil.LogError("failed to count records: %v", err)
							return
						}
						count++
					}

					if count > 0 {
						// Exclude header from the shown record count
						s.SetFoundCount(count - 1)
					}

					// 2. DATA PASS: Parse with pre-allocated slice
					dataReader := csv.NewReader(bytes.NewReader(buf))
					records := make([][]string, 0, count)
					for {
						record, err := dataReader.Read()
						if err == io.EOF {
							break
						}
						if err != nil {
							jsutil.LogError("failed to parse CSV: %v", err)
							return
						}
						records = append(records, record)
					}

					if len(records) > 0 {
						s.SetColumns(records[0])
						s.SetRows(records[1:])
						s.SetSortCol(-1) // Reset sort on new data
					}
				}()
			}),
		),
	)
}
