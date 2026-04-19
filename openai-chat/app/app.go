package app

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/openai-chat/components"
	"github.com/latentarts/gu-examples/openai-chat/state"
)

func App(styles el.Node) el.Node {
	store := jsutil.LocalStorage()
	baseURL, setBaseURL := reactive.NewSignal(store.Get(state.StorageKeyURL))
	if baseURL() == "" {
		setBaseURL("https://api.openai.com/v1")
	}
	apiKey, setAPIKey := reactive.NewSignal(store.Get(state.StorageKeyKey))
	model, setModel := reactive.NewSignal(store.Get(state.StorageKeyModel))
	if model() == "" {
		setModel("gpt-4o-mini")
	}
	systemPrompt, setSystemPrompt := reactive.NewSignal(store.Get(state.StorageKeySystem))
	if systemPrompt() == "" {
		setSystemPrompt("You are a helpful assistant.")
	}
	showSettings, setShowSettings := reactive.NewSignal(false)

	var modelOptions []string
	if store.Get(state.StorageKeyModelIDsForURL) == strings.TrimSpace(baseURL()) {
		if raw := store.Get(state.StorageKeyModelIDs); raw != "" {
			var ids []string
			if err := json.Unmarshal([]byte(raw), &ids); err == nil && len(ids) > 0 {
				modelOptions = ids
			}
		}
	}
	modelsVer, setModelsVer := reactive.NewSignal(0)
	if len(modelOptions) > 0 {
		setModelsVer(1)
	}
	modelsLoading, setModelsLoading := reactive.NewSignal(false)
	modelsErr, setModelsErr := reactive.NewSignal("")
	msgVer, setMsgVer := reactive.NewSignal(0)
	listLen, setListLen := reactive.NewSignal(0)
	var messages []state.ChatMsg
	generating, setGenerating := reactive.NewSignal(false)
	errMsg, setErrMsg := reactive.NewSignal("")
	var chatList dom.Element
	var composerInput dom.Element

	persist := func() {
		store.Set(state.StorageKeyURL, baseURL())
		store.Set(state.StorageKeyKey, apiKey())
		store.Set(state.StorageKeyModel, model())
		store.Set(state.StorageKeySystem, systemPrompt())
	}

	persistModelIDCache := func() {
		if len(modelOptions) == 0 {
			store.Remove(state.StorageKeyModelIDs)
			store.Remove(state.StorageKeyModelIDsForURL)
			return
		}
		raw, err := json.Marshal(modelOptions)
		if err != nil {
			return
		}
		store.Set(state.StorageKeyModelIDs, string(raw))
		store.Set(state.StorageKeyModelIDsForURL, strings.TrimSpace(baseURL()))
	}

	appendAssistantPlaceholder := func() {
		messages = append(messages, state.ChatMsg{Role: "assistant", Streaming: true, Timestamp: state.NowTimestamp()})
		setMsgVer(msgVer() + 1)
		setListLen(len(messages))
	}

	updateLastAssistant := func(thinking, content string, done bool, tft, tps, duration, modelName string) {
		if len(messages) == 0 {
			return
		}
		last := len(messages) - 1
		if messages[last].Role != "assistant" {
			return
		}
		m := messages[last]
		m.Thinking = thinking
		m.Content = content
		m.Streaming = !done
		m.TFT = tft
		m.TPS = tps
		m.Duration = duration
		m.Model = modelName
		messages[last] = m
		setMsgVer(msgVer() + 1)
	}

	input, setInput := reactive.NewSignal("")

	send := func() {
		setErrMsg("")
		text := input()
		if text == "" || generating() {
			return
		}
		setInput("")
		if !composerInput.IsNull() {
			composerInput.SetProperty("value", "")
			composerInput.Focus()
		}
		messages = append(messages, state.ChatMsg{Role: "user", Content: text, Timestamp: state.NowTimestamp()})
		appendAssistantPlaceholder()

		history := make([]state.APIMessage, 0, len(messages)+1)
		if sys := strings.TrimSpace(systemPrompt()); sys != "" {
			history = append(history, state.APIMessage{Role: "system", Content: sys})
		}
		for _, m := range messages {
			if m.Role == "assistant" && m.Streaming {
				continue
			}
			history = append(history, state.APIMessage{Role: m.Role, Content: m.Content})
		}

		setGenerating(true)
		modelName := model()
		go func() {
			chatURL, err := state.NormalizeChatCompletionsURL(baseURL())
			if err != nil {
				setErrMsg(err.Error())
				setGenerating(false)
				return
			}
			thinking := strings.Builder{}
			content := strings.Builder{}
			startTime := time.Now()
			var firstTokenTime time.Time
			tokenCount := 0

			err = state.StreamChat(chatURL, apiKey(), modelName, history, func(rDelta, cDelta string) bool {
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
				updateLastAssistant(strings.TrimSpace(th), strings.TrimSpace(co), false, tft, "", "", modelName)
				return true
			})
			if err != nil {
				setErrMsg(err.Error())
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
			updateLastAssistant(strings.TrimSpace(th), strings.TrimSpace(co), true, tft, tps, fmt.Sprintf("%.2fs", duration), modelName)
			js.Global().Set("__openaiChatAbort", js.Undefined())
			setGenerating(false)
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
		messages = nil
		setMsgVer(msgVer() + 1)
		setListLen(0)
		setErrMsg("")
		setInput("")
	}

	loadModels := func() {
		u := strings.TrimSpace(baseURL())
		if u == "" {
			setModelsErr("Enter the API URL first.")
			return
		}
		modelsURL, err := state.DeriveModelsEndpoint(u)
		if err != nil {
			setModelsErr(err.Error())
			return
		}
		go func() {
			setModelsLoading(true)
			setModelsErr("")
			ids, err := state.FetchModelsList(modelsURL, apiKey())
			setModelsLoading(false)
			if err != nil {
				setModelsErr(err.Error())
				modelOptions = nil
				persistModelIDCache()
			} else {
				modelOptions = ids
				persistModelIDCache()
			}
			setModelsVer(modelsVer() + 1)
		}()
	}

	persist()

	return el.Div(
		styles,
		el.Class("h-screen flex flex-col bg-[#0c0c0f] text-zinc-100"),
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
					el.Button(el.Type("button"), el.Class("text-xs px-3 py-1.5 rounded-lg border border-zinc-700 text-zinc-300 hover:bg-zinc-800/80 transition-colors"), el.Text("Settings"), el.OnClick(func(dom.Event) { setShowSettings(!showSettings()) })),
					el.Button(el.Type("button"), el.Class("text-xs px-3 py-1.5 rounded-lg border border-zinc-700 text-zinc-400 hover:text-red-400 hover:border-red-900/50 transition-colors"), el.Text("Clear"), el.OnClick(func(dom.Event) { clearChat() })),
				),
			),
		),
		el.Show(func() bool { return errMsg() != "" }, el.Div(el.Class("shrink-0 max-w-3xl mx-auto w-full px-4 pt-3"), el.Div(el.Class("text-xs rounded-lg px-3 py-2 bg-red-950/50 text-red-300 border border-red-900/40"), el.DynText(errMsg)))),
		el.Div(
			el.Class("flex-1 relative min-h-0"),
			el.Div(
				el.Class("absolute inset-0 overflow-y-auto px-4 py-6 pb-48"),
				el.Ref(&chatList),
				el.On("scroll", func(dom.Event) {
					if !chatList.IsNull() {
						_ = chatList.GetProperty("scrollHeight").Int()
					}
				}),
				el.Div(
					el.Class("max-w-3xl mx-auto w-full flex flex-col gap-10"),
					el.For(
						func() []int {
							_ = listLen()
							n := len(messages)
							var starts []int
							for k := 0; k < n; k += 2 {
								starts = append(starts, k)
							}
							return starts
						},
						func(userIdx int, _ int) string { return strconv.Itoa(userIdx) },
						func(userIdx int, _ int) el.Node { return components.TurnGroup(userIdx, &messages, msgVer) },
					),
				),
				el.OnMount(func(element dom.Element) {
					reactive.CreateEffect(func() {
						_ = msgVer()
						_ = generating()
						scrollPos := element.GetProperty("scrollTop").Int()
						scrollHeight := element.GetProperty("scrollHeight").Int()
						clientHeight := element.GetProperty("clientHeight").Int()
						if scrollHeight-scrollPos-clientHeight < 300 {
							element.SetProperty("scrollTop", scrollHeight)
						}
					})
				}),
			),
			el.Div(
				el.Class("absolute bottom-0 left-0 right-0 border-t border-zinc-800 bg-[#0c0c0f]/80 backdrop-blur-md z-10"),
				el.Div(
					el.Class("max-w-3xl mx-auto w-full px-4 py-4"),
					el.Div(
						el.Class("flex gap-2 items-end"),
						el.Textarea(
							el.Class("flex-1 min-h-[48px] max-h-40 resize-y bg-zinc-950 border border-zinc-800 rounded-xl px-4 py-3 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-cyan-500/30 leading-relaxed"),
							el.Ref(&composerInput),
							el.Placeholder("Message…"),
							el.Attr("rows", "2"),
							el.DynProp("value", func() any { return input() }),
							el.OnInput(func(e dom.Event) { setInput(e.TargetValue()) }),
							el.OnKeyDown(func(e dom.Event) {
								if e.Key() == "Enter" && !e.Value.Get("shiftKey").Bool() {
									e.PreventDefault()
									send()
								}
							}),
						),
						el.Button(
							el.Type("button"),
							el.Class("px-4 py-3 rounded-xl text-sm font-medium bg-cyan-600 hover:bg-cyan-500 text-zinc-950 transition-colors disabled:opacity-40 disabled:pointer-events-none"),
							el.Text("Send"),
							el.DynAttr("disabled", func() string {
								if generating() || strings.TrimSpace(input()) == "" {
									return "true"
								}
								return ""
							}),
							el.OnClick(func(dom.Event) { send() }),
						),
					),
				),
			),
		),
		el.Div(
			el.Style("z-index", "100"),
			el.DynClass(func() string {
				if !showSettings() {
					return "hidden"
				}
				return "fixed inset-0"
			}),
			el.Div(el.Class("absolute inset-0 bg-black/70"), el.OnClick(func(dom.Event) { setShowSettings(false) })),
			el.Div(
				el.Class("absolute inset-0 flex items-center justify-center p-4 pointer-events-none"),
				el.Div(
					el.Class("pointer-events-auto w-full max-w-lg rounded-xl border border-zinc-700/80 bg-zinc-900 shadow-2xl shadow-black/60"),
					el.Div(
						el.Class("px-5 py-4 border-b border-zinc-800 flex items-center justify-between gap-3"),
						el.Div(el.Class("text-sm font-semibold text-zinc-100"), el.Text("Provider")),
						el.Button(el.Type("button"), el.Class("text-zinc-500 hover:text-zinc-300 text-lg leading-none px-2 py-1 rounded-lg hover:bg-zinc-800"), el.Text("×"), el.OnClick(func(dom.Event) { setShowSettings(false) })),
					),
					el.Div(
						el.Class("px-5 py-4 space-y-4 max-h-[min(70vh,520px)] overflow-y-auto"),
						el.Input(el.Class("w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-200"), el.Type("text"), el.DynProp("value", func() any { return baseURL() }), el.OnInput(func(e dom.Event) { setBaseURL(e.TargetValue()); persist() }), el.OnBlur(func(dom.Event) { loadModels() })),
						el.Input(el.Class("w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-200"), el.Type("password"), el.DynProp("value", func() any { return apiKey() }), el.OnInput(func(e dom.Event) { setAPIKey(e.TargetValue()); persist() }), el.OnBlur(func(dom.Event) { loadModels() })),
						el.Textarea(el.Class("w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-200 min-h-[80px]"), el.DynProp("value", func() any { return systemPrompt() }), el.OnInput(func(e dom.Event) { setSystemPrompt(e.TargetValue()); persist() })),
						el.Dynamic(func() el.Node {
							_ = modelsVer()
							args := []any{el.Class("space-y-2")}
							if modelsLoading() {
								args = append(args, el.Div(el.Class("text-xs text-zinc-500"), el.Text("Loading models...")))
							}
							if modelsErr() != "" {
								args = append(args, el.Div(el.Class("text-xs text-red-400"), el.DynText(modelsErr)))
							}
							for _, opt := range modelOptions {
								opt := opt
								args = append(args, el.Button(el.Type("button"), el.Class("w-full text-left px-3 py-2 rounded-lg border border-zinc-800 hover:bg-zinc-800"), el.Text(opt), el.OnClick(func(dom.Event) { setModel(opt); persist() })))
							}
							return el.Div(args...)
						}),
					),
				),
			),
		),
	)
}
