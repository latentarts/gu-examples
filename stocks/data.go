//go:build js && wasm

package main

import (
	"math"
	"time"
)

// StockQuote represents a single OHLC candlestick.
type StockQuote struct {
	Open  float64
	High  float64
	Low   float64
	Close float64
	Time  time.Time
}

// StockInfo holds metadata for a stock.
type StockInfo struct {
	Symbol string
	Name   string
	Sector string
}

// PopularStocks is the autocomplete list (~50 entries).
var PopularStocks = []StockInfo{
	{"AAPL", "Apple Inc.", "Technology"},
	{"MSFT", "Microsoft Corp.", "Technology"},
	{"GOOGL", "Alphabet Inc.", "Technology"},
	{"AMZN", "Amazon.com Inc.", "Consumer Cyclical"},
	{"NVDA", "NVIDIA Corp.", "Technology"},
	{"META", "Meta Platforms Inc.", "Technology"},
	{"TSLA", "Tesla Inc.", "Consumer Cyclical"},
	{"BRK.B", "Berkshire Hathaway", "Financial"},
	{"JPM", "JPMorgan Chase", "Financial"},
	{"V", "Visa Inc.", "Financial"},
	{"JNJ", "Johnson & Johnson", "Healthcare"},
	{"UNH", "UnitedHealth Group", "Healthcare"},
	{"WMT", "Walmart Inc.", "Consumer Defensive"},
	{"PG", "Procter & Gamble", "Consumer Defensive"},
	{"MA", "Mastercard Inc.", "Financial"},
	{"HD", "Home Depot Inc.", "Consumer Cyclical"},
	{"DIS", "Walt Disney Co.", "Communication"},
	{"BAC", "Bank of America", "Financial"},
	{"XOM", "Exxon Mobil Corp.", "Energy"},
	{"PFE", "Pfizer Inc.", "Healthcare"},
	{"KO", "Coca-Cola Co.", "Consumer Defensive"},
	{"PEP", "PepsiCo Inc.", "Consumer Defensive"},
	{"CSCO", "Cisco Systems", "Technology"},
	{"AVGO", "Broadcom Inc.", "Technology"},
	{"ADBE", "Adobe Inc.", "Technology"},
	{"CRM", "Salesforce Inc.", "Technology"},
	{"NFLX", "Netflix Inc.", "Communication"},
	{"AMD", "Advanced Micro Devices", "Technology"},
	{"INTC", "Intel Corp.", "Technology"},
	{"CMCSA", "Comcast Corp.", "Communication"},
	{"T", "AT&T Inc.", "Communication"},
	{"VZ", "Verizon Communications", "Communication"},
	{"NKE", "Nike Inc.", "Consumer Cyclical"},
	{"MRK", "Merck & Co.", "Healthcare"},
	{"ABT", "Abbott Laboratories", "Healthcare"},
	{"TMO", "Thermo Fisher Scientific", "Healthcare"},
	{"ORCL", "Oracle Corp.", "Technology"},
	{"ACN", "Accenture plc", "Technology"},
	{"LLY", "Eli Lilly & Co.", "Healthcare"},
	{"COST", "Costco Wholesale", "Consumer Defensive"},
	{"DHR", "Danaher Corp.", "Healthcare"},
	{"TXN", "Texas Instruments", "Technology"},
	{"NEE", "NextEra Energy", "Utilities"},
	{"PM", "Philip Morris Intl.", "Consumer Defensive"},
	{"UPS", "United Parcel Service", "Industrials"},
	{"RTX", "RTX Corp.", "Industrials"},
	{"QCOM", "Qualcomm Inc.", "Technology"},
	{"LOW", "Lowe's Companies", "Consumer Cyclical"},
	{"SPGI", "S&P Global Inc.", "Financial"},
	{"BA", "Boeing Co.", "Industrials"},
}

// basePrices gives a deterministic starting price for each symbol.
var basePrices = map[string]float64{
	"AAPL": 185.0, "MSFT": 415.0, "GOOGL": 155.0, "AMZN": 185.0,
	"NVDA": 875.0, "META": 510.0, "TSLA": 245.0, "BRK.B": 410.0,
	"JPM": 195.0, "V": 280.0, "JNJ": 155.0, "UNH": 530.0,
	"WMT": 175.0, "PG": 165.0, "MA": 465.0, "HD": 370.0,
	"DIS": 112.0, "BAC": 35.0, "XOM": 105.0, "PFE": 27.0,
	"KO": 60.0, "PEP": 170.0, "CSCO": 50.0, "AVGO": 1350.0,
	"ADBE": 580.0, "CRM": 270.0, "NFLX": 610.0, "AMD": 175.0,
	"INTC": 44.0, "CMCSA": 42.0, "T": 17.0, "VZ": 40.0,
	"NKE": 105.0, "MRK": 125.0, "ABT": 110.0, "TMO": 570.0,
	"ORCL": 125.0, "ACN": 370.0, "LLY": 780.0, "COST": 720.0,
	"DHR": 250.0, "TXN": 170.0, "NEE": 65.0, "PM": 95.0,
	"UPS": 150.0, "RTX": 95.0, "QCOM": 165.0, "LOW": 235.0,
	"SPGI": 460.0, "BA": 215.0,
}

// simpleHash produces a deterministic seed from a symbol string.
func simpleHash(s string) uint64 {
	var h uint64 = 5381
	for i := 0; i < len(s); i++ {
		h = h*33 + uint64(s[i])
	}
	return h
}

// pseudoRand is a simple LCG PRNG (no math/rand needed, deterministic).
type pseudoRand struct {
	state uint64
}

func newRand(seed uint64) *pseudoRand {
	return &pseudoRand{state: seed}
}

func (r *pseudoRand) next() uint64 {
	r.state = r.state*6364136223846793005 + 1442695040888963407
	return r.state
}

// float64 returns a value in [0, 1).
func (r *pseudoRand) float64() float64 {
	return float64(r.next()>>11) / (1 << 53)
}

// norm returns a roughly normal-distributed value (Box-Muller).
func (r *pseudoRand) norm() float64 {
	u1 := r.float64()
	u2 := r.float64()
	if u1 < 1e-10 {
		u1 = 1e-10
	}
	return math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}

// GenerateCandles produces count historical candlesticks for symbol.
func GenerateCandles(symbol string, count int) []StockQuote {
	base, ok := basePrices[symbol]
	if !ok {
		base = 100.0
	}
	rng := newRand(simpleHash(symbol))

	candles := make([]StockQuote, count)
	price := base
	t := time.Date(2025, 1, 2, 9, 30, 0, 0, time.UTC)

	for i := 0; i < count; i++ {
		volatility := base * 0.015
		change := rng.norm() * volatility
		open := price
		close := open + change
		high := math.Max(open, close) + math.Abs(rng.norm()*volatility*0.5)
		low := math.Min(open, close) - math.Abs(rng.norm()*volatility*0.5)

		candles[i] = StockQuote{
			Open:  round2(open),
			High:  round2(high),
			Low:   round2(low),
			Close: round2(close),
			Time:  t,
		}
		price = close
		t = t.Add(5 * time.Minute)
	}
	return candles
}

// NextCandle generates the next candlestick given the previous close.
func NextCandle(prev StockQuote, rng *pseudoRand) StockQuote {
	base := prev.Close
	volatility := base * 0.012
	change := rng.norm() * volatility
	open := base + rng.norm()*volatility*0.2
	close := open + change
	high := math.Max(open, close) + math.Abs(rng.norm()*volatility*0.5)
	low := math.Min(open, close) - math.Abs(rng.norm()*volatility*0.5)

	return StockQuote{
		Open:  round2(open),
		High:  round2(high),
		Low:   round2(low),
		Close: round2(close),
		Time:  prev.Time.Add(5 * time.Minute),
	}
}

// TickPrice simulates a small intra-candle price movement on the last candle.
// It mutates the candle's Close, and updates High/Low if exceeded.
func TickPrice(candle *StockQuote, rng *pseudoRand) {
	volatility := candle.Open * 0.002 // small tick volatility
	delta := rng.norm() * volatility
	candle.Close = round2(candle.Close + delta)
	if candle.Close > candle.High {
		candle.High = candle.Close
	}
	if candle.Close < candle.Low {
		candle.Low = candle.Close
	}
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}
