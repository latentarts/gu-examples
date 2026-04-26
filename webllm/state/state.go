//go:build js && wasm

package state

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
)

type ChatMsg struct {
	Role      string
	Content   string
	Streaming bool
	Timestamp string
	Model     string
}

type APIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatState struct {
	Store jsutil.Storage

	Model           func() string
	SetModel        func(string)
	SystemPrompt    func() string
	SetSystemPrompt func(string)
	Loading         func() bool
	SetLoading      func(bool)
	Progress        func() string
	SetProgress     func(string)
	EngineReady     func() bool
	SetEngineReady  func(bool)
	Generating      func() bool
	SetGenerating   func(bool)
	ErrMsg          func() string
	SetErrMsg       func(string)
	ShowSettings    func() bool
	SetShowSettings func(bool)
	Input           func() string
	SetInput        func(string)
	MsgVer          func() int
	SetMsgVer       func(int)

	Messages []ChatMsg
}

func NewChatState() *ChatState {
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

	return &ChatState{
		Store:           store,
		Model:           model,
		SetModel:        setModel,
		SystemPrompt:    systemPrompt,
		SetSystemPrompt: setSystemPrompt,
		Loading:         loading,
		SetLoading:      setLoading,
		Progress:        progress,
		SetProgress:     setProgress,
		EngineReady:     engineReady,
		SetEngineReady:  setEngineReady,
		Generating:      generating,
		SetGenerating:   setGenerating,
		ErrMsg:          errMsg,
		SetErrMsg:       setErrMsg,
		ShowSettings:    showSettings,
		SetShowSettings: setShowSettings,
		Input:           input,
		SetInput:        setInput,
		MsgVer:          msgVer,
		SetMsgVer:       setMsgVer,
	}
}

func (s *ChatState) InitEngine(m string) {
	s.SetLoading(true)
	s.SetEngineReady(false)
	s.SetErrMsg("")
	s.SetProgress("Initializing...")

	go func() {
		progressCb := js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) > 0 {
				s.SetProgress(args[0].String())
			}
			return nil
		})
		js.Global().Get("App").Set("onProgress", progressCb)

		promise := js.Global().Get("App").Call("initEngine", m)
		_, err := jsutil.Await(promise)
		progressCb.Release()

		if err != nil {
			s.SetErrMsg(fmt.Sprintf("Failed to load model: %v", err))
			s.SetLoading(false)
			return
		}

		s.SetLoading(false)
		s.SetEngineReady(true)
		s.Store.Set("webllm-model", m)
	}()
}

func (s *ChatState) SendMessage() {
	text := s.Input()
	if text == "" || s.Generating() || !s.EngineReady() {
		return
	}
	s.SetInput("")

	ts := time.Now().Format("15:04:05")
	s.Messages = append(s.Messages, ChatMsg{
		Role:      "user",
		Content:   text,
		Timestamp: ts,
		Model:     s.Model(),
	})

	assistantIdx := len(s.Messages)
	s.Messages = append(s.Messages, ChatMsg{
		Role:      "assistant",
		Content:   "",
		Streaming: true,
		Timestamp: ts,
		Model:     s.Model(),
	})

	s.SetMsgVer(s.MsgVer() + 1)
	s.SetGenerating(true)

	go func() {
		payload := []APIMessage{{Role: "system", Content: s.SystemPrompt()}}
		for i, m := range s.Messages {
			if i < assistantIdx && m.Content != "" {
				payload = append(payload, APIMessage{Role: m.Role, Content: m.Content})
			}
		}

		data, _ := json.Marshal(payload)
		onToken := js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) > 0 {
				s.Messages[assistantIdx].Content = args[0].String()
				s.SetMsgVer(s.MsgVer() + 1)
			}
			return nil
		})

		promise := js.Global().Get("App").Call("chat", string(data), onToken)
		_, err := jsutil.Await(promise)
		onToken.Release()

		if err != nil {
			s.SetErrMsg(fmt.Sprintf("Inference failed: %v", err))
		}

		s.Messages[assistantIdx].Streaming = false
		s.SetMsgVer(s.MsgVer() + 1)
		s.SetGenerating(false)
	}()
}
