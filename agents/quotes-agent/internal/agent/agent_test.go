package agent

import (
	"testing"
	"time"
)

// --- roundFloat ---

func TestRoundFloat_TwoDecimals(t *testing.T) {
	cases := []struct {
		val  float64
		prec int
		want float64
	}{
		{3.14159, 2, 3.14},
		{0.125, 2, 0.13},
		{0.0, 2, 0.0},
		{999.999, 2, 1000.0},
		{100.005, 2, 100.01},
	}
	for _, tc := range cases {
		got := roundFloat(tc.val, tc.prec)
		if got != tc.want {
			t.Errorf("roundFloat(%v, %d) = %v, want %v", tc.val, tc.prec, got, tc.want)
		}
	}
}

func TestRoundFloat_ZeroPrecision(t *testing.T) {
	if got := roundFloat(1.5, 0); got != 2.0 {
		t.Errorf("roundFloat(1.5, 0) = %v, want 2.0", got)
	}
	if got := roundFloat(1.4, 0); got != 1.0 {
		t.Errorf("roundFloat(1.4, 0) = %v, want 1.0", got)
	}
}

// --- generateQuote ---

func TestGenerateQuote_SymbolPreserved(t *testing.T) {
	q := generateQuote("AAPL")
	if q.Symbol != "AAPL" {
		t.Errorf("expected symbol AAPL, got %s", q.Symbol)
	}
}

func TestGenerateQuote_PriceInRange(t *testing.T) {
	for i := 0; i < 200; i++ {
		q := generateQuote("TEST")
		if q.Price < 100.0 || q.Price > 1000.0 {
			t.Errorf("iteration %d: price %v out of range [100, 1000]", i, q.Price)
		}
	}
}

func TestGenerateQuote_VolumeInRange(t *testing.T) {
	for i := 0; i < 200; i++ {
		q := generateQuote("TEST")
		if q.Volume < 1000 || q.Volume > 100000 {
			t.Errorf("iteration %d: volume %d out of range [1000, 100000]", i, q.Volume)
		}
	}
}

func TestGenerateQuote_TimestampRFC3339(t *testing.T) {
	q := generateQuote("GOOGL")
	if q.Timestamp == "" {
		t.Fatal("timestamp should not be empty")
	}
	if _, err := time.Parse(time.RFC3339, q.Timestamp); err != nil {
		t.Errorf("timestamp %q is not RFC3339: %v", q.Timestamp, err)
	}
}

func TestGenerateQuote_PriceRounded(t *testing.T) {
	for i := 0; i < 100; i++ {
		q := generateQuote("X")
		// Price should have at most 2 decimal places after rounding
		rounded := roundFloat(q.Price, 2)
		if q.Price != rounded {
			t.Errorf("price %v is not rounded to 2 decimals", q.Price)
		}
	}
}

// --- generateQuotes ---

func TestGenerateQuotes_CorrectCount(t *testing.T) {
	symbols := []string{"AAPL", "GOOGL", "TSLA"}
	quotes := generateQuotes(symbols)
	if len(quotes) != 3 {
		t.Errorf("expected 3 quotes, got %d", len(quotes))
	}
}

func TestGenerateQuotes_SymbolsMatch(t *testing.T) {
	symbols := []string{"AAPL", "GOOGL", "MSFT"}
	quotes := generateQuotes(symbols)
	for i, q := range quotes {
		if q.Symbol != symbols[i] {
			t.Errorf("quotes[%d].Symbol = %s, want %s", i, q.Symbol, symbols[i])
		}
	}
}

func TestGenerateQuotes_EmptyInput(t *testing.T) {
	quotes := generateQuotes([]string{})
	if len(quotes) != 0 {
		t.Errorf("expected empty slice, got %d quotes", len(quotes))
	}
}

func TestGenerateQuotes_SingleSymbol(t *testing.T) {
	quotes := generateQuotes([]string{"BTC"})
	if len(quotes) != 1 {
		t.Fatalf("expected 1 quote, got %d", len(quotes))
	}
	if quotes[0].Symbol != "BTC" {
		t.Errorf("expected BTC, got %s", quotes[0].Symbol)
	}
}

func TestGenerateQuotes_AllPricesInRange(t *testing.T) {
	symbols := []string{"A", "B", "C", "D", "E"}
	for i := 0; i < 50; i++ {
		quotes := generateQuotes(symbols)
		for _, q := range quotes {
			if q.Price < 100.0 || q.Price > 1000.0 {
				t.Errorf("price %v out of range for symbol %s", q.Price, q.Symbol)
			}
		}
	}
}

func TestGenerateQuotes_AllVolumesInRange(t *testing.T) {
	symbols := []string{"A", "B", "C"}
	for i := 0; i < 50; i++ {
		quotes := generateQuotes(symbols)
		for _, q := range quotes {
			if q.Volume < 1000 || q.Volume > 100000 {
				t.Errorf("volume %d out of range for symbol %s", q.Volume, q.Symbol)
			}
		}
	}
}

// --- ProcessedCount ---

func TestProcessedCount_InitialZero(t *testing.T) {
	a := &QuotesAgent{}
	if a.ProcessedCount() != 0 {
		t.Errorf("expected initial count 0, got %d", a.ProcessedCount())
	}
}
