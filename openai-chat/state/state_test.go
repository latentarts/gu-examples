//go:build js && wasm

package state

import (
	"errors"
	"net/url"
	"testing"
)

func TestSplitThinkingFromContent(t *testing.T) {
	thinking, rest := SplitThinkingFromContent("<think>plan</think>\nfinal")
	if thinking != "plan" {
		t.Fatalf("expected thinking content to be extracted, got %q", thinking)
	}
	if rest != "final" {
		t.Fatalf("expected remaining content to be preserved, got %q", rest)
	}
}

func TestNormalizeChatCompletionsURL(t *testing.T) {
	got, err := NormalizeChatCompletionsURL("https://api.example.com/v1")
	if err != nil {
		t.Fatalf("NormalizeChatCompletionsURL returned error: %v", err)
	}
	want := "https://api.example.com/v1/chat/completions"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestDeriveModelsEndpoint(t *testing.T) {
	got, err := DeriveModelsEndpoint("https://api.example.com/v1/chat/completions")
	if err != nil {
		t.Fatalf("DeriveModelsEndpoint returned error: %v", err)
	}
	want := "https://api.example.com/v1/models"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSameOrigin(t *testing.T) {
	a, _ := url.Parse("https://example.com/v1/chat/completions")
	b, _ := url.Parse("https://example.com/v1/models")
	c, _ := url.Parse("http://example.com/v1/models")

	if !sameOrigin(a, b) {
		t.Fatalf("expected matching scheme+host to be same-origin")
	}
	if sameOrigin(a, c) {
		t.Fatalf("expected differing schemes to be cross-origin")
	}
}

func TestExplainFetchFailurePassesThroughOtherErrors(t *testing.T) {
	src := "some other error"
	err := explainFetchFailure("https://api.example.com/v1/chat/completions", errors.New(src))
	if err == nil || err.Error() != src {
		t.Fatalf("expected non-fetch errors to pass through unchanged, got %v", err)
	}
}
