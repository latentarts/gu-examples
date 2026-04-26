//go:build js && wasm

package components

import (
	"strings"
	"syscall/js"
	"time"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/openai-chat/state"
)

func RenderMarkdown(md string) string {
	if md == "" {
		return ""
	}
	v := js.Global().Get("Markdown").Call("render", md)
	return v.String()
}

func CopyToClipboard(text string) {
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
	document := js.Global().Get("document")
	textArea := document.Call("createElement", "textarea")
	textArea.Set("value", text)
	textArea.Get("style").Set("position", "fixed")
	textArea.Get("style").Set("left", "-9999px")
	textArea.Get("style").Set("top", "0")
	document.Get("body").Call("appendChild", textArea)
	textArea.Call("focus")
	textArea.Call("select")
	document.Call("execCommand", "copy")
	document.Get("body").Call("removeChild", textArea)
}

func TurnGroup(userIdx int, all *[]state.ChatMsg, ver func() int) el.Node {
	if userIdx < 0 || userIdx >= len(*all) || (*all)[userIdx].Role != "user" {
		return el.Div()
	}
	return el.Div(
		el.Class("flex flex-col gap-3 w-full min-w-0"),
		MessageBubble((*all)[userIdx], userIdx, all, ver),
		el.Show(func() bool {
			_ = ver()
			return userIdx+1 < len(*all) && (*all)[userIdx+1].Role == "assistant"
		}, el.Dynamic(func() el.Node {
			return MessageBubble((*all)[userIdx+1], userIdx+1, all, ver)
		})),
	)
}

func MessageBubble(item state.ChatMsg, i int, all *[]state.ChatMsg, ver func() int) el.Node {
	copied, setCopied := reactive.NewSignal(false)

	if item.Role == "user" {
		return el.Div(
			el.Class("flex w-full min-w-0 justify-end gap-3 pt-1"),
			el.Div(
				el.Class("max-w-[88%] min-w-[80px] rounded-2xl rounded-tr-md px-4 py-2 bg-zinc-800/90 text-zinc-100 text-sm leading-relaxed relative"),
				el.Div(
					el.Class("pb-5"), // Space for timestamp
					el.DynText(func() string {
						_ = ver()
						if i >= len(*all) {
							return ""
						}
						return (*all)[i].Content
					}),
				),
				el.Div(
					el.Class("absolute bottom-1.5 right-3 text-[10px] text-zinc-500 font-medium"),
					el.DynText(func() string {
						_ = ver()
						if i >= len(*all) {
							return ""
						}
						return (*all)[i].Timestamp
					}),
				),
			),
			el.Div(
				el.Class("w-8 h-8 shrink-0 rounded-lg bg-gradient-to-br from-violet-600/25 to-zinc-800/80 flex items-center justify-center text-[10px] font-bold text-violet-400"),
				el.Text("ME"),
			),
		)
	}

	return el.Div(
		el.Class("flex w-full min-w-0 gap-3 pt-1"),
		el.Div(
			el.Class("w-8 h-8 shrink-0 rounded-lg bg-gradient-to-br from-cyan-600/25 to-zinc-800/80 flex items-center justify-center text-[10px] font-bold text-cyan-400"),
			el.Text("AI"),
		),
		el.Div(
			el.Class("flex-1 min-w-0 space-y-3"),
			el.Show(
				func() bool {
					_ = ver()
					if i >= len(*all) {
						return false
					}
					return strings.TrimSpace((*all)[i].Thinking) != ""
				},
				el.Tag("details",
					el.Class("group"),
					el.Tag("summary",
						el.Class("cursor-pointer list-none flex w-full items-center justify-between gap-3 py-1.5 text-left [&::-webkit-details-marker]:hidden"),
						el.Div(
							el.Class("flex min-w-0 items-center gap-2"),
							el.Span(
								el.Class("text-[11px] font-medium text-zinc-500 uppercase tracking-wide flex items-center gap-1.5"),
								el.DynText(func() string {
									_ = ver()
									if i >= len(*all) {
										return ""
									}
									m := (*all)[i]
									active := m.Streaming && strings.TrimSpace(m.Content) == ""
									if active {
										return "Thinking"
									}
									return "Thoughts"
								}),
								el.Show(func() bool {
									_ = ver()
									if i >= len(*all) {
										return false
									}
									m := (*all)[i]
									return !m.Streaming || strings.TrimSpace(m.Content) != ""
								}, el.SVGTag("svg",
									el.Attr("viewBox", "0 0 24 24"),
									el.Attr("fill", "none"),
									el.Attr("stroke", "currentColor"),
									el.Attr("stroke-width", "2"),
									el.Class("w-3 h-3 text-zinc-500"),
									el.SVGTag("path", el.Attr("d", "M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96.44 2.5 2.5 0 0 1-2.96-3.08 3 3 0 0 1-.34-5.58 2.5 2.5 0 0 1 1.32-4.24 2.5 2.5 0 0 1 4.44-2.04z")),
									el.SVGTag("path", el.Attr("d", "M14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96.44 2.5 2.5 0 0 0 2.96-3.08 3 3 0 0 0 .34-5.58 2.5 2.5 0 0 0-1.32-4.24 2.5 2.5 0 0 0-4.44-2.04z")),
								)),
							),
							el.Show(
								func() bool {
									_ = ver()
									if i >= len(*all) {
										return false
									}
									m := (*all)[i]
									return m.Role == "assistant" && m.Streaming && strings.TrimSpace(m.Content) == "" && strings.TrimSpace(m.Thinking) != ""
								},
								el.Div(
									el.Class("flex gap-1 items-center"),
									el.Span(el.Class("w-1.5 h-1.5 rounded-full bg-cyan-500/80 thinking-dot")),
									el.Span(el.Class("w-1.5 h-1.5 rounded-full bg-cyan-500/80 thinking-dot")),
									el.Span(el.Class("w-1.5 h-1.5 rounded-full bg-cyan-500/80 thinking-dot")),
								),
							),
						),
						el.SVGTag("svg",
							el.Attr("viewBox", "0 0 20 20"),
							el.Attr("fill", "currentColor"),
							el.Attr("aria-hidden", "true"),
							el.Class("thinking-chevron h-4 w-4 shrink-0 text-zinc-500"),
							el.SVGTag("path", el.Attr("fill-rule", "evenodd"), el.Attr("d", "M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z"), el.Attr("clip-rule", "evenodd")),
						),
					),
					el.Div(
						el.DynClass(func() string {
							_ = ver()
							if i >= len(*all) {
								return ""
							}
							m := (*all)[i]
							active := m.Role == "assistant" && m.Streaming && strings.TrimSpace(m.Content) == "" && strings.TrimSpace(m.Thinking) != ""
							base := "pb-1 pl-5 text-[13px] text-zinc-400 leading-relaxed"
							if active {
								return base + " thinking-shimmer"
							}
							return base
						}),
						MarkdownBlock(func() string {
							_ = ver()
							if i >= len(*all) {
								return ""
							}
							return (*all)[i].Thinking
						}),
					),
				),
			),
			el.Div(
				el.Class("rounded-2xl rounded-tl-md bg-zinc-900/40 px-4 py-3 relative min-w-[160px] group"),
				el.Button(
					el.Type("button"),
					el.Class("absolute top-2 right-2 p-1.5 rounded-lg bg-zinc-800/60 text-zinc-500 hover:text-cyan-400 opacity-0 group-hover:opacity-100 transition-all z-10"),
					el.Attr("title", "Copy markdown"),
					el.OnClick(func(dom.Event) {
						_ = ver()
						if i < len(*all) {
							CopyToClipboard((*all)[i].Content)
							setCopied(true)
							time.AfterFunc(2*time.Second, func() { setCopied(false) })
						}
					}),
					el.Dynamic(func() el.Node {
						if copied() {
							return el.SVGTag("svg",
								el.Attr("viewBox", "0 0 24 24"),
								el.Attr("fill", "none"),
								el.Attr("stroke", "currentColor"),
								el.Attr("stroke-width", "2"),
								el.Class("w-4 h-4 text-emerald-500"),
								el.SVGTag("polyline", el.Attr("points", "20 6 9 17 4 12")),
							)
						}
						return el.SVGTag("svg",
							el.Attr("viewBox", "0 0 24 24"),
							el.Attr("fill", "none"),
							el.Attr("stroke", "currentColor"),
							el.Attr("stroke-width", "2"),
							el.Class("w-4 h-4"),
							el.SVGTag("rect", el.Attr("x", "9"), el.Attr("y", "9"), el.Attr("width", "13"), el.Attr("height", "13"), el.Attr("rx", "2"), el.Attr("ry", "2")),
							el.SVGTag("path", el.Attr("d", "M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1")),
						)
					}),
				),
				el.Div(
					el.Class("prose-chat text-zinc-200 pb-7"),
					MarkdownBlock(func() string {
						_ = ver()
						if i >= len(*all) {
							return ""
						}
						return (*all)[i].Content
					}),
				),
				el.Div(
					el.Class("absolute bottom-2 right-3 flex items-center gap-2 text-[10px] text-zinc-500 font-medium"),
					el.Show(func() bool {
						_ = ver()
						if i >= len(*all) {
							return false
						}
						return (*all)[i].Model != ""
					},
						el.Div(el.Class("flex items-center gap-1"),
							el.SVGTag("svg", el.Attr("viewBox", "0 0 24 24"), el.Attr("fill", "none"), el.Attr("stroke", "currentColor"), el.Attr("stroke-width", "2"), el.Class("w-2.5 h-2.5"),
								el.SVGTag("path", el.Attr("d", "M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z")),
								el.SVGTag("polyline", el.Attr("points", "3.27 6.96 12 12.01 20.73 6.96")),
								el.SVGTag("line", el.Attr("x1", "12"), el.Attr("y1", "22.08"), el.Attr("x2", "12"), el.Attr("y2", "12")),
							),
							el.DynText(func() string {
								_ = ver()
								if i >= len(*all) {
									return ""
								}
								return (*all)[i].Model
							}),
						),
					),
					el.Show(func() bool {
						_ = ver()
						if i >= len(*all) {
							return false
						}
						return (*all)[i].Model != "" && (*all)[i].TFT != ""
					},
						el.Span(el.Class("opacity-30"), el.Text("|")),
					),
					el.Show(func() bool {
						_ = ver()
						if i >= len(*all) {
							return false
						}
						return (*all)[i].TFT != ""
					},
						el.Div(el.Class("flex items-center gap-1"),
							el.SVGTag("svg", el.Attr("viewBox", "0 0 24 24"), el.Attr("fill", "none"), el.Attr("stroke", "currentColor"), el.Attr("stroke-width", "2"), el.Class("w-2.5 h-2.5"),
								el.SVGTag("path", el.Attr("d", "M13 2L3 14h9l-1 8 10-12h-9l1-8z")),
							),
							el.DynText(func() string {
								_ = ver()
								if i >= len(*all) {
									return ""
								}
								return (*all)[i].TFT + " tft"
							}),
						),
					),
					el.Show(func() bool {
						_ = ver()
						if i >= len(*all) {
							return false
						}
						return (*all)[i].TFT != "" && (*all)[i].TPS != ""
					},
						el.Span(el.Class("opacity-30"), el.Text("|")),
					),
					el.Show(func() bool {
						_ = ver()
						if i >= len(*all) {
							return false
						}
						return (*all)[i].TPS != ""
					},
						el.Div(el.Class("flex items-center gap-1"),
							el.SVGTag("svg", el.Attr("viewBox", "0 0 24 24"), el.Attr("fill", "none"), el.Attr("stroke", "currentColor"), el.Attr("stroke-width", "2"), el.Class("w-2.5 h-2.5"),
								el.SVGTag("path", el.Attr("d", "M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z")),
								el.SVGTag("path", el.Attr("d", "M3.27 6.96L12 12.01l8.73-5.05")),
								el.SVGTag("path", el.Attr("d", "M12 22.08V12")),
							),
							el.DynText(func() string {
								_ = ver()
								if i >= len(*all) {
									return ""
								}
								return (*all)[i].TPS + " tps"
							}),
						),
					),
					el.Show(func() bool {
						_ = ver()
						if i >= len(*all) {
							return false
						}
						return (*all)[i].TPS != "" && (*all)[i].Duration != ""
					},
						el.Span(el.Class("opacity-30"), el.Text("|")),
					),
					el.Show(func() bool {
						_ = ver()
						if i >= len(*all) {
							return false
						}
						return (*all)[i].Duration != ""
					},
						el.Div(el.Class("flex items-center gap-1"),
							el.SVGTag("svg", el.Attr("viewBox", "0 0 24 24"), el.Attr("fill", "none"), el.Attr("stroke", "currentColor"), el.Attr("stroke-width", "2"), el.Class("w-2.5 h-2.5"),
								el.SVGTag("circle", el.Attr("cx", "12"), el.Attr("cy", "12"), el.Attr("r", "10")),
								el.SVGTag("path", el.Attr("d", "M12 6v6l4 2")),
							),
							el.DynText(func() string {
								_ = ver()
								if i >= len(*all) {
									return ""
								}
								return (*all)[i].Duration
							}),
						),
					),
					el.Show(func() bool {
						_ = ver()
						if i >= len(*all) {
							return false
						}
						return (*all)[i].Duration != ""
					},
						el.Span(el.Class("opacity-30"), el.Text("|")),
					),
					el.Div(el.Class("flex items-center gap-1"),
						el.SVGTag("svg", el.Attr("viewBox", "0 0 24 24"), el.Attr("fill", "none"), el.Attr("stroke", "currentColor"), el.Attr("stroke-width", "2"), el.Class("w-2.5 h-2.5"),
							el.SVGTag("rect", el.Attr("x", "3"), el.Attr("y", "4"), el.Attr("width", "18"), el.Attr("height", "16"), el.Attr("rx", "2")),
							el.SVGTag("path", el.Attr("d", "M3 10h18")),
						),
						el.DynText(func() string {
							_ = ver()
							if i >= len(*all) {
								return ""
							}
							return (*all)[i].Timestamp
						}),
					),
				),
			),
		),
	)
}

func MarkdownBlock(markdownGetter func() string) el.Node {
	return el.Div(
		el.OnMount(func(element dom.Element) {
			reactive.CreateEffect(func() {
				md := markdownGetter()
				element.SetInnerHTML(RenderMarkdown(md))
			})
		}),
	)
}

func SettingsModal(s *state.ChatState, loadModels func()) el.Node {
	return el.Div(
		el.Style("z-index", "100"),
		el.DynClass(func() string {
			if !s.ShowSettings() {
				return "hidden"
			}
			return "fixed inset-0"
		}),
		el.Div(el.Class("absolute inset-0 bg-black/70"), el.OnClick(func(dom.Event) { s.SetShowSettings(false) })),
		el.Div(
			el.Class("absolute inset-0 flex items-center justify-center p-4 pointer-events-none"),
			el.Div(
				el.Class("pointer-events-auto w-full max-w-lg rounded-xl border border-zinc-700/80 bg-zinc-900 shadow-2xl shadow-black/60"),
				el.Div(
					el.Class("px-5 py-4 border-b border-zinc-800 flex items-center justify-between gap-3"),
					el.Div(el.Class("text-sm font-semibold text-zinc-100"), el.Text("Provider")),
					el.Button(el.Type("button"), el.Class("text-zinc-500 hover:text-zinc-300 text-lg leading-none px-2 py-1 rounded-lg hover:bg-zinc-800"), el.Text("×"), el.OnClick(func(dom.Event) { s.SetShowSettings(false) })),
				),
				el.Div(
					el.Class("px-5 py-4 space-y-4 max-h-[min(70vh,520px)] overflow-y-auto"),
					el.Div(
						el.Class("space-y-1"),
						el.Label(el.Class("text-[10px] uppercase font-semibold text-zinc-500 ml-1"), el.Text("API URL")),
						el.Input(el.Class("w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-200 focus:ring-1 focus:ring-cyan-500/30 outline-none"), el.Type("text"), el.DynProp("value", func() any { return s.BaseURL() }), el.OnInput(func(e dom.Event) { s.SetBaseURL(e.TargetValue()); s.Persist() }), el.OnBlur(func(dom.Event) { loadModels() })),
					),
					el.Div(
						el.Class("space-y-1"),
						el.Label(el.Class("text-[10px] uppercase font-semibold text-zinc-500 ml-1"), el.Text("API Key")),
						el.Input(el.Class("w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-200 focus:ring-1 focus:ring-cyan-500/30 outline-none"), el.Type("password"), el.DynProp("value", func() any { return s.APIKey() }), el.OnInput(func(e dom.Event) { s.SetAPIKey(e.TargetValue()); s.Persist() }), el.OnBlur(func(dom.Event) { loadModels() })),
					),
					el.Div(
						el.Class("space-y-1"),
						el.Label(el.Class("text-[10px] uppercase font-semibold text-zinc-500 ml-1"), el.Text("System Prompt")),
						el.Textarea(el.Class("w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-200 min-h-[80px] focus:ring-1 focus:ring-cyan-500/30 outline-none"), el.DynProp("value", func() any { return s.SystemPrompt() }), el.OnInput(func(e dom.Event) { s.SetSystemPrompt(e.TargetValue()); s.Persist() })),
					),
					el.Div(
						el.Class("space-y-2"),
						el.Label(el.Class("text-[10px] uppercase font-semibold text-zinc-500 ml-1"), el.Text("Model Selection")),
						el.Dynamic(func() el.Node {
							_ = s.ModelsVer()
							mods := []any{
								el.Class("w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-200 focus:ring-1 focus:ring-cyan-500/30 outline-none"),
								el.OnInput(func(e dom.Event) {
									s.SetModel(e.TargetValue())
									s.Persist()
								}),
								el.OnMount(func(element dom.Element) {
									reactive.CreateEffect(func() {
										element.SetProperty("value", s.Model())
									})
								}),
							}
							if len(s.ModelOptions) == 0 {
								mods = append(mods, el.Option(
									el.Value(""),
									el.Text("— Load models: blur the URL or API key field —"),
								))
							} else {
								for _, id := range s.ModelOptions {
									mods = append(mods, el.Option(
										el.Value(id),
										el.Text(id),
									))
								}
							}
							return el.Select(mods...)
						}),
						el.Div(
							el.DynClass(func() string {
								if !s.ModelsLoading() {
									return "hidden"
								}
								return "text-xs text-cyan-500/90 mt-2 flex items-center gap-2"
							}),
							el.Span(el.Class("inline-block w-3 h-3 border-2 border-cyan-500/30 border-t-cyan-500 rounded-full animate-spin")),
							el.Text("Loading models…"),
						),
						el.Show(s.ModelsErr,
							el.Div(
								el.Class("text-xs text-red-400/90 mt-2"),
								el.DynText(s.ErrMsg),
							),
						),
					),
					el.Div(
						el.Class("flex justify-end gap-2 pt-2"),
						el.Button(
							el.Type("button"),
							el.Class("px-4 py-2 rounded-lg text-sm border border-zinc-700 text-zinc-300 hover:bg-zinc-800"),
							el.Text("Cancel"),
							el.OnClick(func(dom.Event) { s.SetShowSettings(false) }),
						),
						el.Button(
							el.Type("button"),
							el.Class("px-4 py-2 rounded-lg text-sm font-medium bg-cyan-600 hover:bg-cyan-500 text-zinc-950"),
							el.Text("Done"),
							el.OnClick(func(dom.Event) { s.SetShowSettings(false) }),
						),
					),
				),
			),
		),
	)
}
