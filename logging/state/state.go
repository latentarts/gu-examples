package state

import (
	"fmt"
	"strconv"

	"github.com/latentart/gu/debugutil"
	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
)

var LevelNames = []string{"Debug", "Info", "Warning", "Error", "Off"}

// LoggingState manages the reactive state of the logging application.
type LoggingState struct {
	ClickCount    func() int
	SetClickCount func(int)

	CurrentLevel    func() int
	SetCurrentLevel func(int)

	RequestID   int
	UserEmail   string
	DbLatency   float64
	CacheHits   int
	CacheMisses int
}

// NewLoggingState initializes a new logging state.
func NewLoggingState() *LoggingState {
	clickCount, setClickCount := reactive.NewSignal(0)
	currentLevel, setCurrentLevel := reactive.NewSignal(int(jsutil.GetLogLevel()))

	return &LoggingState{
		ClickCount:    clickCount,
		SetClickCount: setClickCount,

		CurrentLevel:    currentLevel,
		SetCurrentLevel: setCurrentLevel,

		RequestID:   1042,
		UserEmail:   "alice@example.com",
		DbLatency:   230.5,
		CacheHits:   847,
		CacheMisses: 53,
	}
}

// Bump increments the click count.
func (s *LoggingState) Bump() {
	s.SetClickCount(s.ClickCount() + 1)
}

// SetLevel updates the current log level.
func (s *LoggingState) SetLevel(levelStr string) {
	_ = debugutil.WithOp("logging.set_level", func() error {
		v, _ := strconv.Atoi(levelStr)
		jsutil.SetLogLevel(jsutil.LogLevel(v))
		s.SetCurrentLevel(v)
		jsutil.LogInfo("log level changed to %s", LevelNames[v])
		return nil
	})
}

// LogDebug triggers a debug log message.
func (s *LoggingState) LogDebug() {
	_ = debugutil.WithOp("logging.btn_debug", func() error {
		jsutil.LogDebug("cache stats: %d hits / %d misses (%.1f%% hit rate)",
			s.CacheHits, s.CacheMisses, float64(s.CacheHits)/float64(s.CacheHits+s.CacheMisses)*100,
			jsutil.LogFields{"hits": s.CacheHits, "misses": s.CacheMisses})
		s.Bump()
		return nil
	})
}

// LogInfo triggers an info log message.
func (s *LoggingState) LogInfo() {
	_ = debugutil.WithOp("logging.btn_info", func() error {
		jsutil.LogInfo("request #%d: user %s authenticated, db latency %.1fms",
			s.RequestID, s.UserEmail, s.DbLatency,
			jsutil.LogFields{
				"request_id": s.RequestID,
				"user":       s.UserEmail,
				"db_ms":      s.DbLatency,
			})
		s.RequestID++
		s.Bump()
		return nil
	})
}

// LogWarn triggers a warning log message.
func (s *LoggingState) LogWarn() {
	_ = debugutil.WithOp("logging.btn_warn", func() error {
		jsutil.LogWarn("request #%d: db latency %.1fms exceeds 200ms threshold for user %s",
			s.RequestID, s.DbLatency, s.UserEmail,
			jsutil.LogFields{"threshold_ms": 200, "latency_ms": s.DbLatency})
		s.Bump()
		return nil
	})
}

// LogError triggers an error log message.
func (s *LoggingState) LogError() {
	_ = debugutil.WithOp("logging.btn_error", func() error {
		jsutil.LogError("request #%d: query timeout after %.1fms — user %s will see stale data",
			s.RequestID, s.DbLatency*3, s.UserEmail,
			jsutil.LogFields{"timeout_ms": s.DbLatency * 3, "user": s.UserEmail})
		s.Bump()
		return nil
	})
}

// TriggerException triggers a multi-frame call chain that fails.
func (s *LoggingState) TriggerException() {
	_ = debugutil.WithOp("logging.trigger_exception", func() error {
		err := processOrder(7891, 249.99)
		jsutil.Exception(err)
		s.Bump()
		return nil
	})
}

// CatchPanic catches a simulated panic.
func (s *LoggingState) CatchPanic() {
	_ = debugutil.WithOp("logging.catch_panic", func() error {
		jsutil.Catch(func() {
			items := []string{"widget-A", "widget-B"}
			jsutil.LogInfo("processing %d items for order #%d", len(items), 7891)
			_ = items[5] // out-of-bounds panic
		})
		s.Bump()
		return nil
	})
}

// Helper functions for Exception simulation.

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
