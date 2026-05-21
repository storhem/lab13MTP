package agent

import (
	"context"
	"testing"
)

// mockStateStore implements stateStore without Redis for unit testing.
type mockStateStore struct {
	totalCalls    int
	highRiskCalls int
	savedRisk     string
}

func (m *mockStateStore) IncrementTotal(_ context.Context) error {
	m.totalCalls++
	return nil
}

func (m *mockStateStore) IncrementHighRisk(_ context.Context) error {
	m.highRiskCalls++
	return nil
}

func (m *mockStateStore) SaveLastRisk(_ context.Context, _, level string) error {
	m.savedRisk = level
	return nil
}

// --- assessRisk ---

func TestAssessRisk_BuyHighPrice_HIGH_NotApproved(t *testing.T) {
	level, approved := assessRisk("BUY", 801.0, 100000)
	if level != "HIGH" {
		t.Errorf("expected HIGH, got %s", level)
	}
	if approved {
		t.Error("expected approved=false for HIGH risk")
	}
}

func TestAssessRisk_BuyVeryHighPrice_HIGH(t *testing.T) {
	level, approved := assessRisk("BUY", 1000.0, 100000)
	if level != "HIGH" {
		t.Errorf("expected HIGH for price=1000, got %s", level)
	}
	if approved {
		t.Error("expected approved=false")
	}
}

func TestAssessRisk_BuyExactBoundary800_NotHigh(t *testing.T) {
	// avgPrice == 800 is NOT > 800, so should not be HIGH
	level, approved := assessRisk("BUY", 800.0, 100000)
	if level == "HIGH" {
		t.Error("price exactly 800 should not be HIGH (condition is strictly > 800)")
	}
	if !approved {
		t.Error("expected approved=true for price=800")
	}
}

func TestAssessRisk_BuyLowVolume_MEDIUM_Approved(t *testing.T) {
	level, approved := assessRisk("BUY", 500.0, 4999)
	if level != "MEDIUM" {
		t.Errorf("expected MEDIUM, got %s", level)
	}
	if !approved {
		t.Error("expected approved=true for MEDIUM risk")
	}
}

func TestAssessRisk_BuyExactBoundaryVolume5000_NotMedium(t *testing.T) {
	// volume == 5000 is NOT < 5000, should be LOW
	level, _ := assessRisk("BUY", 500.0, 5000)
	if level == "MEDIUM" {
		t.Error("volume exactly 5000 should not be MEDIUM (condition is strictly < 5000)")
	}
}

func TestAssessRisk_BuyNormalConditions_LOW(t *testing.T) {
	level, approved := assessRisk("BUY", 500.0, 60000)
	if level != "LOW" {
		t.Errorf("expected LOW for normal BUY, got %s", level)
	}
	if !approved {
		t.Error("expected approved=true for LOW risk")
	}
}

func TestAssessRisk_Sell_LOW_Approved(t *testing.T) {
	// SELL recommendation: risk rules only apply to BUY
	level, approved := assessRisk("SELL", 900.0, 100000)
	if level != "LOW" {
		t.Errorf("expected LOW for SELL, got %s", level)
	}
	if !approved {
		t.Error("expected approved=true for SELL")
	}
}

func TestAssessRisk_Hold_LOW_Approved(t *testing.T) {
	level, approved := assessRisk("HOLD", 500.0, 50000)
	if level != "LOW" {
		t.Errorf("expected LOW for HOLD, got %s", level)
	}
	if !approved {
		t.Error("expected approved=true for HOLD")
	}
}

func TestAssessRisk_SellHighPrice_StillLow(t *testing.T) {
	// High price + SELL → still LOW because rules are BUY-specific
	level, _ := assessRisk("SELL", 999.0, 100)
	if level != "LOW" {
		t.Errorf("SELL should always be LOW, got %s", level)
	}
}

func TestAssessRisk_BuyZeroVolume_MEDIUM(t *testing.T) {
	level, approved := assessRisk("BUY", 300.0, 0)
	if level != "MEDIUM" {
		t.Errorf("expected MEDIUM for zero volume, got %s", level)
	}
	if !approved {
		t.Error("expected approved=true for MEDIUM")
	}
}

// --- roundFloat ---

func TestRoundFloat(t *testing.T) {
	cases := []struct {
		val  float64
		prec int
		want float64
	}{
		{0.12345, 2, 0.12},
		{0.125, 2, 0.13},
		{1000.0, 0, 1000.0},
		{0.0, 4, 0.0},
	}
	for _, tc := range cases {
		got := roundFloat(tc.val, tc.prec)
		if got != tc.want {
			t.Errorf("roundFloat(%v, %d) = %v, want %v", tc.val, tc.prec, got, tc.want)
		}
	}
}

// --- mockStateStore ---

func TestMockStateStore_IncrementTotalCounts(t *testing.T) {
	m := &mockStateStore{}
	ctx := context.Background()
	_ = m.IncrementTotal(ctx)
	_ = m.IncrementTotal(ctx)
	_ = m.IncrementTotal(ctx)
	if m.totalCalls != 3 {
		t.Errorf("expected 3 total calls, got %d", m.totalCalls)
	}
}

func TestMockStateStore_IncrementHighRiskCounts(t *testing.T) {
	m := &mockStateStore{}
	ctx := context.Background()
	_ = m.IncrementTotal(ctx)
	_ = m.IncrementHighRisk(ctx)
	if m.highRiskCalls != 1 {
		t.Errorf("expected 1 high risk call, got %d", m.highRiskCalls)
	}
	if m.totalCalls != 1 {
		t.Errorf("expected 1 total call, got %d", m.totalCalls)
	}
}

func TestMockStateStore_SaveLastRisk(t *testing.T) {
	m := &mockStateStore{}
	_ = m.SaveLastRisk(context.Background(), "task-1", "HIGH")
	if m.savedRisk != "HIGH" {
		t.Errorf("expected HIGH, got %s", m.savedRisk)
	}
	// overwrite
	_ = m.SaveLastRisk(context.Background(), "task-2", "LOW")
	if m.savedRisk != "LOW" {
		t.Errorf("expected LOW after overwrite, got %s", m.savedRisk)
	}
}

func TestMockStateStore_NoCallsInitially(t *testing.T) {
	m := &mockStateStore{}
	if m.totalCalls != 0 || m.highRiskCalls != 0 || m.savedRisk != "" {
		t.Error("mock should start with zero state")
	}
}

// --- ProcessedCount ---

func TestProcessedCount_InitialZero(t *testing.T) {
	a := &RiskManagerAgent{}
	if a.ProcessedCount() != 0 {
		t.Errorf("expected initial count 0, got %d", a.ProcessedCount())
	}
}
