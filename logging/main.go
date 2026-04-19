//go:build js && wasm

package main

import (
	"fmt"
	"strconv"

	"github.com/latentart/gu/debugutil"
	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
)

// Simulated app state for realistic log messages.
var (
	requestID   = 1042
	userEmail   = "alice@example.com"
	dbLatency   = 230.5
	cacheHits   = 847
	cacheMisses = 53
)

// processOrder simulates a multi-layer call that fails,
// producing a real stack trace with multiple frames.
func processOrder(orderID int, total float64) error {
	return validatePayment(orderID, total)
}

func validatePayment(orderID int, amount float64) error {
	return chargeCard(orderID, "tok_visa_4242", amount)
}

func chargeCard(orderID int, token string, amount float64) error {
	jsutil.LogInfo("charging card %s for order #%d ($%.2f)", token, orderID, amount)
	return fmt.Errorf("payment declined: card %s has insufficient funds for $%.2f (order #%d)", token, amount, orderID)
}

var levelNames = []string{"Debug", "Info", "Warning", "Error", "Off"}

func App() el.Node {
	jsutil.LogInfo("logging showcase mounted")
	clickCount, setClickCount := reactive.NewSignal(0)
	bump := func() { setClickCount(clickCount() + 1) }

	currentLevel, setCurrentLevel := reactive.NewSignal(int(jsutil.GetLogLevel()))

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
					el.Value(strconv.Itoa(currentLevel())),
					el.Option(el.Value("0"), el.Text("Debug")),
					el.Option(el.Value("1"), el.Text("Info")),
					el.Option(el.Value("2"), el.Text("Warning")),
					el.Option(el.Value("3"), el.Text("Error")),
					el.Option(el.Value("4"), el.Text("Off")),
					el.OnChange(func(e dom.Event) {
						_ = debugutil.WithOp("logging.set_level", func() error {
							v, _ := strconv.Atoi(e.TargetValue())
							jsutil.SetLogLevel(jsutil.LogLevel(v))
							setCurrentLevel(v)
							jsutil.LogInfo("log level changed to %s", levelNames[v])
							return nil
						})
					}),
				),
				el.Span(
					el.Class("level-status"),
					el.DynText(func() string {
						return fmt.Sprintf("Active: %s", levelNames[currentLevel()])
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
						_ = debugutil.WithOp("logging.btn_debug", func() error {
							jsutil.LogDebug("cache stats: %d hits / %d misses (%.1f%% hit rate)",
								cacheHits, cacheMisses, float64(cacheHits)/float64(cacheHits+cacheMisses)*100,
								jsutil.LogFields{"hits": cacheHits, "misses": cacheMisses})
							bump()
							return nil
						})
					}),
				),
				el.Button(
					el.Class("btn info"),
					el.Text("Info"),
					el.OnClick(func(e dom.Event) {
						_ = debugutil.WithOp("logging.btn_info", func() error {
							jsutil.LogInfo("request #%d: user %s authenticated, db latency %.1fms",
								requestID, userEmail, dbLatency,
								jsutil.LogFields{
									"request_id": requestID,
									"user":       userEmail,
									"db_ms":      dbLatency,
								})
							requestID++
							bump()
							return nil
						})
					}),
				),
				el.Button(
					el.Class("btn warn"),
					el.Text("Warning"),
					el.OnClick(func(e dom.Event) {
						_ = debugutil.WithOp("logging.btn_warn", func() error {
							jsutil.LogWarn("request #%d: db latency %.1fms exceeds 200ms threshold for user %s",
								requestID, dbLatency, userEmail,
								jsutil.LogFields{"threshold_ms": 200, "latency_ms": dbLatency})
							bump()
							return nil
						})
					}),
				),
				el.Button(
					el.Class("btn error"),
					el.Text("Error"),
					el.OnClick(func(e dom.Event) {
						_ = debugutil.WithOp("logging.btn_error", func() error {
							jsutil.LogError("request #%d: query timeout after %.1fms — user %s will see stale data",
								requestID, dbLatency*3, userEmail,
								jsutil.LogFields{"timeout_ms": dbLatency * 3, "user": userEmail})
							bump()
							return nil
						})
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
						_ = debugutil.WithOp("logging.trigger_exception", func() error {
							err := processOrder(7891, 249.99)
							jsutil.Exception(err)
							bump()
							return nil
						})
					}),
				),
				el.Button(
					el.Class("btn error"),
					el.Text("Catch Panic"),
					el.OnClick(func(e dom.Event) {
						_ = debugutil.WithOp("logging.catch_panic", func() error {
							jsutil.Catch(func() {
								items := []string{"widget-A", "widget-B"}
								jsutil.LogInfo("processing %d items for order #%d", len(items), 7891)
								_ = items[5] // out-of-bounds panic
							})
							bump()
							return nil
						})
					}),
				),
			),
		),

		// Status
		el.Div(
			el.Class("counter"),
			el.DynText(func() string {
				n := clickCount()
				if n == 0 {
					return "Click a button — then check the DevTools console."
				}
				return fmt.Sprintf("%d log actions triggered — check DevTools console.", n)
			}),
		),
	)
}

func main() {
	_ = debugutil.WithOp("main.mount", func() error {
		el.Mount("#app", App)
		return nil
	})
	select {}
}
