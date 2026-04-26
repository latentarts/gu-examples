//go:build js && wasm

package state

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
)

type QueryResultData struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
	Error   string
}

type AppPhase int

const (
	PhaseInit AppPhase = iota
	PhaseReady
	PhaseRunning
)

// DuckDBState manages the reactive state of the duckdb application.
type DuckDBState struct {
	SQL    func() string
	SetSQL func(string)

	Phase    func() AppPhase
	SetPhase func(AppPhase)

	Results    func() *QueryResultData
	SetResults func(*QueryResultData)
}

// NewDuckDBState initializes a new duckdb state.
func NewDuckDBState() *DuckDBState {
	sql, setSQL := reactive.NewSignal("SELECT category, COUNT(*) AS cnt, ROUND(AVG(price), 2) AS avg_price\nFROM sales\nGROUP BY category\nORDER BY cnt DESC")
	phase, setPhase := reactive.NewSignal(PhaseInit)
	results, setResults := reactive.NewSignal[*QueryResultData](nil)

	return &DuckDBState{
		SQL:        sql,
		SetSQL:     setSQL,
		Phase:      phase,
		SetPhase:   setPhase,
		Results:    results,
		SetResults: setResults,
	}
}

// InitDB monitors DuckDB readiness.
func (s *DuckDBState) InitDB() {
	go func() {
		for {
			app := js.Global().Get("App")
			if !app.IsUndefined() && app.Get("ready").Bool() {
				jsutil.LogInfo("DuckDB ready detected in Go")
				s.SetPhase(PhaseReady)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

// RunQuery executes the current SQL query.
func (s *DuckDBState) RunQuery() {
	if s.Phase() != PhaseReady {
		return
	}

	s.SetPhase(PhaseRunning)
	jsutil.LogDebug("Starting query execution...")

	go func() {
		currentSQL := s.SQL()
		promise := js.Global().Get("App").Call("query", currentSQL)
		val, err := jsutil.Await(promise)

		if err != nil {
			jsutil.LogError("Query failed: %v", err)
			s.SetResults(&QueryResultData{Error: fmt.Sprintf("Query error: %v", err)})
			s.SetPhase(PhaseReady)
			return
		}

		var res QueryResultData
		if jsonErr := json.Unmarshal([]byte(val.String()), &res); jsonErr != nil {
			jsutil.LogError("JSON parse failed: %v", jsonErr)
			s.SetResults(&QueryResultData{Error: fmt.Sprintf("Parse error: %v", jsonErr)})
		} else {
			jsutil.LogInfo("Query successful, %d rows", len(res.Rows))
			s.SetResults(&res)
		}
		s.SetPhase(PhaseReady)
	}()
}
