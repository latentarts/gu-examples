//go:build js && wasm

package main

import (
	"fmt"
	"math"
	"strings"
)

// chartConfig holds layout constants for the SVG chart.
type chartConfig struct {
	Width     float64
	Height    float64
	PadTop    float64
	PadRight  float64
	PadBottom float64
	PadLeft   float64
	CandleGap float64
}

func defaultChartConfig() chartConfig {
	return chartConfig{
		Width:     800,
		Height:    400,
		PadTop:    20,
		PadRight:  70,
		PadBottom: 30,
		PadLeft:   10,
		CandleGap: 2,
	}
}

// RenderChartHTML returns the SVG markup string for the given quotes.
// SVG elements require createElementNS which the framework doesn't use,
// so we build the markup as a string for injection via innerHTML.
func RenderChartHTML(quotes []StockQuote) string {
	return buildSVG(quotes)
}

func buildSVG(quotes []StockQuote) string {
	cfg := defaultChartConfig()
	plotW := cfg.Width - cfg.PadLeft - cfg.PadRight
	plotH := cfg.Height - cfg.PadTop - cfg.PadBottom

	// Find price range
	minPrice := quotes[0].Low
	maxPrice := quotes[0].High
	for _, q := range quotes {
		if q.Low < minPrice {
			minPrice = q.Low
		}
		if q.High > maxPrice {
			maxPrice = q.High
		}
	}

	// Add 5% padding to price range
	priceRange := maxPrice - minPrice
	if priceRange < 0.01 {
		priceRange = 1
	}
	minPrice -= priceRange * 0.05
	maxPrice += priceRange * 0.05
	priceRange = maxPrice - minPrice

	// Scale helpers
	priceToY := func(p float64) float64 {
		return cfg.PadTop + plotH*(1-(p-minPrice)/priceRange)
	}
	n := len(quotes)
	candleW := (plotW - float64(n-1)*cfg.CandleGap) / float64(n)
	if candleW < 3 {
		candleW = 3
	}
	if candleW > 20 {
		candleW = 20
	}
	indexToX := func(i int) float64 {
		return cfg.PadLeft + float64(i)*(candleW+cfg.CandleGap)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf(
		`<svg viewBox="0 0 %.0f %.0f" preserveAspectRatio="xMidYMid meet" style="width:100%%;height:100%%">`,
		cfg.Width, cfg.Height,
	))

	// Background
	b.WriteString(fmt.Sprintf(
		`<rect x="0" y="0" width="%.0f" height="%.0f" rx="8" fill="rgb(9,9,11)"/>`,
		cfg.Width, cfg.Height,
	))

	// Horizontal gridlines + price labels
	gridLines := 5
	for i := 0; i <= gridLines; i++ {
		price := minPrice + priceRange*float64(i)/float64(gridLines)
		y := priceToY(price)
		b.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="rgb(39,39,42)" stroke-width="0.5"/>`,
			cfg.PadLeft, y, cfg.Width-cfg.PadRight, y,
		))
		b.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" fill="rgb(113,113,122)" font-size="10" font-family="ui-monospace,monospace">%.2f</text>`,
			cfg.Width-cfg.PadRight+8, y+4, price,
		))
	}

	// Candlesticks
	for i, q := range quotes {
		x := indexToX(i)
		cx := x + candleW/2

		isUp := q.Close >= q.Open
		color := "rgb(239,68,68)" // red
		if isUp {
			color = "rgb(34,197,94)" // green
		}

		// Wick
		wickY1 := priceToY(q.High)
		wickY2 := priceToY(q.Low)
		b.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1"/>`,
			cx, wickY1, cx, wickY2, color,
		))

		// Body
		bodyTop := priceToY(math.Max(q.Open, q.Close))
		bodyBot := priceToY(math.Min(q.Open, q.Close))
		bodyH := bodyBot - bodyTop
		if bodyH < 1 {
			bodyH = 1
		}
		b.WriteString(fmt.Sprintf(
			`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`,
			x, bodyTop, candleW, bodyH, color,
		))
	}

	// Moving average (7-period SMA)
	if n > 7 {
		b.WriteString(`<path d="`)
		for i := 6; i < n; i++ {
			sum := 0.0
			for j := i - 6; j <= i; j++ {
				sum += quotes[j].Close
			}
			avg := sum / 7
			x := indexToX(i) + candleW/2
			y := priceToY(avg)
			if i == 6 {
				b.WriteString(fmt.Sprintf("M%.1f,%.1f", x, y))
			} else {
				b.WriteString(fmt.Sprintf(" L%.1f,%.1f", x, y))
			}
		}
		b.WriteString(`" fill="none" stroke="rgb(59,130,246)" stroke-width="1.5" stroke-opacity="0.6"/>`)
	}

	// Current price line (dashed)
	lastPrice := quotes[n-1].Close
	lastY := priceToY(lastPrice)
	isUp := quotes[n-1].Close >= quotes[n-1].Open
	priceLineColor := "rgb(239,68,68)"
	if isUp {
		priceLineColor = "rgb(34,197,94)"
	}

	b.WriteString(fmt.Sprintf(
		`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1" stroke-dasharray="4,3" stroke-opacity="0.7"/>`,
		cfg.PadLeft, lastY, cfg.Width-cfg.PadRight, lastY, priceLineColor,
	))

	// Price badge
	badgeW := 58.0
	badgeH := 18.0
	b.WriteString(fmt.Sprintf(
		`<rect x="%.1f" y="%.1f" width="%.0f" height="%.0f" rx="3" fill="%s"/>`,
		cfg.Width-cfg.PadRight+4, lastY-badgeH/2, badgeW, badgeH, priceLineColor,
	))
	b.WriteString(fmt.Sprintf(
		`<text x="%.1f" y="%.1f" fill="white" font-size="10" font-weight="bold" font-family="ui-monospace,monospace">%.2f</text>`,
		cfg.Width-cfg.PadRight+8, lastY+4, lastPrice,
	))

	// Time labels
	labelInterval := n / 5
	if labelInterval < 1 {
		labelInterval = 1
	}
	for i := 0; i < n; i += labelInterval {
		x := indexToX(i) + candleW/2
		y := cfg.Height - 5.0
		label := quotes[i].Time.Format("15:04")
		b.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" fill="rgb(113,113,122)" font-size="9" font-family="ui-monospace,monospace" text-anchor="middle">%s</text>`,
			x, y, label,
		))
	}

	b.WriteString(`</svg>`)
	return b.String()
}
