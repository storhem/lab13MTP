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

	"github.com/storhem/lab13MTP/agents/risk-manager/internal/state"
)

type incomingMsg struct {
	TaskID         string  `json:"task_id"`
	Recommendation string  `json:"recommendation"`
	AvgPrice       float64 `json:"avg_price"`
	TotalVolume    int64   `json:"total_volume"`
	TraceID        string  `json:"trace_id"`
}

type outgoingMsg struct {
	TaskID    string  `json:"task_id"`
	RiskLevel string  `json:"risk_level"`
	RiskScore float64 `json:"risk_score"`
	Approved  bool    `json:"approved"`
	Status    string  `json:"status"`
}

// stateStore abstracts Redis persistence to allow unit testing without Redis.
type stateStore interface {
	IncrementTotal(ctx context.Context) error
	IncrementHighRisk(ctx context.Context) error
	SaveLastRisk(ctx context.Context, taskID, level string) error
}

// RiskManagerAgent assesses trade risk and persists state to Redis.
type RiskManagerAgent struct {
	processedCount int64 // first field for 32-bit atomic alignment
	nc             *nats.Conn
	tracer         trace.Tracer
	state          stateStore
}

// NewRiskManagerAgent creates a new RiskManagerAgent.
func NewRiskManagerAgent(nc *nats.Conn, tracer trace.Tracer, riskState *state.RiskState) *RiskManagerAgent {
	return &RiskManagerAgent{
		nc:     nc,
		tracer: tracer,
		state:  riskState,
	}
}

// ProcessedCount returns the number of messages processed so far.
func (a *RiskManagerAgent) ProcessedCount() int64 {
	return atomic.LoadInt64(&a.processedCount)
}

// Run subscribes to trading.risk.assess and blocks until ctx is cancelled.
func (a *RiskManagerAgent) Run(ctx context.Context) error {
	sub, err := a.nc.QueueSubscribe("trading.risk.assess", "risk-workers", func(msg *nats.Msg) {
		a.handleMessage(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	log.Println("RiskManagerAgent running — listening on trading.risk.assess (queue: risk-workers)")
	<-ctx.Done()
	log.Println("RiskManagerAgent shutting down")
	return nil
}

func (a *RiskManagerAgent) handleMessage(ctx context.Context, msg *nats.Msg) {
	_, span := a.tracer.Start(ctx, "risk-manager.handle")
	defer span.End()

	var in incomingMsg
	if err := json.Unmarshal(msg.Data, &in); err != nil {
		log.Printf("[ERROR] failed to unmarshal risk.assess request: %v", err)
		return
	}

	log.Printf("[INFO] Received risk.assess task_id=%s recommendation=%s avg_price=%.2f total_volume=%d",
		in.TaskID, in.Recommendation, in.AvgPrice, in.TotalVolume)

	riskLevel, approved := assessRisk(in.Recommendation, in.AvgPrice, in.TotalVolume)

	// Simulated ML risk score (0..1)
	riskScore := rand.Float64()

	// Persist to Redis
	if err := a.state.IncrementTotal(ctx); err != nil {
		log.Printf("[WARN] Redis IncrementTotal error: %v", err)
	}
	if riskLevel == "HIGH" {
		if err := a.state.IncrementHighRisk(ctx); err != nil {
			log.Printf("[WARN] Redis IncrementHighRisk error: %v", err)
		}
	}
	if err := a.state.SaveLastRisk(ctx, in.TaskID, riskLevel); err != nil {
		log.Printf("[WARN] Redis SaveLastRisk error: %v", err)
	}

	out := outgoingMsg{
		TaskID:    in.TaskID,
		RiskLevel: riskLevel,
		RiskScore: roundFloat(riskScore, 4),
		Approved:  approved,
		Status:    "ok",
	}

	data, err := json.Marshal(out)
	if err != nil {
		log.Printf("[ERROR] failed to marshal risk.done response: %v", err)
		return
	}

	if err := a.nc.Publish("trading.risk.done", data); err != nil {
		log.Printf("[ERROR] failed to publish risk.done: %v", err)
		return
	}

	atomic.AddInt64(&a.processedCount, 1)
	log.Printf("[INFO] Published risk.done task_id=%s risk_level=%s approved=%v", in.TaskID, riskLevel, approved)
}

// assessRisk determines risk level and trade approval from market data.
func assessRisk(recommendation string, avgPrice float64, totalVolume int64) (riskLevel string, approved bool) {
	riskLevel = "LOW"
	approved = true
	if recommendation == "BUY" && avgPrice > 800 {
		riskLevel = "HIGH"
		approved = false
	} else if recommendation == "BUY" && totalVolume < 5000 {
		riskLevel = "MEDIUM"
		approved = true
	}
	return
}

func roundFloat(val float64, precision int) float64 {
	factor := 1.0
	for i := 0; i < precision; i++ {
		factor *= 10
	}
	return float64(int(val*factor+0.5)) / factor
}
