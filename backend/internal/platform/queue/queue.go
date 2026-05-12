// Package queue defines the abstract job-queue contract used by the build
// pipeline. The concrete implementation (added in Phase 4) is asynq on top
// of Redis. Keeping the interface here makes it possible to swap to River,
// NATS JetStream, or AWS SQS without rewriting modules.
package queue

import (
	"context"
	"time"
)

// JobType is a stable identifier the worker selects on.
type JobType string

// Canonical job types.
const (
	JobTypeBuildRun  JobType = "build.run"
	JobTypeDeployRun JobType = "deploy.run"
	JobTypeRollback  JobType = "deploy.rollback"
)

// Job is the envelope dispatched by Producers and consumed by Workers.
type Job struct {
	Type          JobType
	Payload       []byte
	CorrelationID string
	MaxRetries    int
	Timeout       time.Duration
	ProcessAt     time.Time // zero means "process ASAP"
	Queue         string    // optional named queue (e.g. "builds")
}

// Producer enqueues jobs.
type Producer interface {
	Enqueue(ctx context.Context, job Job) (jobID string, err error)
	Close() error
}

// HandlerFunc processes a single Job.
type HandlerFunc func(ctx context.Context, job Job) error

// Worker registers handlers and runs a processing loop.
type Worker interface {
	Register(jobType JobType, handler HandlerFunc)
	Run(ctx context.Context) error
	Close() error
}
