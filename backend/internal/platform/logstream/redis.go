// Package logstream publishes build/runtime log lines over Redis pub/sub for
// WebSocket fan-out in the API.
package logstream

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	historyMaxLines = 500
	historyTTL      = 24 * time.Hour
)

// Line is one log event streamed to clients.
type Line struct {
	DeploymentID string `json:"deployment_id"`
	Level        string `json:"level,omitempty"`
	Message      string `json:"message"`
	TS           string `json:"ts,omitempty"`
	Source       string `json:"source,omitempty"` // build | runtime
}

func channel(deploymentID string) string {
	return "nebula:logs:" + strings.TrimSpace(deploymentID)
}

func historyKey(deploymentID string) string {
	return "nebula:logs:hist:" + strings.TrimSpace(deploymentID)
}

// Publisher emits log lines to Redis.
type Publisher struct {
	rdb *goredis.Client
}

// NewPublisher connects to Redis at addr (host:port).
func NewPublisher(addr, password string, db int) *Publisher {
	return &Publisher{
		rdb: goredis.NewClient(&goredis.Options{Addr: addr, Password: password, DB: db}),
	}
}

// Close releases the client.
func (p *Publisher) Close() error {
	if p == nil || p.rdb == nil {
		return nil
	}
	return p.rdb.Close()
}

// Publish sends one line to subscribers and appends to deployment history.
func (p *Publisher) Publish(ctx context.Context, line Line) error {
	if p == nil || p.rdb == nil {
		return nil
	}
	if strings.TrimSpace(line.DeploymentID) == "" {
		return fmt.Errorf("logstream: missing deployment_id")
	}
	if strings.TrimSpace(line.TS) == "" {
		line.TS = time.Now().UTC().Format(time.RFC3339Nano)
	}
	b, err := json.Marshal(line)
	if err != nil {
		return err
	}
	depID := strings.TrimSpace(line.DeploymentID)
	pipe := p.rdb.Pipeline()
	pipe.Publish(ctx, channel(depID), b)
	hKey := historyKey(depID)
	pipe.RPush(ctx, hKey, b)
	pipe.LTrim(ctx, hKey, int64(-historyMaxLines), -1)
	pipe.Expire(ctx, hKey, historyTTL)
	_, err = pipe.Exec(ctx)
	return err
}

// Subscriber receives log lines for a deployment.
type Subscriber struct {
	rdb *goredis.Client
}

// NewSubscriber connects to Redis.
func NewSubscriber(addr, password string, db int) *Subscriber {
	return &Subscriber{
		rdb: goredis.NewClient(&goredis.Options{Addr: addr, Password: password, DB: db}),
	}
}

// Close releases the client.
func (s *Subscriber) Close() error {
	if s == nil || s.rdb == nil {
		return nil
	}
	return s.rdb.Close()
}

// History returns persisted log lines for a deployment (oldest first).
func (s *Subscriber) History(ctx context.Context, deploymentID string, limit int) ([]Line, error) {
	if s == nil || s.rdb == nil {
		return nil, fmt.Errorf("logstream: no redis")
	}
	depID := strings.TrimSpace(deploymentID)
	if depID == "" {
		return nil, fmt.Errorf("logstream: missing deployment_id")
	}
	if limit <= 0 || limit > historyMaxLines {
		limit = 200
	}
	raw, err := s.rdb.LRange(ctx, historyKey(depID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	start := 0
	if len(raw) > limit {
		start = len(raw) - limit
	}
	out := make([]Line, 0, len(raw)-start)
	for _, item := range raw[start:] {
		var line Line
		if err := json.Unmarshal([]byte(item), &line); err != nil {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

// Subscribe returns a channel of lines until ctx is canceled.
func (s *Subscriber) Subscribe(ctx context.Context, deploymentID string) (<-chan Line, error) {
	if s == nil || s.rdb == nil {
		return nil, fmt.Errorf("logstream: no redis")
	}
	pubsub := s.rdb.Subscribe(ctx, channel(deploymentID))
	ch := make(chan Line, 64)
	go func() {
		defer close(ch)
		defer func() { _ = pubsub.Close() }()
		for {
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				return
			}
			var line Line
			if err := json.Unmarshal([]byte(msg.Payload), &line); err != nil {
				continue
			}
			select {
			case ch <- line:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}
