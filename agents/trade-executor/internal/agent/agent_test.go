package agent

import (
	"strings"
	"testing"
)

// --- roundFloat ---

func TestRoundFloat(t *testing.T) {
	cases := []struct {
		val  float64
		prec int
		want float64
	}{
		{1.456, 2, 1.46},
		{0.0, 0, 0.0},
		{99.999, 2, 100.0},
		{3.14159, 2, 3.14},
		{50.0, 0, 50.0},
	}
	for _, tc := range cases {
		got := roundFloat(tc.val, tc.prec)
		if got != tc.want {
			t.Errorf("roundFloat(%v, %d) = %v, want %v", tc.val, tc.prec, got, tc.want)
		}
	}
}

// --- buildResult: rejected path ---

func TestBuildResult_Rejected_Status(t *testing.T) {
	out := buildResult("task-1", false, []string{"AAPL"})
	if out.Status != "REJECTED" {
		t.Errorf("expected REJECTED, got %s", out.Status)
	}
}

func TestBuildResult_Rejected_NilOrderID(t *testing.T) {
	out := buildResult("task-2", false, []string{"AAPL"})
	if out.OrderID != nil {
		t.Errorf("expected nil OrderID for rejected trade, got %v", *out.OrderID)
	}
}

func TestBuildResult_Rejected_NilExecutionPrice(t *testing.T) {
	out := buildResult("task-3", false, []string{"AAPL"})
	if out.ExecutionPrice != nil {
		t.Errorf("expected nil ExecutionPrice for rejected trade, got %v", *out.ExecutionPrice)
	}
}

func TestBuildResult_Rejected_Message(t *testing.T) {
	out := buildResult("task-4", false, []string{"AAPL"})
	if out.Message != "Trade rejected due to risk assessment" {
		t.Errorf("unexpected rejection message: %s", out.Message)
	}
}

func TestBuildResult_Rejected_TaskIDPreserved(t *testing.T) {
	out := buildResult("my-task-id", false, []string{"AAPL"})
	if out.TaskID != "my-task-id" {
		t.Errorf("expected my-task-id, got %s", out.TaskID)
	}
}

// --- buildResult: executed path ---

func TestBuildResult_Executed_Status(t *testing.T) {
	out := buildResult("task-5", true, []string{"GOOGL"})
	if out.Status != "EXECUTED" {
		t.Errorf("expected EXECUTED, got %s", out.Status)
	}
}

func TestBuildResult_Executed_HasOrderID(t *testing.T) {
	out := buildResult("task-6", true, []string{"GOOGL"})
	if out.OrderID == nil {
		t.Fatal("expected non-nil OrderID for executed trade")
	}
	if *out.OrderID == "" {
		t.Error("OrderID should not be empty")
	}
}

func TestBuildResult_Executed_HasExecutionPrice(t *testing.T) {
	out := buildResult("task-7", true, []string{"TSLA"})
	if out.ExecutionPrice == nil {
		t.Fatal("expected non-nil ExecutionPrice for executed trade")
	}
	if *out.ExecutionPrice <= 0 {
		t.Errorf("ExecutionPrice should be positive, got %v", *out.ExecutionPrice)
	}
}

func TestBuildResult_Executed_PriceInRange(t *testing.T) {
	// base price is 100–1000, deviation ±2.5, so range is ~[97.5, 1002.5]
	for i := 0; i < 200; i++ {
		out := buildResult("task", true, []string{"X"})
		if out.ExecutionPrice == nil {
			t.Fatal("nil ExecutionPrice")
		}
		if *out.ExecutionPrice < 90 || *out.ExecutionPrice > 1010 {
			t.Errorf("iteration %d: price %v out of expected range", i, *out.ExecutionPrice)
		}
	}
}

func TestBuildResult_Executed_TaskIDPreserved(t *testing.T) {
	out := buildResult("unique-task", true, []string{"BTC"})
	if out.TaskID != "unique-task" {
		t.Errorf("expected unique-task, got %s", out.TaskID)
	}
}

func TestBuildResult_Executed_MessageContainsSymbol(t *testing.T) {
	out := buildResult("task-8", true, []string{"AAPL"})
	if !strings.Contains(out.Message, "AAPL") {
		t.Errorf("message should contain AAPL: %s", out.Message)
	}
}

func TestBuildResult_Executed_MessageMultipleSymbols(t *testing.T) {
	out := buildResult("task-9", true, []string{"AAPL", "GOOGL"})
	if !strings.Contains(out.Message, "AAPL") || !strings.Contains(out.Message, "GOOGL") {
		t.Errorf("message should contain all symbols: %s", out.Message)
	}
}

func TestBuildResult_Executed_EmptySymbols(t *testing.T) {
	out := buildResult("task-10", true, []string{})
	if out.Status != "EXECUTED" {
		t.Errorf("expected EXECUTED even with empty symbols, got %s", out.Status)
	}
}

// --- newUUID ---

func TestNewUUID_HasFiveParts(t *testing.T) {
	u := newUUID()
	parts := strings.Split(u, "-")
	if len(parts) != 5 {
		t.Errorf("UUID should have 5 dash-separated parts, got %d: %s", len(parts), u)
	}
}

func TestNewUUID_NotEmpty(t *testing.T) {
	u := newUUID()
	if u == "" {
		t.Error("UUID should not be empty")
	}
}

func TestNewUUID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool, 200)
	for i := 0; i < 200; i++ {
		u := newUUID()
		if seen[u] {
			t.Errorf("duplicate UUID generated: %s", u)
		}
		seen[u] = true
	}
}

// --- ProcessedCount ---

func TestProcessedCount_InitialZero(t *testing.T) {
	a := &TradeExecutorAgent{}
	if a.ProcessedCount() != 0 {
		t.Errorf("expected initial count 0, got %d", a.ProcessedCount())
	}
}
