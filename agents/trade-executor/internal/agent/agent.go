package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync/atomic"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/trace"
)

type incomingMsg struct {
	TaskID         string   `json:"task_id"`
	Recommendation string   `json:"recommendation"`
	Approved       bool     `json:"approved"`
	RiskLevel      string   `json:"risk_level"`
	Symbols        []string `json:"symbols"`
	TraceID        string   `json:"trace_id"`
}

type outgoingMsg struct {
	TaskID         string   `json:"task_id"`
	OrderID        *string  `json:"order_id"`
	Status         string   `json:"status"`
	ExecutionPrice *float64 `json:"execution_price"`
	Message        string   `json:"message"`
}

// TradeExecutorAgent executes (or rejects) trades based on risk assessment results.
type TradeExecutorAgent struct {
	processedCount int64 // first field for 32-bit atomic alignment
	nc             *nats.Conn
	tracer         trace.Tracer
}

// NewTradeExecutorAgent creates a new TradeExecutorAgent.
func NewTradeExecutorAgent(nc *nats.Conn, tracer trace.Tracer) *TradeExecutorAgent {
	return &TradeExecutorAgent{nc: nc, tracer: tracer}
}

// ProcessedCount returns the number of messages processed so far.
func (a *TradeExecutorAgent) ProcessedCount() int64 {
	return atomic.LoadInt64(&a.processedCount)
}

// Run subscribes to trading.trade.execute and blocks until ctx is cancelled.
func (a *TradeExecutorAgent) Run(ctx context.Context) error {
	sub, err := a.nc.QueueSubscribe("trading.trade.execute", "trade-workers", func(msg *nats.Msg) {
		a.handleMessage(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	log.Println("TradeExecutorAgent running — listening on trading.trade.execute (queue: trade-workers)")
	<-ctx.Done()
	log.Println("TradeExecutorAgent shutting down")
	return nil
}

func (a *TradeExecutorAgent) handleMessage(ctx context.Context, msg *nats.Msg) {
	_, span := a.tracer.Start(ctx, "trade-executor.handle")
	defer span.End()

	var in incomingMsg
	if err := json.Unmarshal(msg.Data, &in); err != nil {
		log.Printf("[ERROR] failed to unmarshal trade.execute request: %v", err)
		return
	}

	log.Printf("[INFO] Received trade.execute task_id=%s recommendation=%s approved=%v risk_level=%s",
		in.TaskID, in.Recommendation, in.Approved, in.RiskLevel)

	out := buildResult(in.TaskID, in.Approved, in.Symbols)

	data, err := json.Marshal(out)
	if err != nil {
		log.Printf("[ERROR] failed to marshal trade.done response: %v", err)
		return
	}

	if err := a.nc.Publish("trading.trade.done", data); err != nil {
		log.Printf("[ERROR] failed to publish trade.done: %v", err)
		return
	}

	atomic.AddInt64(&a.processedCount, 1)
	log.Printf("[INFO] Published trade.done task_id=%s status=%s", in.TaskID, out.Status)
}

// buildResult constructs the trade execution result without side effects.
func buildResult(taskID string, approved bool, symbols []string) outgoingMsg {
	var out outgoingMsg
	out.TaskID = taskID
	if !approved {
		out.Status = "REJECTED"
		out.Message = "Trade rejected due to risk assessment"
	} else {
		orderID := newUUID()
		basePrice := 100.0 + rand.Float64()*900.0
		deviation := (rand.Float64() - 0.5) * 5.0
		execPrice := roundFloat(basePrice+deviation, 2)
		out.Status = "EXECUTED"
		out.OrderID = &orderID
		out.ExecutionPrice = &execPrice
		out.Message = fmt.Sprintf("Order %s executed for %v at %.2f", orderID, symbols, execPrice)
	}
	return out
}

// newUUID generates a simple UUID-like string using random bytes.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func roundFloat(val float64, precision int) float64 {
	factor := 1.0
	for i := 0; i < precision; i++ {
		factor *= 10
	}
	return float64(int(val*factor+0.5)) / factor
}
