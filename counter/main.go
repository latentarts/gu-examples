//go:build js && wasm

package main

import (
	"fmt"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
	"github.com/latentart/gu/theme"
)

func App() el.Node {
	jsutil.LogInfo("counter app mounted")
	count, setCount := reactive.NewSignal(0)
	isDark, setIsDark := reactive.NewSignal(false)

	doubled := reactive.CreateMemo(func() int {
		return count() * 2
	})

	return el.Div(
		el.Class("app"),
		el.H1(el.Text("gu Counter")),

		el.P(
			el.Text("Count: "),
			el.DynText(func() string {
				return fmt.Sprintf("%d", count())
			}),
		),

		el.P(
			el.Text("Doubled: "),
			el.DynText(func() string {
				return fmt.Sprintf("%d", doubled())
			}),
		),

		el.Div(
			el.Class("buttons"),
			el.Button(
				el.Text("-"),
				el.OnClick(func(e dom.Event) {
					setCount(count() - 1)
					jsutil.LogDebug("count decremented to %d", count())
				}),
			),
			el.Button(
				el.Text("+"),
				el.OnClick(func(e dom.Event) {
					setCount(count() + 1)
					jsutil.LogDebug("count incremented to %d", count())
				}),
			),
			el.Button(
				el.Text("Reset"),
				el.OnClick(func(e dom.Event) {
					setCount(0)
					jsutil.LogDebug("count reset")
				}),
			),
		),

		el.Button(
			el.DynText(func() string {
				if isDark() {
					return "Switch to Light"
				}
				return "Switch to Dark"
			}),
			el.OnClick(func(e dom.Event) {
				dark := !isDark()
				setIsDark(dark)
				if dark {
					theme.SetTheme(theme.DefaultDark())
					jsutil.LogInfo("switched to dark theme")
				} else {
					theme.SetTheme(theme.DefaultLight())
					jsutil.LogInfo("switched to light theme")
				}
			}),
			el.Class("theme-toggle"),
		),

		el.Show(
			func() bool { return count() > 10 },
			el.P(
				el.Text("Count is over 10!"),
				el.Style("color", "var(--gu-color-success)"),
				el.Style("font-weight", "bold"),
			),
		),
	)
}

func main() {
	theme.SetTheme(theme.DefaultLight())
	el.Mount("#app", App)
	select {}
}
