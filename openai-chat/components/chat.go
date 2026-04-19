package components

import (
	"strings"
	"syscall/js"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/openai-chat/state"
)

func RenderMarkdown(md string) string {
	md = strings.TrimSpace(md)
	if md == "" {
		return ""
	}
	return js.Global().Get("Markdown").Call("render", md).String()
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
	return el.Div(
		el.Class("flex flex-col gap-10"),
		MessageBubble((*all)[userIdx], userIdx, all, ver),
		el.Show(func() bool {
			_ = ver()
			return userIdx+1 < len(*all)
		}, el.Dynamic(func() el.Node {
			return MessageBubble((*all)[userIdx+1], userIdx+1, all, ver)
		})),
	)
}

func MessageBubble(item state.ChatMsg, i int, all *[]state.ChatMsg, ver func() int) el.Node {
	// Keep this wrapper in components package so app owns layout, while reusable
	// message rendering logic stays out of the app/root layers.
	if item.Role == "user" {
		return el.Div(
			el.Class("ml-auto max-w-[85%]"),
			el.Div(
				el.Class("rounded-2xl rounded-br-md bg-cyan-600 text-zinc-950 px-4 py-3 text-sm leading-relaxed font-medium"),
				el.DynText(func() string {
					_ = ver()
					if i >= len(*all) {
						return ""
					}
					return (*all)[i].Content
				}),
			),
		)
	}

	return el.Div(
		el.Class("max-w-full"),
		el.Div(el.Class("text-[11px] uppercase tracking-wider text-zinc-500 mb-2"), el.Text("Assistant")),
		el.Show(func() bool {
			_ = ver()
			return i < len(*all) && (*all)[i].Thinking != ""
		},
			el.Div(
				el.Class("mb-3 rounded-xl border border-amber-900/40 bg-amber-950/20 px-4 py-3"),
				el.Div(el.Class("text-[11px] uppercase tracking-wider text-amber-400 mb-2"), el.Text("Reasoning")),
				MarkdownBlock(func() string {
					_ = ver()
					return (*all)[i].Thinking
				}),
			),
		),
		el.Div(
			el.Class("rounded-2xl rounded-bl-md bg-zinc-900 border border-zinc-800 px-4 py-4"),
			MarkdownBlock(func() string {
				_ = ver()
				if i >= len(*all) {
					return ""
				}
				return (*all)[i].Content
			}),
			el.Show(func() bool {
				_ = ver()
				return i < len(*all) && (*all)[i].Streaming
			}, el.Div(el.Class("mt-3 text-[11px] text-zinc-500"), el.Text("Streaming…"))),
			el.Div(
				el.Class("mt-4 flex items-center gap-3 text-[11px] text-zinc-500"),
				el.Show(func() bool {
					_ = ver()
					return i < len(*all) && (*all)[i].Model != ""
				}, el.Span(el.DynText(func() string {
					_ = ver()
					return (*all)[i].Model
				}))),
				el.Show(func() bool {
					_ = ver()
					return i < len(*all) && (*all)[i].TFT != ""
				}, el.Span(el.DynText(func() string {
					_ = ver()
					return "TTFT " + (*all)[i].TFT
				}))),
				el.Show(func() bool {
					_ = ver()
					return i < len(*all) && (*all)[i].TPS != ""
				}, el.Span(el.DynText(func() string {
					_ = ver()
					return (*all)[i].TPS + " tok/s"
				}))),
				el.Show(func() bool {
					_ = ver()
					return i < len(*all) && (*all)[i].Duration != ""
				}, el.Span(el.DynText(func() string {
					_ = ver()
					return (*all)[i].Duration
				}))),
			),
		),
	)
}

func MarkdownBlock(markdownGetter func() string) el.Node {
	return el.Div(
		el.Class("prose prose-invert max-w-none prose-pre:bg-zinc-950 prose-pre:border prose-pre:border-zinc-800 prose-code:text-cyan-300"),
		el.OnMount(func(element dom.Element) {
			reactive.CreateEffect(func() {
				element.SetInnerHTML(RenderMarkdown(markdownGetter()))
			})
		}),
	)
}
