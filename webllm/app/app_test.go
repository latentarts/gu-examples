//go:build js && wasm

package app

import (
	"testing"

	"github.com/latentarts/gu-examples/webllm/components"
)

func TestRenderMarkdownEmpty(t *testing.T) {
	if got := components.RenderMarkdown(""); got != "" {
		t.Fatalf("expected empty markdown render result, got %q", got)
	}
}

func TestMarkdownBlockConstructs(t *testing.T) {
	if components.MarkdownBlock(func() string { return "" }) == nil {
		t.Fatal("markdownBlock returned nil")
	}
}
