package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"github.com/nebulacloud/nebula/internal/jobs"
)

// Named queues isolate job types so build-worker and runtime-agent do not
// compete for the same tasks (both previously listened on "critical").
const (
	QueueBuild  = "build"
	QueueDeploy = "deploy"
)

const defaultQueue = QueueBuild

// AsynqProducer implements Producer using github.com/hibiken/asynq.
type AsynqProducer struct {
	client *asynq.Client
}

// NewAsynqProducer builds a Producer from a Redis address (host:port or URL).
func NewAsynqProducer(redisAddr string, redisPassword string, redisDB int) *AsynqProducer {
	opt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	}
	return &AsynqProducer{client: asynq.NewClient(opt)}
}

// Enqueue implements Producer.
func (p *AsynqProducer) Enqueue(ctx context.Context, job Job) (string, error) {
	_ = ctx
	var task *asynq.Task
	switch job.Type {
	case JobTypeBuildRun:
		task = asynq.NewTask(jobs.TaskBuild, job.Payload)
	case JobTypeDeployRun:
		task = asynq.NewTask(jobs.TaskDeploy, job.Payload)
	default:
		return "", fmt.Errorf("queue: unknown job type %q", job.Type)
	}
	queue := job.Queue
	if queue == "" {
		queue = defaultQueue
	}
	opts := []asynq.Option{asynq.Queue(queue)}
	if job.MaxRetries > 0 {
		opts = append(opts, asynq.MaxRetry(job.MaxRetries))
	}
	if job.Timeout > 0 {
		opts = append(opts, asynq.Timeout(job.Timeout))
	}
	if !job.ProcessAt.IsZero() {
		opts = append(opts, asynq.ProcessAt(job.ProcessAt))
	}
	info, err := p.client.Enqueue(task, opts...)
	if err != nil {
		return "", err
	}
	return info.ID, nil
}

// Close implements Producer.
func (p *AsynqProducer) Close() error {
	return p.client.Close()
}

// AsynqWorker runs asynq workers for the supplied handlers.
type AsynqWorker struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

// NewAsynqWorker constructs a Worker that listens only on the given queues.
// Pass BuildWorkerQueues() or DeployWorkerQueues() from each cmd binary.
// errorHandler is optional (runtime-agent uses it to mark deployments failed).
func NewAsynqWorker(redisAddr, redisPassword string, redisDB int, queues map[string]int, errorHandler asynq.ErrorHandler) *AsynqWorker {
	if len(queues) == 0 {
		queues = BuildWorkerQueues()
	}
	cfg := asynq.Config{
		Concurrency: 4,
		Queues:      queues,
	}
	if errorHandler != nil {
		cfg.ErrorHandler = errorHandler
	}
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword, DB: redisDB},
		cfg,
	)
	return &AsynqWorker{server: srv, mux: asynq.NewServeMux()}
}

// BuildWorkerQueues returns queue weights for build-worker only.
func BuildWorkerQueues() map[string]int {
	return map[string]int{QueueBuild: 10}
}

// DeployWorkerQueues returns queue weights for runtime-agent only.
func DeployWorkerQueues() map[string]int {
	return map[string]int{QueueDeploy: 10}
}

// Register implements Worker.
func (w *AsynqWorker) Register(jobType JobType, fn HandlerFunc) {
	var name string
	switch jobType {
	case JobTypeBuildRun:
		name = jobs.TaskBuild
	case JobTypeDeployRun:
		name = jobs.TaskDeploy
	default:
		return
	}
	w.mux.HandleFunc(name, func(ctx context.Context, t *asynq.Task) error {
		return fn(ctx, Job{Type: jobType, Payload: t.Payload()})
	})
}

// Run implements Worker.
func (w *AsynqWorker) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- w.server.Run(w.mux)
	}()
	select {
	case <-ctx.Done():
		w.server.Shutdown()
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// Close implements Worker.
func (w *AsynqWorker) Close() error {
	w.server.Shutdown()
	return nil
}

// MustMarshalJSON encodes v or panics (startup / enqueue paths only).
func MustMarshalJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// DefaultBuildJobTimeout caps long builds.
func DefaultBuildJobTimeout() time.Duration { return 45 * time.Minute }

// DefaultDeployJobTimeout caps pull+run.
func DefaultDeployJobTimeout() time.Duration { return 15 * time.Minute }
