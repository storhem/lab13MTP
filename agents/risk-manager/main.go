package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"

	"github.com/storhem/lab13MTP/agents/risk-manager/internal/agent"
	"github.com/storhem/lab13MTP/agents/risk-manager/internal/state"
	"github.com/storhem/lab13MTP/agents/risk-manager/internal/tracing"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("Starting risk-manager")

	shutdown, err := tracing.InitTracer("risk-manager")
	if err != nil {
		log.Printf("[WARN] tracing init failed: %v — continuing without traces", err)
	} else {
		defer shutdown()
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS at %s: %v", natsURL, err)
	}
	defer nc.Close()
	log.Printf("Connected to NATS at %s", natsURL)

	riskState, err := state.NewRiskState(redisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis at %s: %v", redisURL, err)
	}
	defer func() { _ = riskState.Close() }()
	log.Printf("Connected to Redis at %s", redisURL)

	tracer := otel.Tracer("risk-manager")
	a := agent.NewRiskManagerAgent(nc, tracer, riskState)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := a.Run(ctx); err != nil {
		log.Fatalf("Agent error: %v", err)
	}

	log.Println("risk-manager stopped")
}
