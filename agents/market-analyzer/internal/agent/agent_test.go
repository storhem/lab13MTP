package agent

import (
	"testing"
)

// helper: build a QuoteEntry without a timestamp (not needed for analysis tests)
func makeQuote(symbol string, price float64, volume int64) QuoteEntry {
	return QuoteEntry{Symbol: symbol, Price: price, Volume: volume}
}

// --- roundFloat ---

func TestRoundFloat_BasicCases(t *testing.T) {
	cases := []struct {
		val  float64
		prec int
		want float64
	}{
		{3.14159, 2, 3.14},
		{0.125, 2, 0.13},
		{0.0, 2, 0.0},
		{100.005, 2, 100.01},
		{50.0, 0, 50.0},
	}
	for _, tc := range cases {
		got := roundFloat(tc.val, tc.prec)
		if got != tc.want {
			t.Errorf("roundFloat(%v, %d) = %v, want %v", tc.val, tc.prec, got, tc.want)
		}
	}
}

// --- analyzeQuotes: trend detection ---

func TestAnalyzeQuotes_Bullish_LastHigherThanFirst(t *testing.T) {
	quotes := []QuoteEntry{
		makeQuote("AAPL", 100.0, 30000),
		makeQuote("AAPL", 150.0, 30000), // last > first
	}
	out := analyzeQuotes("task-1", quotes)
	if out.Trend != "bullish" {
		t.Errorf("expected bullish, got %s", out.Trend)
	}
}

func TestAnalyzeQuotes_Bearish_LastLowerThanFirst(t *testing.T) {
	quotes := []QuoteEntry{
		makeQuote("TSLA", 500.0, 40000),
		makeQuote("TSLA", 400.0, 40000), // last < first
	}
	out := analyzeQuotes("task-2", quotes)
	if out.Trend != "bearish" {
		t.Errorf("expected bearish, got %s", out.Trend)
	}
}

func TestAnalyzeQuotes_Neutral_EqualPrices(t *testing.T) {
	quotes := []QuoteEntry{
		makeQuote("MSFT", 300.0, 40000),
		makeQuote("MSFT", 300.0, 40000), // equal
	}
	out := analyzeQuotes("task-3", quotes)
	if out.Trend != "neutral" {
		t.Errorf("expected neutral, got %s", out.Trend)
	}
}

func TestAnalyzeQuotes_SingleQuote_Neutral(t *testing.T) {
	quotes := []QuoteEntry{makeQuote("A", 150.0, 80000)}
	out := analyzeQuotes("task-4", quotes)
	// first == last → neutral
	if out.Trend != "neutral" {
		t.Errorf("single quote should be neutral, got %s", out.Trend)
	}
}

// --- analyzeQuotes: recommendation ---

func TestAnalyzeQuotes_BullishHighVolume_BUY(t *testing.T) {
	// bullish + volume > 50000 → BUY
	quotes := []QuoteEntry{
		makeQuote("AAPL", 100.0, 30000),
		makeQuote("AAPL", 200.0, 30000), // total = 60000 > 50000
	}
	out := analyzeQuotes("task-5", quotes)
	if out.Recommendation != "BUY" {
		t.Errorf("expected BUY, got %s", out.Recommendation)
	}
}

func TestAnalyzeQuotes_BullishLowVolume_HOLD(t *testing.T) {
	// bullish but total volume ≤ 50000 → HOLD
	quotes := []QuoteEntry{
		makeQuote("AAPL", 100.0, 10000),
		makeQuote("AAPL", 200.0, 10000), // total = 20000
	}
	out := analyzeQuotes("task-6", quotes)
	if out.Recommendation != "HOLD" {
		t.Errorf("expected HOLD, got %s", out.Recommendation)
	}
}

func TestAnalyzeQuotes_BullishExactVolume50000_HOLD(t *testing.T) {
	// volume exactly 50000 → NOT > 50000 → HOLD
	quotes := []QuoteEntry{
		makeQuote("X", 100.0, 25000),
		makeQuote("X", 200.0, 25000), // total = 50000, not > 50000
	}
	out := analyzeQuotes("task-7", quotes)
	if out.Recommendation != "HOLD" {
		t.Errorf("volume=50000 should give HOLD, got %s", out.Recommendation)
	}
}

func TestAnalyzeQuotes_Bearish_SELL(t *testing.T) {
	quotes := []QuoteEntry{
		makeQuote("TSLA", 600.0, 80000),
		makeQuote("TSLA", 400.0, 80000),
	}
	out := analyzeQuotes("task-8", quotes)
	if out.Recommendation != "SELL" {
		t.Errorf("expected SELL, got %s", out.Recommendation)
	}
}

func TestAnalyzeQuotes_Neutral_HOLD(t *testing.T) {
	quotes := []QuoteEntry{
		makeQuote("X", 200.0, 80000),
		makeQuote("X", 200.0, 80000),
	}
	out := analyzeQuotes("task-9", quotes)
	if out.Recommendation != "HOLD" {
		t.Errorf("expected HOLD for neutral, got %s", out.Recommendation)
	}
}

// --- analyzeQuotes: aggregates ---

func TestAnalyzeQuotes_AvgPrice_TwoQuotes(t *testing.T) {
	quotes := []QuoteEntry{
		makeQuote("X", 100.0, 1000),
		makeQuote("X", 300.0, 1000),
	}
	out := analyzeQuotes("task-10", quotes)
	if out.AvgPrice != 200.0 {
		t.Errorf("expected avg_price 200.0, got %v", out.AvgPrice)
	}
}

func TestAnalyzeQuotes_AvgPrice_ThreeQuotes(t *testing.T) {
	quotes := []QuoteEntry{
		makeQuote("X", 100.0, 1000),
		makeQuote("X", 200.0, 1000),
		makeQuote("X", 300.0, 1000),
	}
	out := analyzeQuotes("task-11", quotes)
	if out.AvgPrice != 200.0 {
		t.Errorf("expected avg_price 200.0, got %v", out.AvgPrice)
	}
}

func TestAnalyzeQuotes_TotalVolume(t *testing.T) {
	quotes := []QuoteEntry{
		makeQuote("X", 100.0, 10000),
		makeQuote("X", 200.0, 20000),
		makeQuote("X", 300.0, 30000),
	}
	out := analyzeQuotes("task-12", quotes)
	if out.TotalVolume != 60000 {
		t.Errorf("expected total_volume 60000, got %d", out.TotalVolume)
	}
}

func TestAnalyzeQuotes_TaskIDPreserved(t *testing.T) {
	quotes := []QuoteEntry{makeQuote("A", 100.0, 1000)}
	out := analyzeQuotes("unique-task-id", quotes)
	if out.TaskID != "unique-task-id" {
		t.Errorf("expected unique-task-id, got %s", out.TaskID)
	}
}

func TestAnalyzeQuotes_StatusOK(t *testing.T) {
	quotes := []QuoteEntry{makeQuote("A", 100.0, 1000)}
	out := analyzeQuotes("task", quotes)
	if out.Status != "ok" {
		t.Errorf("expected status ok, got %s", out.Status)
	}
}

func TestAnalyzeQuotes_MultipleSymbols_BUY(t *testing.T) {
	// Mixed symbols: last price > first → bullish, total > 50000 → BUY
	quotes := []QuoteEntry{
		makeQuote("AAPL", 150.0, 20000),
		makeQuote("GOOGL", 200.0, 20000),
		makeQuote("TSLA", 300.0, 20000), // last(300) > first(150) → bullish, total=60000
	}
	out := analyzeQuotes("task-13", quotes)
	if out.Recommendation != "BUY" {
		t.Errorf("expected BUY, got %s", out.Recommendation)
	}
	if out.TotalVolume != 60000 {
		t.Errorf("expected total_volume 60000, got %d", out.TotalVolume)
	}
}

// --- ProcessedCount ---

func TestProcessedCount_InitialZero(t *testing.T) {
	a := &MarketAnalyzerAgent{}
	if a.ProcessedCount() != 0 {
		t.Errorf("expected initial count 0, got %d", a.ProcessedCount())
	}
}
