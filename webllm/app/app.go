package app

import (
	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/webllm/components"
	"github.com/latentarts/gu-examples/webllm/state"
)

func App(styles el.Node) el.Node {
	s := state.NewChatState()

	return el.Div(
		styles,
		el.Class("flex flex-col h-screen overflow-hidden bg-surface text-zinc-100"),
		el.OnMount(func(e dom.Element) {
			s.InitEngine(s.Model())
		}),
		el.Header(
			el.Class("flex-shrink-0 h-14 border-b border-zinc-800/50 flex items-center justify-between px-4 bg-surface/80 backdrop-blur-md z-10"),
			el.Div(
				el.Class("flex items-center gap-3"),
				el.Div(el.Class("w-8 h-8 rounded-lg bg-cyan-600/20 flex items-center justify-center text-cyan-400 font-bold"), el.Text("gu")),
				el.Div(
					el.H1(el.Class("text-sm font-semibold"), el.Text("WebLLM Chat")),
					el.Show(func() bool { return s.EngineReady() },
						el.P(el.Class("text-[10px] text-zinc-500"), el.DynText(s.Model)),
					),
				),
			),
			el.Div(
				el.Class("flex items-center gap-2"),
				el.Button(
					el.Class("p-2 rounded-lg hover:bg-zinc-800 text-zinc-400 transition-colors"),
					el.OnClick(func(e dom.Event) { s.SetShowSettings(true) }),
					el.SVGTag("svg", el.Attr("viewBox", "0 0 24 24"), el.Attr("fill", "none"), el.Attr("stroke", "currentColor"), el.Attr("stroke-width", "2"), el.Class("w-5 h-5"),
						el.SVGTag("path", el.Attr("d", "M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.38a2 2 0 0 0-.73-2.73l-.15-.1a2 2 0 0 1-1-1.72v-.51a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z")),
						el.SVGTag("circle", el.Attr("cx", "12"), el.Attr("cy", "12"), el.Attr("r", "3")),
					),
				),
			),
		),
		el.Main(
			el.Class("flex-1 overflow-y-auto min-h-0 flex flex-col scroll-smooth"),
			el.OnMount(func(element dom.Element) {
				reactive.CreateEffect(func() {
					_ = s.MsgVer()
					_ = s.Generating()
					element.SetProperty("scrollTop", element.GetProperty("scrollHeight"))
				})
			}),
			el.Show(func() bool { return len(s.Messages) == 0 && !s.Loading() },
				el.Div(
					el.Class("flex-1 flex flex-col items-center justify-center p-8 text-center space-y-4"),
					el.Div(el.Class("w-16 h-16 rounded-2xl bg-cyan-600/10 flex items-center justify-center text-cyan-400 text-3xl"), el.Text("gu")),
					el.Div(
						el.H2(el.Class("text-xl font-bold"), el.Text("How can I help you today?")),
						el.P(el.Class("text-zinc-500 text-sm mt-2 max-w-sm"), el.Text("WebLLM runs entirely in your browser. All your data stays private and local.")),
					),
				),
			),
			el.Div(
				el.Class("max-w-3xl mx-auto w-full px-4 py-8 space-y-8"),
				el.Dynamic(func() el.Node {
					_ = s.MsgVer()
					args := []any{el.Class("space-y-8")}
					for i := 0; i < len(s.Messages); i += 2 {
						args = append(args, components.TurnGroup(s, i))
					}
					return el.Div(args...)
				}),
			),
		),
		el.Show(func() bool { return s.Loading() },
			el.Div(
				el.Class("fixed inset-0 bg-surface/60 backdrop-blur-sm z-50 flex items-center justify-center p-6"),
				el.Div(
					el.Class("bg-zinc-900 border border-zinc-800 rounded-2xl p-8 max-w-sm w-full shadow-2xl space-y-6"),
					el.Div(el.Class("w-12 h-12 border-4 border-cyan-600/20 border-t-cyan-500 rounded-full animate-spin mx-auto")),
					el.Div(
						el.Class("text-center space-y-2"),
						el.H3(el.Class("text-lg font-bold"), el.Text("Loading Model")),
						el.P(el.Class("text-xs text-zinc-500 font-mono"), el.DynText(s.Progress)),
					),
				),
			),
		),
		el.Show(func() bool { return s.ErrMsg() != "" },
			el.Div(
				el.Class("mx-4 my-2 p-3 bg-red-900/20 border border-red-900/30 rounded-xl text-red-400 text-xs flex items-center gap-3"),
				el.DynText(s.ErrMsg),
				el.Button(el.Class("ml-auto hover:text-red-300"), el.OnClick(func(e dom.Event) { s.SetErrMsg("") }), el.Text("Dismiss")),
			),
		),
		el.Footer(
			el.Class("flex-shrink-0 p-4 bg-surface border-t border-zinc-800/50"),
			el.Div(
				el.Class("max-w-3xl mx-auto"),
				el.Div(
					el.Class("relative flex items-end gap-2 bg-zinc-900 border border-zinc-800 rounded-2xl p-2 focus-within:border-cyan-600/50 transition-colors shadow-lg"),
					el.Textarea(
						el.Class("flex-1 bg-transparent border-0 focus:ring-0 text-sm py-2 px-3 resize-none min-h-[44px] max-h-40"),
						el.Placeholder("Message WebLLM..."),
						el.DynProp("value", func() any { return s.Input() }),
						el.OnInput(func(e dom.Event) { s.SetInput(e.TargetValue()) }),
						el.OnKeyDown(func(e dom.Event) {
							if e.Key() == "Enter" && !e.Value.Get("shiftKey").Bool() {
								e.PreventDefault()
								s.SendMessage()
							}
						}),
					),
					el.Button(
						el.Class("w-10 h-10 flex items-center justify-center rounded-xl bg-cyan-600 hover:bg-cyan-500 text-zinc-950 disabled:opacity-20 disabled:grayscale transition-all"),
						el.OnClick(func(e dom.Event) { s.SendMessage() }),
						el.DynAttr("disabled", func() string {
							if s.Input() == "" || s.Generating() || !s.EngineReady() {
								return "true"
							}
							return ""
						}),
						el.Text("→"),
					),
				),
				el.P(el.Class("text-[10px] text-zinc-600 text-center mt-3"), el.Text("WebLLM runs SmolLM2, Llama 3.2 or Phi 3.5 locally using WebGPU.")),
			),
		),
		el.Show(func() bool { return s.ShowSettings() },
			el.Div(
				el.Class("fixed inset-0 z-[60] flex items-center justify-center p-4"),
				el.Div(el.Class("absolute inset-0 bg-surface/80 backdrop-blur-md"), el.OnClick(func(e dom.Event) { s.SetShowSettings(false) })),
				el.Div(
					el.Class("relative bg-zinc-900 border border-zinc-800 rounded-2xl w-full max-w-md shadow-2xl overflow-hidden"),
					el.Div(
						el.Class("p-6 space-y-6"),
						el.Div(
							el.H3(el.Class("text-lg font-bold"), el.Text("Settings")),
							el.P(el.Class("text-xs text-zinc-500"), el.Text("Configure your local WebLLM engine")),
						),
						el.Div(
							el.Class("space-y-2"),
							el.Label(el.Class("text-xs font-medium text-zinc-400"), el.Text("Model Selection")),
							el.Select(
								el.Class("w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-600"),
								el.OnInput(func(e dom.Event) {
									m := e.TargetValue()
									s.SetModel(m)
									s.SetShowSettings(false)
									s.InitEngine(m)
								}),
								el.Option(el.Attr("value", "SmolLM2-360M-Instruct-q4f32_1-MLC"), el.Text("SmolLM2 360M (Fastest)"), el.DynAttr("selected", func() string { if s.Model() == "SmolLM2-360M-Instruct-q4f32_1-MLC" { return "true" }; return "" })),
								el.Option(el.Attr("value", "Llama-3.2-1B-Instruct-q4f32_1-MLC"), el.Text("Llama 3.2 1B (Balanced)"), el.DynAttr("selected", func() string { if s.Model() == "Llama-3.2-1B-Instruct-q4f32_1-MLC" { return "true" }; return "" })),
								el.Option(el.Attr("value", "Phi-3.5-mini-instruct-q4f32_1-MLC"), el.Text("Phi 3.5 3.8B (Smartest)"), el.DynAttr("selected", func() string { if s.Model() == "Phi-3.5-mini-instruct-q4f32_1-MLC" { return "true" }; return "" })),
							),
						),
						el.Div(
							el.Class("space-y-2"),
							el.Label(el.Class("text-xs font-medium text-zinc-400"), el.Text("System Prompt")),
							el.Textarea(
								el.Class("w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-600 min-h-[100px] resize-none"),
								el.DynProp("value", func() any { return s.SystemPrompt() }),
								el.OnInput(func(e dom.Event) {
									s.SetSystemPrompt(e.TargetValue())
									s.Store.Set("webllm-system", e.TargetValue())
								}),
							),
						),
						el.Button(el.Class("w-full py-3 bg-zinc-800 hover:bg-zinc-700 rounded-xl text-sm font-medium transition-colors"), el.Text("Close"), el.OnClick(func(e dom.Event) { s.SetShowSettings(false) })),
					),
				),
			),
		),
	)
}
