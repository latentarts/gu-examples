package components

import (
	"syscall/js"
	"time"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/webllm/state"
)

func RenderMarkdown(md string) string {
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
		}
	}
}

func TurnGroup(s *state.ChatState, userIdx int) el.Node {
	return el.Div(
		el.Class("flex flex-col gap-6 w-full"),
		MessageBubble(s, userIdx),
		el.Show(func() bool {
			_ = s.MsgVer()
			return userIdx+1 < len(s.Messages)
		}, el.Dynamic(func() el.Node {
			return MessageBubble(s, userIdx+1)
		})),
	)
}

func MessageBubble(s *state.ChatState, i int) el.Node {
	copied, setCopied := reactive.NewSignal(false)
	item := s.Messages[i]

	if item.Role == "user" {
		return el.Div(
			el.Class("flex w-full justify-end gap-3"),
			el.Div(
				el.Class("max-w-[85%] rounded-2xl rounded-tr-md px-4 py-2 bg-zinc-800 text-zinc-100 text-sm leading-relaxed shadow-sm"),
				el.DynText(func() string {
					_ = s.MsgVer()
					if i >= len(s.Messages) {
						return ""
					}
					return s.Messages[i].Content
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
				MarkdownBlock(func() string {
					_ = s.MsgVer()
					if i >= len(s.Messages) {
						return ""
					}
					return s.Messages[i].Content
				}),
				el.Show(func() bool {
					_ = s.MsgVer()
					return i < len(s.Messages) && s.Messages[i].Streaming && s.Messages[i].Content == ""
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
						_ = s.MsgVer()
						if i < len(s.Messages) {
							CopyToClipboard(s.Messages[i].Content)
							setCopied(true)
							time.AfterFunc(2*time.Second, func() { setCopied(false) })
						}
					}),
					el.SVGTag("svg", el.Attr("viewBox", "0 0 24 24"), el.Attr("fill", "none"), el.Attr("stroke", "currentColor"), el.Attr("stroke-width", "2"), el.Class("w-3 h-3"),
						el.SVGTag("rect", el.Attr("x", "9"), el.Attr("y", "9"), el.Attr("width", "13"), el.Attr("height", "13"), el.Attr("rx", "2"), el.Attr("ry", "2")),
						el.SVGTag("path", el.Attr("d", "M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1")),
					),
					el.DynText(func() string {
						if copied() {
							return "Copied!"
						}
						return "Copy"
					}),
				),
			),
		),
	)
}

func MarkdownBlock(markdownGetter func() string) el.Node {
	return el.Div(
		el.OnMount(func(element dom.Element) {
			reactive.CreateEffect(func() {
				element.SetInnerHTML(RenderMarkdown(markdownGetter()))
			})
		}),
	)
}
