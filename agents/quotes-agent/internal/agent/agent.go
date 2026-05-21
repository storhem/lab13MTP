package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/trace"
)

// QuoteEntry represents a single market quote.
type QuoteEntry struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Volume    int64   `json:"volume"`
	Timestamp string  `json:"timestamp"`
}

// incomingMsg is the payload received on trading.quotes.collect.
type incomingMsg struct {
	TaskID  string   `json:"task_id"`
	Symbols []string `json:"symbols"`
	TraceID string   `json:"trace_id"`
}

// outgoingMsg is the payload published on trading.quotes.done.
type outgoingMsg struct {
	TaskID string       `json:"task_id"`
	Quotes []QuoteEntry `json:"quotes"`
	Status string       `json:"status"`
}

// QuotesAgent subscribes to trading.quotes.collect and publishes collected quotes.
type QuotesAgent struct {
	processedCount int64 // first field for 32-bit atomic alignment
	nc             *nats.Conn
	tracer         trace.Tracer
}

// NewQuotesAgent creates a new QuotesAgent.
func NewQuotesAgent(nc *nats.Conn, tracer trace.Tracer) *QuotesAgent {
	return &QuotesAgent{
		nc:     nc,
		tracer: tracer,
	}
}

// ProcessedCount returns the number of messages processed so far.
func (a *QuotesAgent) ProcessedCount() int64 {
	return atomic.LoadInt64(&a.processedCount)
}

// Run subscribes to the collect topic and starts processing messages.
// It blocks until the context is cancelled.
func (a *QuotesAgent) Run(ctx context.Context) error {
	sub, err := a.nc.QueueSubscribe("trading.quotes.collect", "quotes-workers", func(msg *nats.Msg) {
		a.handleMessage(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	log.Println("QuotesAgent running — listening on trading.quotes.collect (queue: quotes-workers)")
	<-ctx.Done()
	log.Println("QuotesAgent shutting down")
	return nil
}

func (a *QuotesAgent) handleMessage(ctx context.Context, msg *nats.Msg) {
	_, span := a.tracer.Start(ctx, "quotes-agent.handle")
	defer span.End()

	var in incomingMsg
	if err := json.Unmarshal(msg.Data, &in); err != nil {
		log.Printf("[ERROR] failed to unmarshal quotes request: %v", err)
		return
	}

	log.Printf("[INFO] Received quotes request task_id=%s symbols=%v", in.TaskID, in.Symbols)

	quotes := generateQuotes(in.Symbols)

	out := outgoingMsg{
		TaskID: in.TaskID,
		Quotes: quotes,
		Status: "ok",
	}

	data, err := json.Marshal(out)
	if err != nil {
		log.Printf("[ERROR] failed to marshal quotes response: %v", err)
		return
	}

	if err := a.nc.Publish("trading.quotes.done", data); err != nil {
		log.Printf("[ERROR] failed to publish quotes.done: %v", err)
		return
	}

	atomic.AddInt64(&a.processedCount, 1)
	log.Printf("[INFO] Published quotes.done task_id=%s count=%d", in.TaskID, atomic.LoadInt64(&a.processedCount))
}

// generateQuote generates a single random market quote for the given symbol.
func generateQuote(symbol string) QuoteEntry {
	price := 100.0 + rand.Float64()*900.0
	volume := int64(1000 + rand.Intn(99001))
	return QuoteEntry{
		Symbol:    symbol,
		Price:     roundFloat(price, 2),
		Volume:    volume,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// generateQuotes creates a quote for each symbol in the slice.
func generateQuotes(symbols []string) []QuoteEntry {
	quotes := make([]QuoteEntry, 0, len(symbols))
	for _, sym := range symbols {
		quotes = append(quotes, generateQuote(sym))
	}
	return quotes
}

func roundFloat(val float64, precision int) float64 {
	factor := 1.0
	for i := 0; i < precision; i++ {
		factor *= 10
	}
	return float64(int(val*factor+0.5)) / factor
}
