package state

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyTotalAssessments = "risk:total_assessments"
	keyHighRiskCount    = "risk:high_risk_count"
	lastRiskTTL         = time.Hour
)

// RiskState provides Redis-backed counters and last-risk storage for the risk manager.
type RiskState struct {
	client *redis.Client
}

// NewRiskState creates a RiskState connected to the given Redis URL (e.g. "redis://redis:6379").
func NewRiskState(redisURL string) (*RiskState, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL %q: %w", redisURL, err)
	}
	client := redis.NewClient(opts)

	// Ping to verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("cannot connect to Redis: %w", err)
	}

	return &RiskState{client: client}, nil
}

// IncrementTotal atomically increments the total assessments counter.
func (s *RiskState) IncrementTotal(ctx context.Context) error {
	return s.client.Incr(ctx, keyTotalAssessments).Err()
}

// IncrementHighRisk atomically increments the high-risk assessments counter.
func (s *RiskState) IncrementHighRisk(ctx context.Context) error {
	return s.client.Incr(ctx, keyHighRiskCount).Err()
}

// GetStats returns the total and high-risk assessment counts.
func (s *RiskState) GetStats(ctx context.Context) (total, highRisk int64, err error) {
	total, err = s.client.Get(ctx, keyTotalAssessments).Int64()
	if err == redis.Nil {
		total, err = 0, nil
	}
	if err != nil {
		return 0, 0, fmt.Errorf("get total: %w", err)
	}

	highRisk, err = s.client.Get(ctx, keyHighRiskCount).Int64()
	if err == redis.Nil {
		highRisk, err = 0, nil
	}
	if err != nil {
		return total, 0, fmt.Errorf("get high_risk: %w", err)
	}

	return total, highRisk, nil
}

// SaveLastRisk saves the risk level for a given task with a 1-hour TTL.
func (s *RiskState) SaveLastRisk(ctx context.Context, taskID string, level string) error {
	key := fmt.Sprintf("risk:last:%s", taskID)
	return s.client.Set(ctx, key, level, lastRiskTTL).Err()
}

// Close releases the Redis connection.
func (s *RiskState) Close() error {
	return s.client.Close()
}
