//go:build js && wasm

package main

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateCandlesAndRenderChart(t *testing.T) {
	quotes := GenerateCandles("AAPL", 8)
	if len(quotes) != 8 {
		t.Fatalf("expected 8 candles, got %d", len(quotes))
	}
	if quotes[0].Time.IsZero() {
		t.Fatal("expected generated candles to include timestamps")
	}

	html := RenderChartHTML(quotes)
	if !strings.Contains(html, "<svg") {
		t.Fatal("expected chart HTML to contain svg markup")
	}
}

func TestNextCandleAdvancesTime(t *testing.T) {
	rng := newRand(simpleHash("AAPL"))
	prev := StockQuote{
		Open:  100,
		High:  102,
		Low:   99,
		Close: 101,
		Time:  time.Date(2025, 1, 2, 9, 30, 0, 0, time.UTC),
	}

	next := NextCandle(prev, rng)
	if !next.Time.After(prev.Time) {
		t.Fatal("expected next candle time to advance")
	}
}
