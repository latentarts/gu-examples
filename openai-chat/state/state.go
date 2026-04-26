//go:build js && wasm

package state

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"syscall/js"
	"time"

	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
)

const (
	StorageKeyURL            = "openai-chat-base-url"
	StorageKeyKey            = "openai-chat-api-key"
	StorageKeyModel          = "openai-chat-model"
	StorageKeyModelIDs       = "openai-chat-model-ids-json"
	StorageKeyModelIDsForURL = "openai-chat-model-ids-for-url"
	StorageKeySystem         = "openai-chat-system-prompt"
)

type ChatMsg struct {
	Role      string
	Content   string
	Thinking  string
	Streaming bool
	Timestamp string
	Model     string
	TFT       string
	TPS       string
	Duration  string
}

type APIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type StreamDelta struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			Reasoning        string `json:"reasoning"`
		} `json:"delta"`
	} `json:"choices"`
}

type modelsListResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

var ThinkingTagRE = regexp.MustCompile(`(?s)<(?:think|thinking)>(.*?)</(?:think|thinking)>`)

type ChatState struct {
	BaseURL      func() string
	SetBaseURL   func(string)
	APIKey       func() string
	SetAPIKey    func(string)
	Model        func() string
	SetModel     func(string)
	SystemPrompt func() string
	SetSystemPrompt func(string)
	
	ShowSettings    func() bool
	SetShowSettings func(bool)
	
	ModelsLoading    func() bool
	SetModelsLoading func(bool)
	ModelsErr        func() bool
	SetModelsErr     func(string)
	ModelsVer        func() int
	SetModelsVer     func(int)
	ModelOptions     []string

	Messages []ChatMsg
	MsgVer   func() int
	SetMsgVer func(int)
	ListLen  func() int
	SetListLen func(int)
	
	Generating    func() bool
	SetGenerating func(bool)
	ErrMsg        func() string
	SetErrMsg     func(string)
	
	Input    func() string
	SetInput func(string)
}

func NewChatState() *ChatState {
	store := jsutil.LocalStorage()
	
	baseURL, setBaseURL := reactive.NewSignal(store.Get(StorageKeyURL))
	if baseURL() == "" {
		setBaseURL("https://api.openai.com/v1")
	}
	apiKey, setAPIKey := reactive.NewSignal(store.Get(StorageKeyKey))
	model, setModel := reactive.NewSignal(store.Get(StorageKeyModel))
	if model() == "" {
		setModel("gpt-4o-mini")
	}
	systemPrompt, setSystemPrompt := reactive.NewSignal(store.Get(StorageKeySystem))
	if systemPrompt() == "" {
		setSystemPrompt("You are a helpful assistant.")
	}
	
	showSettings, setShowSettings := reactive.NewSignal(false)
	modelsLoading, setModelsLoading := reactive.NewSignal(false)
	modelsErr, setModelsErr := reactive.NewSignal("")
	modelsVer, setModelsVer := reactive.NewSignal(0)
	
	msgVer, setMsgVer := reactive.NewSignal(0)
	listLen, setListLen := reactive.NewSignal(0)
	generating, setGenerating := reactive.NewSignal(false)
	errMsg, setErrMsg := reactive.NewSignal("")
	input, setInput := reactive.NewSignal("")

	s := &ChatState{
		BaseURL:      baseURL,
		SetBaseURL:   setBaseURL,
		APIKey:       apiKey,
		SetAPIKey:    setAPIKey,
		Model:        model,
		SetModel:     setModel,
		SystemPrompt: systemPrompt,
		SetSystemPrompt: setSystemPrompt,
		
		ShowSettings:    showSettings,
		SetShowSettings: setShowSettings,
		
		ModelsLoading:    modelsLoading,
		SetModelsLoading: setModelsLoading,
		ModelsErr:        func() bool { return modelsErr() != "" },
		SetModelsErr:     setModelsErr,
		ModelsVer:        modelsVer,
		SetModelsVer:     setModelsVer,

		MsgVer:   msgVer,
		SetMsgVer: setMsgVer,
		ListLen:  listLen,
		SetListLen: setListLen,
		Generating:    generating,
		SetGenerating: setGenerating,
		ErrMsg:        errMsg,
		SetErrMsg:     setErrMsg,
		Input:    input,
		SetInput: setInput,
	}
	
	// Initial model options load from storage
	if store.Get(StorageKeyModelIDsForURL) == strings.TrimSpace(baseURL()) {
		if raw := store.Get(StorageKeyModelIDs); raw != "" {
			var ids []string
			if err := json.Unmarshal([]byte(raw), &ids); err == nil && len(ids) > 0 {
				s.ModelOptions = ids
				setModelsVer(1)
			}
		}
	}
	
	return s
}

func (s *ChatState) Persist() {
	store := jsutil.LocalStorage()
	store.Set(StorageKeyURL, s.BaseURL())
	store.Set(StorageKeyKey, s.APIKey())
	store.Set(StorageKeyModel, s.Model())
	store.Set(StorageKeySystem, s.SystemPrompt())
}

func (s *ChatState) PersistModelIDCache() {
	store := jsutil.LocalStorage()
	if len(s.ModelOptions) == 0 {
		store.Remove(StorageKeyModelIDs)
		store.Remove(StorageKeyModelIDsForURL)
		return
	}
	raw, err := json.Marshal(s.ModelOptions)
	if err != nil {
		return
	}
	store.Set(StorageKeyModelIDs, string(raw))
	store.Set(StorageKeyModelIDsForURL, strings.TrimSpace(s.BaseURL()))
}

func SplitThinkingFromContent(s string) (thinking, rest string) {
	s = strings.TrimSpace(s)
	if m := ThinkingTagRE.FindStringSubmatch(s); len(m) > 1 {
		th := strings.TrimSpace(m[1])
		rest = strings.TrimSpace(ThinkingTagRE.ReplaceAllString(s, ""))
		return th, rest
	}
	return "", s
}

func HeadersToJSObject(h map[string]string) js.Value {
	obj := js.Global().Get("Object").New()
	for k, v := range h {
		obj.Set(k, v)
	}
	return obj
}

func explainFetchFailure(targetURL string, err error) error {
	msg := ""
	if err != nil {
		msg = strings.TrimSpace(err.Error())
	}
	if msg == "" {
		msg = "fetch failed before an HTTP response was returned"
	}
	lower := strings.ToLower(msg)
	if !strings.Contains(lower, "failed to fetch") && !strings.Contains(lower, "networkerror") {
		return err
	}

	parsedTarget, parseErr := url.Parse(strings.TrimSpace(targetURL))
	if parseErr != nil {
		return fmt.Errorf("%s. The configured API URL could not be parsed: %v", msg, parseErr)
	}

	pageURL := js.Global().Get("location").Get("href").String()
	parsedPage, _ := url.Parse(pageURL)
	hints := []string{
		"The browser blocked the request before any HTTP response was available.",
	}
	if parsedPage != nil && parsedPage.Scheme == "https" && parsedTarget.Scheme == "http" {
		hints = append(hints, "This page is loaded over HTTPS but the API URL is HTTP, so the browser treats it as mixed content.")
	}
	if parsedPage != nil && parsedPage.Host != "" && parsedTarget.Host != "" && !sameOrigin(parsedPage, parsedTarget) {
		hints = append(hints, "The API is cross-origin from this page, so the server must allow CORS and any required OPTIONS preflight.")
	}
	if parsedTarget.Host == "" {
		hints = append(hints, "The API URL is missing a host.")
	}
	hints = append(hints, "Verify the host, port, TLS certificate, and that the API endpoint is reachable from the browser devtools Network tab.")
	hints = append(hints, "If your provider does not expose CORS, call it through a same-origin proxy instead of directly from the browser.")

	return fmt.Errorf("%s. %s", msg, strings.Join(hints, " "))
}

func sameOrigin(a, b *url.URL) bool {
	if a == nil || b == nil {
		return false
	}
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}

func StreamChat(
	url string,
	apiKey string,
	model string,
	history []APIMessage,
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
	}
	if apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}

	ac := js.Global().Get("AbortController").New()
	
	// Create the init object for fetch
	init := js.Global().Get("Object").New()
	init.Set("method", "POST")
	init.Set("body", string(raw))
	init.Set("headers", HeadersToJSObject(headers))
	init.Set("signal", ac.Get("signal"))

	// Store bound abort function globally for manual interruption
	abortFunc := ac.Get("abort").Call("bind", ac)
	abortHolder := js.Global().Get("Object").New()
	abortHolder.Set("abort", abortFunc)
	js.Global().Set("__openaiChatAbort", abortHolder)

	promise := js.Global().Call("fetch", url, init)
	respVal, err := jsutil.Await(promise)
	if err != nil {
		return explainFetchFailure(url, err)
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

	parseLine := func(payload string) bool {
		if payload == "[DONE]" {
			return false
		}
		var sd StreamDelta
		if err := json.Unmarshal([]byte(payload), &sd); err != nil || len(sd.Choices) == 0 {
			return true
		}
		d := sd.Choices[0].Delta
		r := d.ReasoningContent
		if r == "" {
			r = d.Reasoning
		}
		c := d.Content
		if r == "" && c == "" {
			return true
		}
		return onDelta(r, c)
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
			line := strings.TrimRight(data[:idx], "\r")
			lineBuf.Reset()
			lineBuf.WriteString(data[idx+1:])
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}
			if !parseLine(strings.TrimSpace(strings.TrimPrefix(line, "data: "))) {
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
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		if !parseLine(strings.TrimSpace(strings.TrimPrefix(line, "data: "))) {
			return nil
		}
	}

	return nil
}

func DeriveModelsEndpoint(chatURL string) (string, error) {
	normalized, err := NormalizeChatCompletionsURL(chatURL)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(normalized)
	if err != nil {
		return "", err
	}
	u.Path = strings.TrimSuffix(u.Path, "/chat/completions") + "/models"
	return u.String(), nil
}

func NormalizeChatCompletionsURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("missing base URL")
	}
	
	// Add scheme if missing
	if !strings.Contains(raw, "://") {
		// Use http for localhost by default, https for others
		if strings.HasPrefix(raw, "localhost") || strings.HasPrefix(raw, "127.0.0.1") {
			raw = "http://" + raw
		} else {
			raw = "https://" + raw
		}
	}
	
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid URL: missing host")
	}
	path := strings.TrimRight(u.Path, "/")
	switch {
	case path == "":
		path = "/v1/chat/completions"
	case strings.HasSuffix(path, "/chat/completions"):
	case strings.HasSuffix(path, "/v1"):
		path += "/chat/completions"
	default:
		path += "/chat/completions"
	}
	u.Path = path
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func FetchModelsList(modelsURL, apiKey string) ([]string, error) {
	headers := map[string]string{"Accept": "application/json"}
	if apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}
	
	init := js.Global().Get("Object").New()
	init.Set("method", "GET")
	init.Set("headers", HeadersToJSObject(headers))
	
	respVal, err := jsutil.Await(js.Global().Call("fetch", modelsURL, init))
	if err != nil {
		return nil, explainFetchFailure(modelsURL, err)
	}
	bodyVal, _ := jsutil.Await(respVal.Call("text"))
	body := bodyVal.String()
	if !respVal.Get("ok").Bool() {
		return nil, fmt.Errorf("HTTP %d: %s", respVal.Get("status").Int(), body)
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

func NowTimestamp() string {
	return time.Now().Format("15:04:05")
}
