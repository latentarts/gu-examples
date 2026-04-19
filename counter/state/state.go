package state

import (
	"github.com/latentart/gu/reactive"
)

// CounterState manages the reactive state of the counter application.
type CounterState struct {
	Count    func() int
	SetCount func(int)

	IsDark    func() bool
	SetIsDark func(bool)

	Doubled func() int
}

// NewCounterState initializes a new counter state.
func NewCounterState() *CounterState {
	count, setCount := reactive.NewSignal(0)
	isDark, setIsDark := reactive.NewSignal(false)

	doubled := reactive.CreateMemo(func() int {
		return count() * 2
	})

	return &CounterState{
		Count:     count,
		SetCount:  setCount,
		IsDark:    isDark,
		SetIsDark: setIsDark,
		Doubled:   doubled,
	}
}
