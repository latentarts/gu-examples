//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
)

type chatMsg struct {
	Role      string
	Content   string
	Streaming bool
	Timestamp string
	Model     string
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func renderMarkdown(md string) string {
	if md == "" {
		return ""
	}
	return js.Global().Get("Markdown").Call("render", md).String()
}

func copyToClipboard(text string) {
	if text == "" {
		return
	}
	nav := js.Global().Get("navigator")
	if !nav.IsUndefined() {
		clipboard := nav.Get("clipboard")
		if !clipboard.IsUndefined() && !clipboard.Get("writeText").IsUndefined() {
			clipboard.Call("writeText", text)
			return
		}
	}
}

func App() el.Node {
	store := jsutil.LocalStorage()
	
	model, setModel := reactive.NewSignal(store.Get("webllm-model"))
	if model() == "" {
		setModel("SmolLM2-360M-Instruct-q4f32_1-MLC")
	}
	
	systemPrompt, setSystemPrompt := reactive.NewSignal(store.Get("webllm-system"))
	if systemPrompt() == "" {
		setSystemPrompt("You are a helpful assistant. Keep responses concise.")
	}

	loading, setLoading := reactive.NewSignal(false)
	progress, setProgress := reactive.NewSignal("")
	engineReady, setEngineReady := reactive.NewSignal(false)
	generating, setGenerating := reactive.NewSignal(false)
	errMsg, setErrMsg := reactive.NewSignal("")
	showSettings, setShowSettings := reactive.NewSignal(false)
	
	input, setInput := reactive.NewSignal("")
	msgVer, setMsgVer := reactive.NewSignal(0)
	var messages []chatMsg

	// Engine initialization
	initEngine := func(m string) {
		setLoading(true)
		setEngineReady(false)
		setErrMsg("")
		setProgress("Initializing...")

		go func() {
			progressCb := js.FuncOf(func(this js.Value, args []js.Value) any {
				if len(args) > 0 {
					setProgress(args[0].String())
				}
				return nil
			})
			js.Global().Get("App").Set("onProgress", progressCb)

			promise := js.Global().Get("App").Call("initEngine", m)
			_, err := jsutil.Await(promise)

			progressCb.Release()

			if err != nil {
				setErrMsg(fmt.Sprintf("Failed to load model: %v", err))
				setLoading(false)
				return
			}

			setLoading(false)
			setEngineReady(true)
			store.Set("webllm-model", m)
		}()
	}

	sendMessage := func() {
		text := input()
		if text == "" || generating() || !engineReady() {
			return
		}
		setInput("")

		ts := time.Now().Format("15:04:05")
		messages = append(messages, chatMsg{
			Role:      "user",
			Content:   text,
			Timestamp: ts,
			Model:     model(),
		})
		
		assistantIdx := len(messages)
		messages = append(messages, chatMsg{
			Role:      "assistant",
			Content:   "",
			Streaming: true,
			Timestamp: ts,
			Model:     model(),
		})
		
		setMsgVer(msgVer() + 1)
		setGenerating(true)

		go func() {
			payload := []apiMessage{
				{Role: "system", Content: systemPrompt()},
			}
			// Only include previous messages and the current user message (skip the assistant placeholder)
			for i, m := range messages {
				if i < assistantIdx && m.Content != "" {
					payload = append(payload, apiMessage{Role: m.Role, Content: m.Content})
				}
			}

			data, _ := json.Marshal(payload)
			
			onToken := js.FuncOf(func(this js.Value, args []js.Value) any {
				if len(args) > 0 {
					messages[assistantIdx].Content = args[0].String()
					setMsgVer(msgVer() + 1)
				}
				return nil
			})

			promise := js.Global().Get("App").Call("chat", string(data), onToken)
			_, err := jsutil.Await(promise)
			onToken.Release()

			if err != nil {
				setErrMsg(fmt.Sprintf("Inference failed: %v", err))
			}

			messages[assistantIdx].Streaming = false
			setMsgVer(msgVer() + 1)
			setGenerating(false)
		}()
	}

	return el.Div(
		el.Class("flex flex-col h-screen overflow-hidden bg-surface text-zinc-100"),
		el.OnMount(func(e dom.Element) {
			initEngine(model())
		}),

		// Header
		el.Header(
			el.Class("flex-shrink-0 h-14 border-b border-zinc-800/50 flex items-center justify-between px-4 bg-surface/80 backdrop-blur-md z-10"),
			el.Div(
				el.Class("flex items-center gap-3"),
				el.Div(el.Class("w-8 h-8 rounded-lg bg-cyan-600/20 flex items-center justify-center text-cyan-400 font-bold"), el.Text("gu")),
				el.Div(
					el.H1(el.Class("text-sm font-semibold"), el.Text("WebLLM Chat")),
					el.Show(func() bool { return engineReady() },
						el.P(el.Class("text-[10px] text-zinc-500"), el.DynText(model)),
					),
				),
			),
			el.Div(
				el.Class("flex items-center gap-2"),
				el.Button(
					el.Class("p-2 rounded-lg hover:bg-zinc-800 text-zinc-400 transition-colors"),
					el.OnClick(func(e dom.Event) { setShowSettings(true) }),
					el.SVGTag("svg", el.Attr("viewBox", "0 0 24 24"), el.Attr("fill", "none"), el.Attr("stroke", "currentColor"), el.Attr("stroke-width", "2"), el.Class("w-5 h-5"),
						el.SVGTag("path", el.Attr("d", "M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.38a2 2 0 0 0-.73-2.73l-.15-.1a2 2 0 0 1-1-1.72v-.51a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z")),
						el.SVGTag("circle", el.Attr("cx", "12"), el.Attr("cy", "12"), el.Attr("r", "3")),
					),
				),
			),
		),

		// Main Chat Area
		el.Main(
			el.Class("flex-1 overflow-y-auto min-h-0 flex flex-col scroll-smooth"),
			el.OnMount(func(element dom.Element) {
				reactive.CreateEffect(func() {
					_ = msgVer()
					_ = generating()
					element.SetProperty("scrollTop", element.GetProperty("scrollHeight"))
				})
			}),

			el.Show(func() bool { return len(messages) == 0 && !loading() },
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
					_ = msgVer()
					args := []any{el.Class("space-y-8")}
					for i := 0; i < len(messages); i += 2 {
						args = append(args, turnGroup(i, &messages, msgVer))
					}
					return el.Div(args...)
				}),
			),
		),

		// Loading Overlay
		el.Show(func() bool { return loading() },
			el.Div(
				el.Class("fixed inset-0 bg-surface/60 backdrop-blur-sm z-50 flex items-center justify-center p-6"),
				el.Div(
					el.Class("bg-zinc-900 border border-zinc-800 rounded-2xl p-8 max-w-sm w-full shadow-2xl space-y-6"),
					el.Div(
						el.Class("w-12 h-12 border-4 border-cyan-600/20 border-t-cyan-500 rounded-full animate-spin mx-auto"),
					),
					el.Div(
						el.Class("text-center space-y-2"),
						el.H3(el.Class("text-lg font-bold"), el.Text("Loading Model")),
						el.P(el.Class("text-xs text-zinc-500 font-mono"), el.DynText(progress)),
					),
				),
			),
		),

		// Error Message
		el.Show(func() bool { return errMsg() != "" },
			el.Div(
				el.Class("mx-4 my-2 p-3 bg-red-900/20 border border-red-900/30 rounded-xl text-red-400 text-xs flex items-center gap-3"),
				el.SVGTag("svg", el.Attr("viewBox", "0 0 24 24"), el.Attr("fill", "none"), el.Attr("stroke", "currentColor"), el.Attr("stroke-width", "2"), el.Class("w-4 h-4 flex-shrink-0"),
					el.SVGTag("circle", el.Attr("cx", "12"), el.Attr("cy", "12"), el.Attr("r", "10")),
					el.SVGTag("line", el.Attr("x1", "12"), el.Attr("y1", "8"), el.Attr("x2", "12"), el.Attr("y2", "12")),
					el.SVGTag("line", el.Attr("x1", "12"), el.Attr("y1", "16"), el.Attr("x2", "12.01"), el.Attr("y2", "16")),
				),
				el.DynText(errMsg),
				el.Button(
					el.Class("ml-auto hover:text-red-300"),
					el.OnClick(func(e dom.Event) { setErrMsg("") }),
					el.Text("Dismiss"),
				),
			),
		),

		// Input Area
		el.Footer(
			el.Class("flex-shrink-0 p-4 bg-surface border-t border-zinc-800/50"),
			el.Div(
				el.Class("max-w-3xl mx-auto"),
				el.Div(
					el.Class("relative flex items-end gap-2 bg-zinc-900 border border-zinc-800 rounded-2xl p-2 focus-within:border-cyan-600/50 transition-colors shadow-lg"),
					el.Textarea(
						el.Class("flex-1 bg-transparent border-0 focus:ring-0 text-sm py-2 px-3 resize-none min-h-[44px] max-h-40"),
						el.Placeholder("Message WebLLM..."),
						el.DynProp("value", func() any { return input() }),
						el.OnInput(func(e dom.Event) { setInput(e.TargetValue()) }),
						el.OnKeyDown(func(e dom.Event) {
							if e.Key() == "Enter" && !e.Value.Get("shiftKey").Bool() {
								e.PreventDefault()
								sendMessage()
							}
						}),
					),
					el.Button(
						el.Class("w-10 h-10 flex items-center justify-center rounded-xl bg-cyan-600 hover:bg-cyan-500 text-zinc-950 disabled:opacity-20 disabled:grayscale transition-all"),
						el.OnClick(func(e dom.Event) { sendMessage() }),
						el.DynAttr("disabled", func() string {
							if input() == "" || generating() || !engineReady() {
								return "true"
							}
							return ""
						}),
						el.SVGTag("svg", el.Attr("viewBox", "0 0 24 24"), el.Attr("fill", "none"), el.Attr("stroke", "currentColor"), el.Attr("stroke-width", "2.5"), el.Class("w-5 h-5"),
							el.SVGTag("path", el.Attr("d", "M22 2L11 13")),
							el.SVGTag("path", el.Attr("d", "M22 2L15 22L11 13L2 9L22 2Z")),
						),
					),
				),
				el.P(el.Class("text-[10px] text-zinc-600 text-center mt-3"),
					el.Text("WebLLM runs SmolLM2, Llama 3.2 or Phi 3.5 locally using WebGPU."),
				),
			),
		),

		// Settings Modal
		el.Show(func() bool { return showSettings() },
			el.Div(
				el.Class("fixed inset-0 z-[60] flex items-center justify-center p-4"),
				el.Div(el.Class("absolute inset-0 bg-surface/80 backdrop-blur-md"), el.OnClick(func(e dom.Event) { setShowSettings(false) })),
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
									setModel(m)
									setShowSettings(false)
									initEngine(m)
								}),
								el.Option(el.Attr("value", "SmolLM2-360M-Instruct-q4f32_1-MLC"), el.Text("SmolLM2 360M (Fastest)"), el.DynAttr("selected", func() string { if model() == "SmolLM2-360M-Instruct-q4f32_1-MLC" { return "true" }; return "" })),
								el.Option(el.Attr("value", "Llama-3.2-1B-Instruct-q4f32_1-MLC"), el.Text("Llama 3.2 1B (Balanced)"), el.DynAttr("selected", func() string { if model() == "Llama-3.2-1B-Instruct-q4f32_1-MLC" { return "true" }; return "" })),
								el.Option(el.Attr("value", "Phi-3.5-mini-instruct-q4f32_1-MLC"), el.Text("Phi 3.5 3.8B (Smartest)"), el.DynAttr("selected", func() string { if model() == "Phi-3.5-mini-instruct-q4f32_1-MLC" { return "true" }; return "" })),
							),
						),

						el.Div(
							el.Class("space-y-2"),
							el.Label(el.Class("text-xs font-medium text-zinc-400"), el.Text("System Prompt")),
							el.Textarea(
								el.Class("w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-cyan-600 min-h-[100px] resize-none"),
								el.DynProp("value", func() any { return systemPrompt() }),
								el.OnInput(func(e dom.Event) {
									setSystemPrompt(e.TargetValue())
									store.Set("webllm-system", e.TargetValue())
								}),
							),
						),

						el.Button(
							el.Class("w-full py-3 bg-zinc-800 hover:bg-zinc-700 rounded-xl text-sm font-medium transition-colors"),
							el.Text("Close"),
							el.OnClick(func(e dom.Event) { setShowSettings(false) }),
						),
					),
				),
			),
		),
	)
}

func turnGroup(userIdx int, all *[]chatMsg, ver func() int) el.Node {
	return el.Div(
		el.Class("flex flex-col gap-6 w-full"),
		messageBubble((*all)[userIdx], userIdx, all, ver),
		el.Show(func() bool {
			_ = ver()
			return userIdx+1 < len(*all)
		}, el.Dynamic(func() el.Node {
			return messageBubble((*all)[userIdx+1], userIdx+1, all, ver)
		})),
	)
}

func messageBubble(item chatMsg, i int, all *[]chatMsg, ver func() int) el.Node {
	copied, setCopied := reactive.NewSignal(false)

	if item.Role == "user" {
		return el.Div(
			el.Class("flex w-full justify-end gap-3"),
			el.Div(
				el.Class("max-w-[85%] rounded-2xl rounded-tr-md px-4 py-2 bg-zinc-800 text-zinc-100 text-sm leading-relaxed shadow-sm"),
				el.DynText(func() string {
					_ = ver()
					if i >= len(*all) { return "" }
					return (*all)[i].Content
				}),
			),
		)
	}

	return el.Div(
		el.Class("flex w-full gap-4 items-start group"),
		el.Div(el.Class("w-8 h-8 rounded-lg bg-cyan-600/20 flex items-center justify-center text-cyan-400 flex-shrink-0 mt-1"),
			el.SVGTag("svg", el.Attr("viewBox", "0 0 24 24"), el.Attr("fill", "none"), el.Attr("stroke", "currentColor"), el.Attr("stroke-width", "2"), el.Class("w-5 h-5"),
				el.SVGTag("path", el.Attr("d", "M12 2L2 7L12 12L22 7L12 2Z")),
				el.SVGTag("path", el.Attr("d", "M2 17L12 22L22 17")),
				el.SVGTag("path", el.Attr("d", "M2 12L12 17L22 12")),
			),
		),
		el.Div(
			el.Class("flex-1 min-w-0 space-y-2"),
			el.Div(el.Class("text-[10px] font-bold text-cyan-500 uppercase tracking-widest"), el.Text("Assistant")),
			el.Div(
				el.Class("prose-chat text-zinc-300 w-full"),
				markdownBlock(func() string {
					_ = ver()
					if i >= len(*all) { return "" }
					return (*all)[i].Content
				}),
				el.Show(func() bool {
					_ = ver()
					return i < len(*all) && (*all)[i].Streaming && (*all)[i].Content == ""
				}, el.Div(el.Class("flex gap-1 py-2"),
					el.Span(el.Class("w-1.5 h-1.5 bg-cyan-600/50 rounded-full thinking-dot")),
					el.Span(el.Class("w-1.5 h-1.5 bg-cyan-600/50 rounded-full thinking-dot")),
					el.Span(el.Class("w-1.5 h-1.5 bg-cyan-600/50 rounded-full thinking-dot")),
				)),
			),
			el.Div(
				el.Class("flex items-center gap-4 mt-4 opacity-0 group-hover:opacity-100 transition-opacity"),
				el.Button(
					el.Class("text-[10px] text-zinc-500 hover:text-cyan-400 flex items-center gap-1.5"),
					el.OnClick(func(e dom.Event) {
						_ = ver()
						if i < len(*all) {
							copyToClipboard((*all)[i].Content)
							setCopied(true)
							time.AfterFunc(2*time.Second, func() { setCopied(false) })
						}
					}),
					el.SVGTag("svg", el.Attr("viewBox", "0 0 24 24"), el.Attr("fill", "none"), el.Attr("stroke", "currentColor"), el.Attr("stroke-width", "2"), el.Class("w-3 h-3"),
						el.SVGTag("rect", el.Attr("x", "9"), el.Attr("y", "9"), el.Attr("width", "13"), el.Attr("height", "13"), el.Attr("rx", "2"), el.Attr("ry", "2")),
						el.SVGTag("path", el.Attr("d", "M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1")),
					),
					el.DynText(func() string {
						if copied() { return "Copied!" }
						return "Copy"
					}),
				),
			),
		),
	)
}

func markdownBlock(markdownGetter func() string) el.Node {
	return el.Div(
		el.OnMount(func(element dom.Element) {
			reactive.CreateEffect(func() {
				md := markdownGetter()
				element.SetInnerHTML(renderMarkdown(md))
			})
		}),
	)
}

func main() {
	el.Mount("#app", App)
	select {}
}
