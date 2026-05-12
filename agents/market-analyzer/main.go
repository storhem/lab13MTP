package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"

	"github.com/storhem/lab13MTP/agents/market-analyzer/internal/agent"
	"github.com/storhem/lab13MTP/agents/market-analyzer/internal/tracing"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("Starting market-analyzer")

	shutdown, err := tracing.InitTracer("market-analyzer")
	if err != nil {
		log.Printf("[WARN] tracing init failed: %v — continuing without traces", err)
	} else {
		defer shutdown()
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS at %s: %v", natsURL, err)
	}
	defer nc.Close()
	log.Printf("Connected to NATS at %s", natsURL)

	tracer := otel.Tracer("market-analyzer")
	a := agent.NewMarketAnalyzerAgent(nc, tracer)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := a.Run(ctx); err != nil {
		log.Fatalf("Agent error: %v", err)
	}

	log.Println("market-analyzer stopped")
}
