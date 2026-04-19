//go:build js && wasm

package main

import (
	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
)

func App() el.Node {
	jsutil.LogInfo("tailwind showcase mounted")
	isDark, setIsDark := reactive.NewSignal(false)

	return el.Div(
		el.DynClass(func() string {
			if isDark() {
				return "dark min-h-screen bg-gray-950 text-gray-100 transition-colors duration-300"
			}
			return "min-h-screen bg-gray-50 text-gray-900 transition-colors duration-300"
		}),

		// Header
		el.Div(
			el.Class("max-w-6xl mx-auto px-6 py-8"),
			el.Div(
				el.Class("flex items-center justify-between mb-8"),
				el.Div(
					el.H1(el.Class("text-3xl font-bold text-brand"), el.Text("gu + Tailwind")),
					el.P(el.Class("text-sm text-gray-500 dark:text-gray-400 mt-1"), el.Text("Component showcase built with Go WASM")),
				),
				el.Button(
					el.Class("px-4 py-2 rounded-lg text-sm font-medium bg-gray-200 dark:bg-gray-800 hover:bg-gray-300 dark:hover:bg-gray-700 transition-colors"),
					el.DynText(func() string {
						if isDark() {
							return "Light Mode"
						}
						return "Dark Mode"
					}),
					el.OnClick(func(e dom.Event) {
						dark := !isDark()
						setIsDark(dark)
						if dark {
							jsutil.LogDebug("switched to dark mode")
						} else {
							jsutil.LogDebug("switched to light mode")
						}
					}),
				),
			),

			// Grid
			el.Div(
				el.Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"),
				HeroCard(),
				ProfileCard(),
				PricingCard(),
				StatsCard(),
				TestimonialCard(),
				CTACard(),
			),
		),
	)
}

func HeroCard() el.Node {
	return el.Div(
		el.Class("md:col-span-2 lg:col-span-2 rounded-2xl bg-gradient-to-br from-brand to-brand-dark p-8 text-white shadow-lg"),
		el.Div(
			el.Class("max-w-lg"),
			el.Span(el.Class("inline-block px-3 py-1 text-xs font-semibold bg-white/20 rounded-full mb-4"), el.Text("New Release")),
			el.H2(el.Class("text-3xl font-bold mb-3"), el.Text("Build reactive UIs in Go")),
			el.P(el.Class("text-indigo-100 mb-6 leading-relaxed"), el.Text("gu brings SolidJS-style fine-grained reactivity to Go WASM. No virtual DOM, no framework overhead — just signals, effects, and direct DOM updates.")),
			el.Button(el.Class("px-6 py-2.5 bg-white text-brand-dark font-semibold rounded-lg hover:bg-indigo-50 transition-colors"), el.Text("Get Started")),
		),
	)
}

func ProfileCard() el.Node {
	return el.Div(
		el.Class("rounded-2xl bg-white dark:bg-gray-900 p-6 shadow-lg border border-gray-100 dark:border-gray-800 text-center"),
		el.Div(
			el.Class("w-20 h-20 rounded-full bg-gradient-to-br from-amber-400 to-orange-500 mx-auto mb-4 flex items-center justify-center text-2xl font-bold text-white"),
			el.Text("JD"),
		),
		el.H3(el.Class("text-lg font-semibold"), el.Text("Jane Doe")),
		el.P(el.Class("text-sm text-gray-500 dark:text-gray-400 mb-4"), el.Text("Senior Go Engineer")),
		el.Div(
			el.Class("flex justify-center gap-6 text-sm"),
			statItem("Posts", "128"),
			statItem("Followers", "4.2k"),
			statItem("Following", "312"),
		),
	)
}

func statItem(label, value string) el.Node {
	return el.Div(
		el.Span(el.Class("block font-bold text-gray-900 dark:text-gray-100"), el.Text(value)),
		el.Span(el.Class("text-gray-500 dark:text-gray-400"), el.Text(label)),
	)
}

func PricingCard() el.Node {
	return el.Div(
		el.Class("rounded-2xl bg-white dark:bg-gray-900 p-6 shadow-lg border border-gray-100 dark:border-gray-800"),
		el.Span(el.Class("inline-block px-3 py-1 text-xs font-semibold bg-brand/10 text-brand rounded-full mb-3"), el.Text("Popular")),
		el.H3(el.Class("text-lg font-semibold mb-1"), el.Text("Pro Plan")),
		el.Div(
			el.Class("mb-4"),
			el.Span(el.Class("text-4xl font-bold"), el.Text("$29")),
			el.Span(el.Class("text-gray-500 dark:text-gray-400"), el.Text("/month")),
		),
		el.Ul(
			el.Class("space-y-2 mb-6 text-sm text-gray-600 dark:text-gray-300"),
			featureItem("Unlimited projects"),
			featureItem("Priority support"),
			featureItem("Advanced analytics"),
			featureItem("Custom integrations"),
		),
		el.Button(el.Class("w-full py-2.5 bg-brand text-white font-semibold rounded-lg hover:bg-brand-dark transition-colors"), el.Text("Subscribe")),
	)
}

func featureItem(text string) el.Node {
	return el.Li(
		el.Class("flex items-center gap-2"),
		el.Span(el.Class("text-green-500 font-bold"), el.Text("+")),
		el.Text(text),
	)
}

func StatsCard() el.Node {
	return el.Div(
		el.Class("rounded-2xl bg-white dark:bg-gray-900 p-6 shadow-lg border border-gray-100 dark:border-gray-800"),
		el.H3(el.Class("text-lg font-semibold mb-4"), el.Text("This Month")),
		el.Div(
			el.Class("grid grid-cols-2 gap-4"),
			statBlock("Revenue", "$12.4k", "text-green-500"),
			statBlock("Users", "1,429", "text-brand"),
			statBlock("Orders", "384", "text-amber-500"),
			statBlock("Growth", "+22%", "text-emerald-500"),
		),
	)
}

func statBlock(label, value, colorClass string) el.Node {
	return el.Div(
		el.Class("text-center p-3 rounded-xl bg-gray-50 dark:bg-gray-800"),
		el.Div(el.Class("text-2xl font-bold "+colorClass), el.Text(value)),
		el.Div(el.Class("text-xs text-gray-500 dark:text-gray-400 mt-1"), el.Text(label)),
	)
}

func TestimonialCard() el.Node {
	return el.Div(
		el.Class("rounded-2xl bg-white dark:bg-gray-900 p-6 shadow-lg border border-gray-100 dark:border-gray-800"),
		el.Div(
			el.Class("text-4xl text-brand/30 mb-2 leading-none"),
			el.Text("\u201C"),
		),
		el.P(
			el.Class("text-gray-600 dark:text-gray-300 mb-4 italic leading-relaxed"),
			el.Text("gu completely changed how I think about building web UIs with Go. The reactive model is incredibly intuitive."),
		),
		el.Div(
			el.Class("flex items-center gap-3"),
			el.Div(
				el.Class("w-10 h-10 rounded-full bg-gradient-to-br from-sky-400 to-blue-500 flex items-center justify-center text-sm font-bold text-white"),
				el.Text("AK"),
			),
			el.Div(
				el.Div(el.Class("font-semibold text-sm"), el.Text("Alex Kim")),
				el.Div(el.Class("text-xs text-gray-500 dark:text-gray-400"), el.Text("CTO at Stackworks")),
			),
		),
	)
}

func CTACard() el.Node {
	return el.Div(
		el.Class("rounded-2xl bg-gradient-to-br from-amber-400 to-orange-500 p-6 shadow-lg text-white"),
		el.H3(el.Class("text-xl font-bold mb-2"), el.Text("Ready to ship?")),
		el.P(el.Class("text-amber-50 mb-4 text-sm leading-relaxed"), el.Text("Start building production-grade Go WASM apps today. Zero JavaScript required.")),
		el.Div(
			el.Class("flex gap-3"),
			el.Button(el.Class("px-5 py-2 bg-white text-orange-600 font-semibold rounded-lg hover:bg-orange-50 transition-colors text-sm"), el.Text("Documentation")),
			el.Button(el.Class("px-5 py-2 bg-white/20 font-semibold rounded-lg hover:bg-white/30 transition-colors text-sm"), el.Text("GitHub")),
		),
	)
}

func main() {
	el.Mount("#app", App)
	select {}
}
