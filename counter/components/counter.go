package components

import (
	"fmt"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/theme"
	"github.com/latentarts/gu-examples/counter/state"
)

// Display renders the current count and its doubled value.
func Display(s *state.CounterState) el.Node {
	return el.Div(
		el.P(
			el.Text("Count: "),
			el.DynText(func() string {
				return fmt.Sprintf("%d", s.Count())
			}),
		),
		el.P(
			el.Text("Doubled: "),
			el.DynText(func() string {
				return fmt.Sprintf("%d", s.Doubled())
			}),
		),
	)
}

// Controls renders the buttons to increment, decrement, and reset the counter.
func Controls(s *state.CounterState) el.Node {
	return el.Div(
		el.Class("buttons"),
		el.Button(
			el.Text("-"),
			el.OnClick(func(e dom.Event) {
				s.SetCount(s.Count() - 1)
				jsutil.LogDebug("count decremented to %d", s.Count())
			}),
		),
		el.Button(
			el.Text("+"),
			el.OnClick(func(e dom.Event) {
				s.SetCount(s.Count() + 1)
				jsutil.LogDebug("count incremented to %d", s.Count())
			}),
		),
		el.Button(
			el.Text("Reset"),
			el.OnClick(func(e dom.Event) {
				s.SetCount(0)
				jsutil.LogDebug("count reset")
			}),
		),
	)
}

// ThemeToggle renders a button to switch between light and dark themes.
func ThemeToggle(s *state.CounterState) el.Node {
	return el.Button(
		el.DynText(func() string {
			if s.IsDark() {
				return "Switch to Light"
			}
			return "Switch to Dark"
		}),
		el.OnClick(func(e dom.Event) {
			dark := !s.IsDark()
			s.SetIsDark(dark)
			if dark {
				theme.SetTheme(theme.DefaultDark())
				jsutil.LogInfo("switched to dark theme")
			} else {
				theme.SetTheme(theme.DefaultLight())
				jsutil.LogInfo("switched to light theme")
			}
		}),
		el.Class("theme-toggle"),
	)
}
