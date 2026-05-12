package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"

	"github.com/storhem/lab13MTP/agents/quotes-agent/internal/agent"
	"github.com/storhem/lab13MTP/agents/quotes-agent/internal/tracing"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("Starting quotes-agent")

	// Initialise OpenTelemetry tracer
	shutdown, err := tracing.InitTracer("quotes-agent")
	if err != nil {
		log.Printf("[WARN] tracing init failed: %v — continuing without traces", err)
	} else {
		defer shutdown()
	}

	// Connect to NATS
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

	tracer := otel.Tracer("quotes-agent")
	a := agent.NewQuotesAgent(nc, tracer)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := a.Run(ctx); err != nil {
		log.Fatalf("Agent error: %v", err)
	}

	log.Println("quotes-agent stopped")
}
