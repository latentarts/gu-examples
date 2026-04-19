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
		"Cache-Control": "no-cache",
	}
	if apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}

	init := js.Global().Get("Object").New()
	init.Set("method", "POST")
	init.Set("body", string(raw))
	init.Set("headers", HeadersToJSObject(headers))

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
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
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
		return nil, err
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
