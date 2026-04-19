//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
)

const (
	storageKeyURL            = "openai-chat-base-url"
	storageKeyKey            = "openai-chat-api-key"
	storageKeyModel          = "openai-chat-model"
	storageKeyModelIDs       = "openai-chat-model-ids-json"
	storageKeyModelIDsForURL = "openai-chat-model-ids-for-url"
	storageKeySystem         = "openai-chat-system-prompt"
)

type chatMsg struct {
	Role      string
	Content   string
	Thinking  string
	Streaming bool
	Timestamp string
	Model     string // Model name used for this message
	TFT       string // Time to first token
	TPS       string // Tokens per second
	Duration  string // Total duration
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type streamDelta struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			Reasoning        string `json:"reasoning"`
		} `json:"delta"`
	} `json:"choices"`
}

// Matches <think>, <thinking>, or <think> (common in reasoning models).
var thinkingTagRE = regexp.MustCompile(`(?s)<(?:think|thinking)>(.*?)</(?:think|thinking)>`)

func renderMarkdown(md string) string {
	md = strings.TrimSpace(md)
	if md == "" {
		return ""
	}
	v := js.Global().Get("Markdown").Call("render", md)
	return v.String()
}

func copyToClipboard(text string) {
	if text == "" {
		return
	}
	// Try modern clipboard API
	nav := js.Global().Get("navigator")
	if !nav.IsUndefined() {
		clipboard := nav.Get("clipboard")
		if !clipboard.IsUndefined() && !clipboard.Get("writeText").IsUndefined() {
			clipboard.Call("writeText", text)
			return
		}
	}

	// Fallback to execCommand('copy')
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

func splitThinkingFromContent(s string) (thinking, rest string) {
	s = strings.TrimSpace(s)
	if m := thinkingTagRE.FindStringSubmatch(s); len(m) > 1 {
		th := strings.TrimSpace(m[1])
		rest = strings.TrimSpace(thinkingTagRE.ReplaceAllString(s, ""))
		return th, rest
	}
	return "", s
}

func streamChat(
	url string,
	apiKey string,
	model string,
	history []apiMessage,
	onDelta func(reasoningDelta, contentDelta string) bool,
) error {
	body := map[string]any{
		"model":    model,
		"messages": history,
		"stream":   true,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Accept":        "text/event-stream",
		"Cache-Control": "no-cache",
	}
	if apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}

	init := js.Global().Get("Object").New()
	init.Set("method", "POST")
	init.Set("body", string(raw))
	init.Set("headers", headersToJSObject(headers))

	ac := js.Global().Get("AbortController").New()
	signal := ac.Get("signal")
	init.Set("signal", signal)
	abortHolder := js.Global().Get("Object").New()
	abortHolder.Set("abort", ac.Get("abort"))
	js.Global().Set("__openaiChatAbort", abortHolder)

	promise := js.Global().Call("fetch", url, init)
	respVal, err := jsutil.Await(promise)
	if err != nil {
		return err
	}
	if !respVal.Get("ok").Bool() {
		text, _ := jsutil.Await(respVal.Call("text"))
		return fmt.Errorf("HTTP %d: %s", respVal.Get("status").Int(), text.String())
	}

	reader := respVal.Get("body").Call("getReader")
	decoder := js.Global().Get("TextDecoder").New()

	var lineBuf strings.Builder
	readNext := func() (js.Value, error) {
		return jsutil.Await(reader.Call("read"))
	}

	for {
		result, err := readNext()
		if err != nil {
			return err
		}
		if result.Get("done").Bool() {
			break
		}
		chunk := result.Get("value")
		text := decoder.Call("decode", chunk, js.ValueOf(map[string]any{"stream": true})).String()
		lineBuf.WriteString(text)

		for {
			data := lineBuf.String()
			idx := strings.IndexByte(data, '\n')
			if idx < 0 {
				break
			}
			line := data[:idx]
			lineBuf.Reset()
			lineBuf.WriteString(data[idx+1:])

			line = strings.TrimRight(line, "\r")
			if line == "" {
				continue
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")
			payload = strings.TrimSpace(payload)
			if payload == "[DONE]" {
				return nil
			}

			var sd streamDelta
			if err := json.Unmarshal([]byte(payload), &sd); err != nil {
				continue
			}
			if len(sd.Choices) == 0 {
				continue
			}
			d := sd.Choices[0].Delta
			r := d.ReasoningContent
			if r == "" {
				r = d.Reasoning
			}
			c := d.Content
			if r == "" && c == "" {
				continue
			}
			if !onDelta(r, c) {
				return nil
			}
		}
	}

	emptyBuf := js.Global().Get("Uint8Array").New()
	flush := decoder.Call("decode", emptyBuf, js.ValueOf(map[string]any{"stream": false})).String()
	if flush != "" {
		lineBuf.WriteString(flush)
	}
	rem := lineBuf.String()
	for rem != "" {
		idx := strings.IndexByte(rem, '\n')
		var line string
		if idx < 0 {
			line, rem = rem, ""
		} else {
			line, rem = rem[:idx], rem[idx+1:]
		}
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if payload == "[DONE]" {
			return nil
		}
		var sd streamDelta
		if err := json.Unmarshal([]byte(payload), &sd); err != nil {
			continue
		}
		if len(sd.Choices) == 0 {
			continue
		}
		d := sd.Choices[0].Delta
		r := d.ReasoningContent
		if r == "" {
			r = d.Reasoning
		}
		c := d.Content
		if r == "" && c == "" {
			continue
		}
		if !onDelta(r, c) {
			return nil
		}
	}
	return nil
}

func headersToJSObject(h map[string]string) js.Value {
	o := js.Global().Get("Object").New()
	for k, v := range h {
		o.Set(k, v)
	}
	return o
}

// deriveModelsEndpoint maps a chat completions or API base URL to the OpenAI-compatible GET …/models URL.
func deriveModelsEndpoint(chatURL string) (string, error) {
	u := strings.TrimSpace(chatURL)
	if u == "" {
		return "", fmt.Errorf("URL is empty")
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	path := strings.TrimSuffix(parsed.Path, "/")
	switch {
	case strings.HasSuffix(path, "/chat/completions"):
		path = strings.TrimSuffix(path, "/chat/completions") + "/models"
	case strings.HasSuffix(path, "/models"):
		// already a models list URL
	case strings.HasSuffix(path, "/v1"):
		path = path + "/models"
	case path == "" || path == "/":
		path = "/v1/models"
	default:
		path = "/v1/models"
	}
	parsed.Path = path
	parsed.RawQuery = ""
	parsed.RawFragment = ""
	return parsed.String(), nil
}

// normalizeChatCompletionsURL turns a base URL (e.g. …/v1 or …/v1/models) into POST …/v1/chat/completions.
func normalizeChatCompletionsURL(raw string) (string, error) {
	u := strings.TrimSpace(raw)
	if u == "" {
		return "", fmt.Errorf("URL is empty")
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	path := strings.TrimSuffix(parsed.Path, "/")
	switch {
	case strings.HasSuffix(path, "/chat/completions"):
		// already the completions endpoint
	case strings.HasSuffix(path, "/models"):
		path = strings.TrimSuffix(path, "/models") + "/chat/completions"
	case strings.HasSuffix(path, "/v1"):
		path = path + "/chat/completions"
	case path == "" || path == "/":
		path = "/v1/chat/completions"
	default:
		path = "/v1/chat/completions"
	}
	parsed.Path = path
	parsed.RawQuery = ""
	parsed.RawFragment = ""
	return parsed.String(), nil
}

type modelsListResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func fetchModelsList(modelsURL, apiKey string) ([]string, error) {
	headers := map[string]string{
		"Accept": "application/json",
	}
	if apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}
	resp, err := jsutil.Fetch(modelsURL, &jsutil.FetchOptions{
		Method:  "GET",
		Headers: headers,
	})
	if err != nil {
		return nil, err
	}
	body, err := resp.Text()
	if err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.Status, body)
	}
	var parsed modelsListResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return nil, fmt.Errorf("parse models: %w", err)
	}
	ids := make([]string, 0, len(parsed.Data))
	for _, d := range parsed.Data {
		if d.ID != "" {
			ids = append(ids, d.ID)
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return nil, fmt.Errorf("no models in response")
	}
	return ids, nil
}

func App() el.Node {
	store := jsutil.LocalStorage()
	baseURL, setBaseURL := reactive.NewSignal(store.Get(storageKeyURL))
	if baseURL() == "" {
		setBaseURL("https://api.openai.com/v1")
	}
	apiKey, setAPIKey := reactive.NewSignal(store.Get(storageKeyKey))
	model, setModel := reactive.NewSignal(store.Get(storageKeyModel))
	if model() == "" {
		setModel("gpt-4o-mini")
	}
	systemPrompt, setSystemPrompt := reactive.NewSignal(store.Get(storageKeySystem))
	if systemPrompt() == "" {
		setSystemPrompt("You are a helpful assistant.")
	}
	showSettings, setShowSettings := reactive.NewSignal(false)

	canScroll, setCanScroll := reactive.NewSignal(false)

	var modelOptions []string
	// Restore cached model list if it was saved for this same API URL.
	if store.Get(storageKeyModelIDsForURL) == strings.TrimSpace(baseURL()) {
		if raw := store.Get(storageKeyModelIDs); raw != "" {
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
	// Bumps only when the message *count* changes (append / clear), not on every
	// Bumps only when message count changes, not each SSE token — keeps el.For stable.
	listLen, setListLen := reactive.NewSignal(0)
	var messages []chatMsg

	generating, setGenerating := reactive.NewSignal(false)
	errMsg, setErrMsg := reactive.NewSignal("")

	var chatList dom.Element
	var composerInput dom.Element

	persist := func() {
		store.Set(storageKeyURL, baseURL())
		store.Set(storageKeyKey, apiKey())
		store.Set(storageKeyModel, model())
		store.Set(storageKeySystem, systemPrompt())
	}

	persistModelIDCache := func() {
		if len(modelOptions) == 0 {
			store.Remove(storageKeyModelIDs)
			store.Remove(storageKeyModelIDsForURL)
			return
		}
		raw, err := json.Marshal(modelOptions)
		if err != nil {
			return
		}
		store.Set(storageKeyModelIDs, string(raw))
		store.Set(storageKeyModelIDsForURL, strings.TrimSpace(baseURL()))
	}

	appendAssistantPlaceholder := func() {
		messages = append(messages, chatMsg{
			Role:      "assistant",
			Content:   "",
			Thinking:  "",
			Streaming: true,
			Timestamp: time.Now().Format("15:04:05"),
		})
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
		messages = append(messages, chatMsg{
			Role:      "user",
			Content:   text,
			Timestamp: time.Now().Format("15:04:05"),
		})
		appendAssistantPlaceholder()

		history := make([]apiMessage, 0, len(messages)+1)
		if sys := strings.TrimSpace(systemPrompt()); sys != "" {
			history = append(history, apiMessage{Role: "system", Content: sys})
		}
		for _, m := range messages {
			if m.Role == "assistant" && m.Streaming {
				continue
			}
			history = append(history, apiMessage{Role: m.Role, Content: m.Content})
		}

		setGenerating(true)
		modelName := model()
		go func() {
			chatURL, err := normalizeChatCompletionsURL(baseURL())
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

			err = streamChat(chatURL, apiKey(), modelName, history, func(rDelta, cDelta string) bool {
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
				// Merge inline <think> tags from content stream into thinking panel
				if extraTh, rest := splitThinkingFromContent(co); extraTh != "" {
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
			if extraTh, rest := splitThinkingFromContent(co); extraTh != "" {
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
		setCanScroll(false)
		if !composerInput.IsNull() {
			composerInput.SetProperty("value", "")
			composerInput.Focus()
		}
		if !chatList.IsNull() {
			chatList.SetProperty("scrollTop", 0)
		}
	}

	loadModels := func() {
		u := strings.TrimSpace(baseURL())
		if u == "" {
			setModelsErr("Enter the API URL first.")
			return
		}
		modelsURL, err := deriveModelsEndpoint(u)
		if err != nil {
			setModelsErr(err.Error())
			return
		}
		go func() {
			setModelsLoading(true)
			setModelsErr("")
			key := apiKey()
			ids, err := fetchModelsList(modelsURL, key)
			setModelsLoading(false)
			if err != nil {
				setModelsErr(err.Error())
				modelOptions = nil
				persistModelIDCache()
			} else {
				modelOptions = ids
				persistModelIDCache()
				cur := model()
				found := false
				for _, id := range ids {
					if id == cur {
						found = true
						break
					}
				}
				if !found && len(ids) > 0 {
					setModel(ids[0])
					persist()
				} else {
					persist()
				}
			}
			setModelsVer(modelsVer() + 1)
		}()
	}

	closeSettings := func() {
		setShowSettings(false)
		persist()
	}

	openSettings := func() {
		setShowSettings(true)
		setModelsErr("")
		loadModels()
	}

	// Write current URL, key, and model to localStorage (including defaults on first run).
	persist()

	return el.Div(
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
					el.Button(
						el.Type("button"),
						el.Class("text-xs px-3 py-1.5 rounded-lg border border-zinc-700 text-zinc-300 hover:bg-zinc-800/80 transition-colors"),
						el.Text("Settings"),
						el.OnClick(func(dom.Event) {
							if showSettings() {
								closeSettings()
							} else {
								openSettings()
							}
						}),
					),
					el.Button(
						el.Type("button"),
						el.Class("text-xs px-3 py-1.5 rounded-lg border border-zinc-700 text-zinc-400 hover:text-red-400 hover:border-red-900/50 transition-colors"),
						el.Text("Clear"),
						el.OnClick(func(dom.Event) {
							clearChat()
						}),
					),
				),
			),
		),

		el.Show(
			func() bool { return errMsg() != "" },
			el.Div(
				el.Class("shrink-0 max-w-3xl mx-auto w-full px-4 pt-3"),
				el.Div(
					el.Class("text-xs rounded-lg px-3 py-2 bg-red-950/50 text-red-300 border border-red-900/40"),
					el.DynText(errMsg),
				),
			),
		),

		// Main Content Area
		el.Div(
			el.Class("flex-1 relative min-h-0"),

			// Messages
			el.Div(
				el.Class("absolute inset-0 overflow-y-auto px-4 py-6 pb-48"),
				el.Ref(&chatList),
				el.On("scroll", func(dom.Event) {
					if !chatList.IsNull() {
						scrollHeight := chatList.GetProperty("scrollHeight").Int()
						clientHeight := chatList.GetProperty("clientHeight").Int()

						setCanScroll(scrollHeight > clientHeight+100)
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
						func(userIdx int, _ int) string {
							return strconv.Itoa(userIdx)
						},
						func(userIdx int, _ int) el.Node {
							return turnGroup(userIdx, &messages, msgVer)
						},
					),
				),
				el.OnMount(func(element dom.Element) {
					reactive.CreateEffect(func() {
						_ = msgVer()
						_ = generating()
						// Only auto-scroll to bottom if the user is not manually scrolling up
						// (Simple heuristic: if they're near the bottom already)
						scrollPos := element.GetProperty("scrollTop").Int()
						scrollHeight := element.GetProperty("scrollHeight").Int()
						clientHeight := element.GetProperty("clientHeight").Int()
						if scrollHeight-scrollPos-clientHeight < 300 {
							element.SetProperty("scrollTop", scrollHeight)
						}

						setCanScroll(scrollHeight > clientHeight+100)
					})
				}),
			),

			// Floating Scroll button - now a rectangular green bubble
			el.Show(
				func() bool { return canScroll() },
				el.Button(
					el.Type("button"),
					el.Class("absolute bottom-30 right-6 md:right-10 px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white flex items-center gap-2 shadow-lg z-30 transition-all text-xs font-medium"),
					el.OnClick(func(dom.Event) {
						if !chatList.IsNull() {
							scrollTop := chatList.GetProperty("scrollTop").Int()
							scrollHeight := chatList.GetProperty("scrollHeight").Int()
							clientHeight := chatList.GetProperty("clientHeight").Int()

							// If we are closer to the bottom, go to top. Otherwise go to bottom.
							if scrollTop > (scrollHeight-clientHeight)/2 {
								chatList.SetProperty("scrollTop", 0)
							} else {
								chatList.SetProperty("scrollTop", scrollHeight)
							}
						}
					}),
					el.SVGTag("svg",
						el.Attr("viewBox", "0 0 20 20"),
						el.Attr("fill", "currentColor"),
						el.DynAttr("class", func() string {
							scrollTop := 0
							scrollHeight := 0
							clientHeight := 0
							if !chatList.IsNull() {
								scrollTop = chatList.GetProperty("scrollTop").Int()
								scrollHeight = chatList.GetProperty("scrollHeight").Int()
								clientHeight = chatList.GetProperty("clientHeight").Int()
							}

							// If we are in the bottom half, show UP arrow (to go to first)
							if scrollTop > (scrollHeight-clientHeight)/2 {
								return "w-4 h-4 transition-transform" // Default is UP
							}
							// In top half, show DOWN arrow (to go to last)
							return "w-4 h-4 rotate-180 transition-transform"
						}),
						el.SVGTag("path", el.Attr("d", "M14.77 12.79a.75.75 0 01-1.06-.02L10 8.832 6.29 12.77a.75.75 0 11-1.08-1.04l4.25-4.5a.75.75 0 011.08 0l4.25 4.5a.75.75 0 01-.02 1.06z"), el.Attr("clip-rule", "evenodd")),
					),
					el.DynText(func() string {
						scrollTop := 0
						scrollHeight := 0
						clientHeight := 0
						if !chatList.IsNull() {
							scrollTop = chatList.GetProperty("scrollTop").Int()
							scrollHeight = chatList.GetProperty("scrollHeight").Int()
							clientHeight = chatList.GetProperty("clientHeight").Int()
						}

						if scrollTop > (scrollHeight-clientHeight)/2 {
							return "go to first message"
						}
						return "go to last message"
					}),
				),
			),

			// Composer
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
							el.OnInput(func(e dom.Event) {
								setInput(e.TargetValue())
							}),
							el.OnKeyDown(func(e dom.Event) {
								if e.Key() == "Enter" && !e.Value.Get("shiftKey").Bool() {
									e.PreventDefault()
									send()
								}
							}),
						),
						el.Div(
							el.Class("flex flex-col gap-2"),
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
							el.Show(
								func() bool { return generating() },
								el.Button(
									el.Type("button"),
									el.Class("px-4 py-2 rounded-xl text-xs border border-zinc-700 text-zinc-400 hover:bg-zinc-800"),
									el.Text("Stop"),
									el.OnClick(func(dom.Event) { stop() }),
								),
							),
						),
					),
				),
			),
		),

		// Settings modal: last in tree + inline z-index; toggled with Tailwind hidden (avoids el.Show mount quirks).
		el.Div(
			el.Style("z-index", "100"),
			el.DynClass(func() string {
				if !showSettings() {
					return "hidden"
				}
				return "fixed inset-0"
			}),
			el.Div(
				el.Class("absolute inset-0 bg-black/70"),
				el.OnClick(func(dom.Event) { closeSettings() }),
			),
			el.Div(
				el.Class("absolute inset-0 flex items-center justify-center p-4 pointer-events-none"),
				el.Div(
					el.Class("pointer-events-auto w-full max-w-lg rounded-xl border border-zinc-700/80 bg-zinc-900 shadow-2xl shadow-black/60"),
					el.OnClick(func(e dom.Event) { e.StopPropagation() }),
					el.Div(
						el.Class("px-5 py-4 border-b border-zinc-800 flex items-center justify-between gap-3"),
						el.Div(
							el.Class("text-sm font-semibold text-zinc-100"),
							el.Text("Provider"),
						),
						el.Button(
							el.Type("button"),
							el.Class("text-zinc-500 hover:text-zinc-300 text-lg leading-none px-2 py-1 rounded-lg hover:bg-zinc-800"),
							el.Text("×"),
							el.Attr("aria-label", "Close"),
							el.OnClick(func(dom.Event) { closeSettings() }),
						),
					),
					el.Div(
						el.Class("px-5 py-4 space-y-4 max-h-[min(70vh,520px)] overflow-y-auto"),
						el.Div(
							el.Label(
								el.Class("block text-[11px] uppercase tracking-wider text-zinc-500 mb-1.5"),
								el.Text("Chat completions URL"),
							),
							el.P(
								el.Class("text-[11px] text-zinc-600 mb-1.5"),
								el.Text("Tab or click away from this field to load models from the provider."),
							),
							el.Input(
								el.Class("w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-200 placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-cyan-500/40"),
								el.Type("text"),
								el.Placeholder("https://api.openai.com/v1/chat/completions or http://127.0.0.1:1234/v1"),
								el.DynProp("value", func() any { return baseURL() }),
								el.OnInput(func(e dom.Event) {
									next := e.TargetValue()
									setBaseURL(next)
									if strings.TrimSpace(next) != store.Get(storageKeyModelIDsForURL) {
										modelOptions = nil
										setModelsVer(modelsVer() + 1)
										persistModelIDCache()
									}
									persist()
								}),
								el.OnBlur(func(dom.Event) { loadModels() }),
							),
						),
						el.Div(
							el.Label(
								el.Class("block text-[11px] uppercase tracking-wider text-zinc-500 mb-1.5"),
								el.Text("API key (optional)"),
							),
							el.Input(
								el.Class("w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-200 placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-cyan-500/40"),
								el.Type("password"),
								el.Placeholder("Bearer token if required"),
								el.DynProp("value", func() any { return apiKey() }),
								el.OnInput(func(e dom.Event) {
									setAPIKey(e.TargetValue())
									persist()
								}),
								el.OnBlur(func(dom.Event) { loadModels() }),
							),
						),
						el.Div(
							el.Label(
								el.Class("block text-[11px] uppercase tracking-wider text-zinc-500 mb-1.5"),
								el.Text("System Prompt"),
							),
							el.Textarea(
								el.Class("w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-200 placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-cyan-500/40 min-h-[80px]"),
								el.Placeholder("You are a helpful assistant."),
								el.DynProp("value", func() any { return systemPrompt() }),
								el.OnInput(func(e dom.Event) {
									setSystemPrompt(e.TargetValue())
									persist()
								}),
							),
						),
						el.Div(
							el.Label(
								el.Class("block text-[11px] uppercase tracking-wider text-zinc-500 mb-1.5"),
								el.Text("Model"),
							),
							el.Dynamic(func() el.Node {
								_ = modelsVer()
								_ = model()
								mods := []any{
									el.Class("w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm text-zinc-200 focus:outline-none focus:ring-1 focus:ring-cyan-500/40"),
									el.DynProp("value", func() any { return model() }),
									el.DynAttr("disabled", func() string {
										if modelsLoading() {
											return "true"
										}
										return ""
									}),
									el.OnChange(func(e dom.Event) {
										setModel(e.TargetValue())
										persist()
									}),
								}
								if len(modelOptions) == 0 {
									mods = append(mods, el.Option(
										el.Value(""),
										el.Text("— Load models: blur the URL or API key field —"),
									))
								} else {
									for _, id := range modelOptions {
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
									if !modelsLoading() {
										return "hidden"
									}
									return "text-xs text-cyan-500/90 mt-2 flex items-center gap-2"
								}),
								el.Span(el.Class("inline-block w-3 h-3 border-2 border-cyan-500/30 border-t-cyan-500 rounded-full animate-spin")),
								el.Text("Loading models…"),
							),
							el.Div(
								el.DynClass(func() string {
									if modelsErr() == "" || modelsLoading() {
										return "hidden"
									}
									return "text-xs text-red-400/90 mt-2"
								}),
								el.DynText(modelsErr),
							),
						),
						el.Div(
							el.Class("flex justify-end gap-2 pt-2"),
							el.Button(
								el.Type("button"),
								el.Class("px-4 py-2 rounded-lg text-sm border border-zinc-700 text-zinc-300 hover:bg-zinc-800"),
								el.Text("Cancel"),
								el.OnClick(func(dom.Event) { closeSettings() }),
							),
							el.Button(
								el.Type("button"),
								el.Class("px-4 py-2 rounded-lg text-sm font-medium bg-cyan-600 hover:bg-cyan-500 text-zinc-950"),
								el.Text("Done"),
								el.OnClick(func(dom.Event) { closeSettings() }),
							),
						),
					),
				),
			),
		),
	)
}

// turnGroup renders one chat turn: the user message with the assistant reply directly
// beneath it, so requests are not visually separated from their responses.
func turnGroup(userIdx int, all *[]chatMsg, ver func() int) el.Node {
	if userIdx < 0 || userIdx >= len(*all) || (*all)[userIdx].Role != "user" {
		return el.Div()
	}
	return el.Div(
		el.Class("flex flex-col gap-3 w-full min-w-0"),
		messageBubble((*all)[userIdx], userIdx, all, ver),
		el.Show(func() bool {
			_ = ver()
			return userIdx+1 < len(*all) && (*all)[userIdx+1].Role == "assistant"
		}, el.Dynamic(func() el.Node {
			return messageBubble((*all)[userIdx+1], userIdx+1, all, ver)
		})),
	)
}

func messageBubble(item chatMsg, i int, all *[]chatMsg, ver func() int) el.Node {
	copied, setCopied := reactive.NewSignal(false)

	// Branch on item.Role only (snapshot when this row is first rendered). Do not
	// read ver() here or el.For would subscribe to msgVer and re-run the whole list
	// on every streaming token, breaking inner effects (e.g. thinking "done" state).
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
									// Still thinking if streaming AND content is empty
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
						markdownBlock(func() string {
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
							copyToClipboard((*all)[i].Content)
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
					markdownBlock(func() string {
						_ = ver()
						if i >= len(*all) {
							return ""
						}
						return (*all)[i].Content
					}),
				),
				el.Div(
					el.Class("absolute bottom-2 right-3 flex items-center gap-2 text-[10px] text-zinc-500 font-medium"),
					// Model
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
					// Pipe
					el.Show(func() bool {
						_ = ver()
						if i >= len(*all) {
							return false
						}
						return (*all)[i].Model != "" && (*all)[i].TFT != ""
					},
						el.Span(el.Class("opacity-30"), el.Text("|")),
					),
					// TFT
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
					// Pipe
					el.Show(func() bool {
						_ = ver()
						if i >= len(*all) {
							return false
						}
						return (*all)[i].TFT != "" && (*all)[i].TPS != ""
					},
						el.Span(el.Class("opacity-30"), el.Text("|")),
					),
					// TPS
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
					// Pipe
					el.Show(func() bool {
						_ = ver()
						if i >= len(*all) {
							return false
						}
						return (*all)[i].TPS != "" && (*all)[i].Duration != ""
					},
						el.Span(el.Class("opacity-30"), el.Text("|")),
					),
					// Duration
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
					// Pipe
					el.Show(func() bool {
						_ = ver()
						if i >= len(*all) {
							return false
						}
						return (*all)[i].Duration != ""
					},
						el.Span(el.Class("opacity-30"), el.Text("|")),
					),
					// Time
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
