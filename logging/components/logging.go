package components

import (
	"fmt"
	"strconv"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/logging/state"
)

// Logging is the main component for the logging showcase.
func Logging(s *state.LoggingState) el.Node {
	return el.Div(
		el.Class("app"),

		el.H1(el.Text("gu Logging Showcase")),
		el.P(el.Class("subtitle"), el.Text("Open DevTools (F12) for console output. Enable the WASM observability window with ?DEBUG_CONSOLE=true or ?GU_DEBUG_CONSOLE_ENABLED=1, or GU_DEBUG_CONSOLE_ENABLED=1 with gu dev (see DEBUG.md).")),

		// Log level control
		el.Div(
			el.Class("section"),
			el.H2(el.Text("Log Level")),
			el.P(el.Class("desc"), el.Text("Messages below the selected level are suppressed.")),
			el.Div(
				el.Class("level-row"),
				el.Select(
					el.Class("level-select"),
					el.Value(strconv.Itoa(s.CurrentLevel())),
					el.Option(el.Value("0"), el.Text("Debug")),
					el.Option(el.Value("1"), el.Text("Info")),
					el.Option(el.Value("2"), el.Text("Warning")),
					el.Option(el.Value("3"), el.Text("Error")),
					el.Option(el.Value("4"), el.Text("Off")),
					el.OnChange(func(e dom.Event) {
						s.SetLevel(e.TargetValue())
					}),
				),
				el.Span(
					el.Class("level-status"),
					el.DynText(func() string {
						return fmt.Sprintf("Active: %s", state.LevelNames[s.CurrentLevel()])
					}),
				),
			),
		),

		// Log level buttons
		el.Div(
			el.Class("section"),
			el.H2(el.Text("Log Levels")),
			el.Div(
				el.Class("buttons"),
				el.Button(
					el.Class("btn debug"),
					el.Text("Debug"),
					el.OnClick(func(e dom.Event) {
						s.LogDebug()
					}),
				),
				el.Button(
					el.Class("btn info"),
					el.Text("Info"),
					el.OnClick(func(e dom.Event) {
						s.LogInfo()
					}),
				),
				el.Button(
					el.Class("btn warn"),
					el.Text("Warning"),
					el.OnClick(func(e dom.Event) {
						s.LogWarn()
					}),
				),
				el.Button(
					el.Class("btn error"),
					el.Text("Error"),
					el.OnClick(func(e dom.Event) {
						s.LogError()
					}),
				),
			),
		),

		// Exception
		el.Div(
			el.Class("section"),
			el.H2(el.Text("Exception with Stack Trace")),
			el.P(el.Class("desc"), el.Text("Triggers a real multi-frame call chain (processOrder → validatePayment → chargeCard) that fails.")),
			el.Div(
				el.Class("buttons"),
				el.Button(
					el.Class("btn error"),
					el.Text("Trigger Exception"),
					el.OnClick(func(e dom.Event) {
						s.TriggerException()
					}),
				),
				el.Button(
					el.Class("btn error"),
					el.Text("Catch Panic"),
					el.OnClick(func(e dom.Event) {
						s.CatchPanic()
					}),
				),
			),
		),

		// Status
		el.Div(
			el.Class("counter"),
			el.DynText(func() string {
				n := s.ClickCount()
				if n == 0 {
					return "Click a button — then check the DevTools console."
				}
				return fmt.Sprintf("%d log actions triggered — check DevTools console.", n)
			}),
		),
	)
}
