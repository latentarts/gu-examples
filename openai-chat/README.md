# OpenAI-Compatible Chat

Browser-based chat client for OpenAI-compatible `/v1/chat/completions` APIs, built with `gu` and Go WASM. It stores the base URL, API key, model selection, and system prompt locally in the browser, streams responses token-by-token, and exposes reasoning content when the upstream model returns it.

This example talks to the API directly from the browser. The upstream endpoint therefore must be reachable from browser JavaScript, allow CORS for the page origin, and handle any required `OPTIONS` preflight. If it does not, use a same-origin proxy.

## Purpose

This example demonstrates how to build a fully local chat UI in Go that talks directly to an OpenAI-style HTTP API from the browser. It is useful for testing self-hosted, proxy, or third-party providers that implement the chat completions and models endpoints.

## How It Works

The UI is rendered with `gu` in [`main.go`](./main.go). Configuration is loaded from browser storage on startup, the app normalizes the configured API URL into a canonical chat-completions endpoint, and model discovery hits the sibling `/models` endpoint derived from that URL.

Chat requests use the browser `fetch` API with server-sent event streaming. The app incrementally parses `data:` lines from the response body, extracts either reasoning or assistant deltas, and updates the current assistant message in place as tokens arrive.

## gu Implementation Details

The example uses fine-grained signals for chat configuration and UI state:

```go
loading, setLoading := reactive.NewSignal(false)
generating, setGenerating := reactive.NewSignal(false)
errMsg, setErrMsg := reactive.NewSignal("")
msgVer, setMsgVer := reactive.NewSignal(0)
```

The message list uses the version-counter pattern so a plain Go slice can remain outside the reactive graph while still triggering re-renders:

```go
var messages []chatMsg
msgVer, setMsgVer := reactive.NewSignal(0)

messages = append(messages, chatMsg{Role: "user", Content: text})
setMsgVer(msgVer() + 1)
```

Streaming updates are applied incrementally from the event stream parser:

```go
if !onDelta(r, c) {
	return nil
}
```

## Developer Guidance

- Use `make test` to run the example’s WASM tests with Go’s wasm runner.
- The URL normalization helpers are covered by unit tests and are the safest place to extend provider compatibility.
- Browser storage access is runtime-only; pure URL and parsing helpers are the most reliable parts to test outside full UI interaction.

## Run It

```sh
make serve
```

Open the local server printed by the Makefile, enter a compatible base URL and API key, then start chatting.

If the UI reports `Failed to fetch`, the request was blocked before any HTTP response came back. The most common causes are:

- The API does not allow CORS from your page origin.
- The API rejects the browser's `OPTIONS` preflight for the `Authorization` header.
- The page is served over `https://` but the API URL is `http://`.
- The API host, port, or TLS certificate is not reachable from the browser.
