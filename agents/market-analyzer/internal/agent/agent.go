package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/trace"
)

// QuoteEntry mirrors the structure sent by quotes-agent.
type QuoteEntry struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Volume    int64   `json:"volume"`
	Timestamp string  `json:"timestamp"`
}

type incomingMsg struct {
	TaskID  string       `json:"task_id"`
	Quotes  []QuoteEntry `json:"quotes"`
	TraceID string       `json:"trace_id"`
}

type outgoingMsg struct {
	TaskID          string  `json:"task_id"`
	AvgPrice        float64 `json:"avg_price"`
	Trend           string  `json:"trend"`
	TotalVolume     int64   `json:"total_volume"`
	Recommendation  string  `json:"recommendation"`
	Status          string  `json:"status"`
}

// MarketAnalyzerAgent analyses market quotes and produces recommendations.
type MarketAnalyzerAgent struct {
	processedCount int64 // first field for 32-bit atomic alignment
	nc             *nats.Conn
	tracer         trace.Tracer
}

// NewMarketAnalyzerAgent creates a new MarketAnalyzerAgent.
func NewMarketAnalyzerAgent(nc *nats.Conn, tracer trace.Tracer) *MarketAnalyzerAgent {
	return &MarketAnalyzerAgent{nc: nc, tracer: tracer}
}

// ProcessedCount returns the number of messages processed so far.
func (a *MarketAnalyzerAgent) ProcessedCount() int64 {
	return atomic.LoadInt64(&a.processedCount)
}

// Run subscribes to trading.market.analyze and blocks until ctx is cancelled.
func (a *MarketAnalyzerAgent) Run(ctx context.Context) error {
	sub, err := a.nc.QueueSubscribe("trading.market.analyze", "market-workers", func(msg *nats.Msg) {
		a.handleMessage(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	log.Println("MarketAnalyzerAgent running — listening on trading.market.analyze (queue: market-workers)")
	<-ctx.Done()
	log.Println("MarketAnalyzerAgent shutting down")
	return nil
}

func (a *MarketAnalyzerAgent) handleMessage(ctx context.Context, msg *nats.Msg) {
	_, span := a.tracer.Start(ctx, "market-analyzer.handle")
	defer span.End()

	var in incomingMsg
	if err := json.Unmarshal(msg.Data, &in); err != nil {
		log.Printf("[ERROR] failed to unmarshal market.analyze request: %v", err)
		return
	}

	log.Printf("[INFO] Received market.analyze task_id=%s quotes=%d", in.TaskID, len(in.Quotes))

	if len(in.Quotes) == 0 {
		log.Printf("[WARN] No quotes to analyse for task_id=%s", in.TaskID)
		return
	}

	out := analyzeQuotes(in.TaskID, in.Quotes)

	data, err := json.Marshal(out)
	if err != nil {
		log.Printf("[ERROR] failed to marshal market.done response: %v", err)
		return
	}

	if err := a.nc.Publish("trading.market.done", data); err != nil {
		log.Printf("[ERROR] failed to publish market.done: %v", err)
		return
	}

	atomic.AddInt64(&a.processedCount, 1)
	log.Printf("[INFO] Published market.done task_id=%s trend=%s recommendation=%s", in.TaskID, out.Trend, out.Recommendation)
}

// analyzeQuotes computes market metrics and a trading recommendation from quotes.
func analyzeQuotes(taskID string, quotes []QuoteEntry) outgoingMsg {
	var totalPrice float64
	var totalVolume int64
	for _, q := range quotes {
		totalPrice += q.Price
		totalVolume += q.Volume
	}
	avgPrice := totalPrice / float64(len(quotes))

	firstPrice := quotes[0].Price
	lastPrice := quotes[len(quotes)-1].Price
	trend := "neutral"
	if lastPrice > firstPrice {
		trend = "bullish"
	} else if lastPrice < firstPrice {
		trend = "bearish"
	}

	recommendation := "HOLD"
	if trend == "bullish" && totalVolume > 50000 {
		recommendation = "BUY"
	} else if trend == "bearish" {
		recommendation = "SELL"
	}

	return outgoingMsg{
		TaskID:         taskID,
		AvgPrice:       roundFloat(avgPrice, 2),
		Trend:          trend,
		TotalVolume:    totalVolume,
		Recommendation: recommendation,
		Status:         "ok",
	}
}

func roundFloat(val float64, precision int) float64 {
	factor := 1.0
	for i := 0; i < precision; i++ {
		factor *= 10
	}
	return float64(int(val*factor+0.5)) / factor
}
