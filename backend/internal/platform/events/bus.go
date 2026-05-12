// Package events defines the abstract event bus used to decouple modules
// that emit domain events (e.g. "deployment.requested") from the workers
// that consume them.
//
// The default implementation (added in Phase 4) is a Redis-Streams-backed
// bus; the interface intentionally stays narrow so a future migration to
// NATS or Kafka requires no changes in callers.
package events

import (
	"context"
	"encoding/json"
	"time"
)

// Topic is a strongly-typed event channel.
type Topic string

// Canonical platform topics.
const (
	TopicBuildRequested    Topic = "build.requested"
	TopicBuildProgress     Topic = "build.progress"
	TopicBuildCompleted    Topic = "build.completed"
	TopicDeployRequested   Topic = "deploy.requested"
	TopicDeployCompleted   Topic = "deploy.completed"
	TopicDeployFailed      Topic = "deploy.failed"
	TopicServiceRestarted  Topic = "service.restarted"
	TopicServiceUnhealthy  Topic = "service.unhealthy"
)

// Event is the envelope for a single message on the bus.
type Event struct {
	ID            string          `json:"id"`
	Topic         Topic           `json:"topic"`
	OccurredAt    time.Time       `json:"occurred_at"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	Source        string          `json:"source,omitempty"`
	Payload       json.RawMessage `json:"payload"`
}

// Decode unmarshals the raw payload into the supplied destination.
func (e Event) Decode(dst any) error { return json.Unmarshal(e.Payload, dst) }

// Handler processes a single event. Returning an error signals the bus to
// retry (with backoff) or dead-letter according to its policy.
type Handler func(ctx context.Context, e Event) error

// Bus is the minimum surface every backing store must implement.
type Bus interface {
	// Publish sends an event to the named topic. The implementation is
	// responsible for assigning Event.ID if not set.
	Publish(ctx context.Context, topic Topic, payload any) error

	// Subscribe registers a Handler for the named topic. The returned
	// cancel function unsubscribes; the bus may call the handler from
	// multiple goroutines.
	Subscribe(ctx context.Context, topic Topic, handler Handler) (cancel func(), err error)

	// Close shuts down the bus.
	Close() error
}
