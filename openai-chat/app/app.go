//go:build js && wasm

package app

import (
	"fmt"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/openai-chat/components"
	"github.com/latentarts/gu-examples/openai-chat/state"
)

func App(styles el.Node) el.Node {
	s := state.NewChatState()
	
	var chatList dom.Element
	var composerInput dom.Element

	updateLastAssistant := func(thinking, content string, done bool, tft, tps, duration, modelName string) {
		if len(s.Messages) == 0 {
			return
		}
		last := len(s.Messages) - 1
		if s.Messages[last].Role != "assistant" {
			return
		}
		m := s.Messages[last]
		m.Thinking = thinking
		m.Content = content
		m.Streaming = !done
		m.TFT = tft
		m.TPS = tps
		m.Duration = duration
		m.Model = modelName
		s.Messages[last] = m
		s.SetMsgVer(s.MsgVer() + 1)
	}

	send := func() {
		s.SetErrMsg("")
		text := s.Input()
		if text == "" || s.Generating() {
			return
		}
		s.SetInput("")
		if !composerInput.IsNull() {
			composerInput.SetProperty("value", "")
			composerInput.Focus()
		}
		s.Messages = append(s.Messages, state.ChatMsg{Role: "user", Content: text, Timestamp: state.NowTimestamp()})
		s.Messages = append(s.Messages, state.ChatMsg{Role: "assistant", Streaming: true, Timestamp: state.NowTimestamp()})
		s.SetMsgVer(s.MsgVer() + 1)
		s.SetListLen(len(s.Messages))

		history := make([]state.APIMessage, 0, len(s.Messages)+1)
		if sys := strings.TrimSpace(s.SystemPrompt()); sys != "" {
			history = append(history, state.APIMessage{Role: "system", Content: sys})
		}
		for _, m := range s.Messages {
			if m.Role == "assistant" && m.Streaming {
				continue
			}
			history = append(history, state.APIMessage{Role: m.Role, Content: m.Content})
		}

		s.SetGenerating(true)
		modelName := s.Model()
		go func() {
			chatURL, err := state.NormalizeChatCompletionsURL(s.BaseURL())
			if err != nil {
				s.SetErrMsg(err.Error())
				s.SetGenerating(false)
				return
			}
			thinking := strings.Builder{}
			content := strings.Builder{}
			startTime := time.Now()
			var firstTokenTime time.Time
			tokenCount := 0

			err = state.StreamChat(chatURL, s.APIKey(), modelName, history, func(rDelta, cDelta string) bool {
				if firstTokenTime.IsZero() && (rDelta != "" || cDelta != "") {
					firstTokenTime = time.Now()
				}
				if cDelta != "" {
					tokenCount++
				}
				if rDelta != "" {
					thinking.WriteString(rDelta)
				}
				if cDelta != "" {
					content.WriteString(cDelta)
				}
				th := thinking.String()
				co := content.String()
				if extraTh, rest := state.SplitThinkingFromContent(co); extraTh != "" {
					th = th + "\n" + extraTh
					co = rest
				}
				tft := ""
				if !firstTokenTime.IsZero() {
					tft = fmt.Sprintf("%.2fs", firstTokenTime.Sub(startTime).Seconds())
				}
				updateLastAssistant(th, co, false, tft, "", "", modelName)
				return true
			})
			if err != nil {
				s.SetErrMsg(err.Error())
			}
			endTime := time.Now()
			duration := endTime.Sub(startTime).Seconds()
			tft := ""
			if !firstTokenTime.IsZero() {
				tft = fmt.Sprintf("%.2fs", firstTokenTime.Sub(startTime).Seconds())
			}
			tps := ""
			if duration > 0 {
				tps = fmt.Sprintf("%.1f", float64(tokenCount)/duration)
			}
			th := thinking.String()
			co := content.String()
			if extraTh, rest := state.SplitThinkingFromContent(co); extraTh != "" {
				th = th + "\n" + extraTh
				co = rest
			}
			updateLastAssistant(th, co, true, tft, tps, fmt.Sprintf("%.2fs", duration), modelName)
			js.Global().Set("__openaiChatAbort", js.Undefined())
			s.SetGenerating(false)
		}()
	}

	stop := func() {
		abort := js.Global().Get("__openaiChatAbort")
		if !abort.IsUndefined() && abort.Get("abort").Truthy() {
			abort.Call("abort")
		}
	}

	clearChat := func() {
		stop()
		s.Messages = nil
		s.SetMsgVer(s.MsgVer() + 1)
		s.SetListLen(0)
		s.SetErrMsg("")
		s.SetInput("")
	}

	loadModels := func() {
		u := strings.TrimSpace(s.BaseURL())
		if u == "" {
			s.SetModelsErr("Enter the API URL first.")
			return
		}
		modelsURL, err := state.DeriveModelsEndpoint(u)
		if err != nil {
			s.SetModelsErr(err.Error())
			return
		}
		go func() {
			s.SetModelsLoading(true)
			s.SetModelsErr("")
			ids, err := state.FetchModelsList(modelsURL, s.APIKey())
			s.SetModelsLoading(false)
			if err != nil {
				s.SetModelsErr(err.Error())
				s.ModelOptions = nil
				s.PersistModelIDCache()
			} else {
				s.ModelOptions = ids
				s.PersistModelIDCache()
			}
			s.SetModelsVer(s.ModelsVer() + 1)
		}()
	}

	return el.Div(
		styles,
		el.Class("h-screen flex flex-col bg-[#0c0c0f] text-zinc-100"),
		
		// Header
		el.Div(
			el.Class("shrink-0 border-b border-zinc-800/80 backdrop-blur-sm bg-[#0c0c0f]/90 z-20"),
			el.Div(
				el.Class("max-w-3xl mx-auto w-full px-4 py-3 flex items-center justify-between gap-3"),
				el.Div(
					el.Class("min-w-0"),
					el.Div(el.Class("text-sm font-semibold tracking-tight text-zinc-100"), el.Text("Chat")),
					el.Div(el.Class("text-[11px] text-zinc-500 truncate"), el.Text("OpenAI-compatible SSE · streaming · markdown")),
				),
				el.Div(
					el.Class("flex items-center gap-2 shrink-0 relative z-20"),
					el.Button(el.Type("button"), el.Class("text-[11px] font-medium px-3 py-1.5 rounded-lg border border-zinc-700 text-zinc-300 hover:bg-zinc-800/80 transition-colors uppercase tracking-wider"), el.Text("Settings"), el.OnClick(func(dom.Event) { s.SetShowSettings(!s.ShowSettings()) })),
					el.Button(el.Type("button"), el.Class("text-[11px] font-medium px-3 py-1.5 rounded-lg border border-zinc-700 text-zinc-400 hover:text-red-400 hover:border-red-900/50 transition-colors uppercase tracking-wider"), el.Text("Clear"), el.OnClick(func(dom.Event) { clearChat() })),
				),
			),
		),
		
		// Error bar
		el.Show(func() bool { return s.ErrMsg() != "" }, el.Div(el.Class("shrink-0 max-w-3xl mx-auto w-full px-4 pt-3"), el.Div(el.Class("text-xs rounded-lg px-3 py-2 bg-red-950/50 text-red-300 border border-red-900/40"), el.DynText(s.ErrMsg)))),
		
		// Chat List
		el.Div(
			el.Class("flex-1 relative min-h-0"),
			el.Div(
				el.Class("absolute inset-0 overflow-y-auto px-4 py-6 pb-48 scroll-smooth"),
				el.Ref(&chatList),
				el.Div(
					el.Class("max-w-3xl mx-auto w-full flex flex-col gap-10"),
					el.For(
						func() []int {
							_ = s.ListLen()
							n := len(s.Messages)
							var starts []int
							for k := 0; k < n; k += 2 {
								starts = append(starts, k)
							}
							return starts
						},
						func(userIdx int, _ int) string { return strconv.Itoa(userIdx) },
						func(userIdx int, _ int) el.Node { return components.TurnGroup(userIdx, &s.Messages, s.MsgVer) },
					),
				),
				el.OnMount(func(element dom.Element) {
					reactive.CreateEffect(func() {
						_ = s.MsgVer()
						_ = s.Generating()
						scrollPos := element.GetProperty("scrollTop").Int()
						scrollHeight := element.GetProperty("scrollHeight").Int()
						clientHeight := element.GetProperty("clientHeight").Int()
						if scrollHeight-scrollPos-clientHeight < 400 {
							element.SetProperty("scrollTop", scrollHeight)
						}
					})
				}),
			),
			
			// Composer
			el.Div(
				el.Class("absolute bottom-0 left-0 right-0 border-t border-zinc-800/50 bg-gradient-to-t from-[#0c0c0f] via-[#0c0c0f]/95 to-transparent pt-12 pb-6 z-10"),
				el.Div(
					el.Class("max-w-3xl mx-auto w-full px-4"),
					el.Div(
						el.Class("relative flex gap-3 items-end bg-zinc-900/80 backdrop-blur-sm border border-zinc-800 rounded-2xl p-2 shadow-2xl"),
						el.Textarea(
							el.Class("flex-1 min-h-[44px] max-h-40 resize-none bg-transparent border-none px-3 py-2.5 text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none leading-relaxed"),
							el.Ref(&composerInput),
							el.Placeholder("Ask anything…"),
							el.Attr("rows", "1"),
							el.DynProp("value", func() any { return s.Input() }),
							el.OnInput(func(e dom.Event) { s.SetInput(e.TargetValue()) }),
							el.OnKeyDown(func(e dom.Event) {
								if e.Key() == "Enter" && !e.Value.Get("shiftKey").Bool() {
									e.PreventDefault()
									send()
								}
							}),
						),
						el.Button(
							el.Type("button"),
							el.Class("w-10 h-10 shrink-0 rounded-xl flex items-center justify-center bg-cyan-600 hover:bg-cyan-500 text-zinc-950 transition-all disabled:opacity-20 disabled:grayscale disabled:pointer-events-none shadow-lg shadow-cyan-900/20"),
							el.DynAttr("disabled", func() string {
								if s.Generating() || strings.TrimSpace(s.Input()) == "" {
									return "true"
								}
								return ""
							}),
							el.OnClick(func(dom.Event) { send() }),
							el.SVGTag("svg", el.Attr("viewBox", "0 0 24 24"), el.Attr("fill", "none"), el.Attr("stroke", "currentColor"), el.Attr("stroke-width", "2.5"), el.Class("w-5 h-5"),
								el.SVGTag("line", el.Attr("x1", "12"), el.Attr("y1", "19"), el.Attr("x2", "12"), el.Attr("y2", "5")),
								el.SVGTag("polyline", el.Attr("points", "5 12 12 5 19 12")),
							),
						),
					),
					el.Div(
						el.Class("mt-3 flex justify-center"),
						el.Div(
							el.Class("text-[10px] text-zinc-500 font-medium flex items-center gap-3 uppercase tracking-widest"),
							el.DynText(func() string { return s.Model() }),
							el.Span(el.Class("w-1 h-1 rounded-full bg-zinc-800")),
							el.Text("OpenAI API Compatible"),
						),
					),
				),
			),
		),
		
		// Settings
		components.SettingsModal(s, loadModels),
	)
}
