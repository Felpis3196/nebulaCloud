// Command build-worker drains asynq build jobs (git clone → docker build → push → enqueue deploy.run).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nebulacloud/nebula/internal/jobs"
	projectsinfra "github.com/nebulacloud/nebula/internal/modules/projects/infrastructure"
	"github.com/nebulacloud/nebula/internal/platform/config"
	"github.com/nebulacloud/nebula/internal/platform/database"
	"github.com/nebulacloud/nebula/internal/platform/logger"
	platformqueue "github.com/nebulacloud/nebula/internal/platform/queue"
)

func main() {
	if err := run(); err != nil && !strings.Contains(strings.ToLower(err.Error()), "canceled") {
		_, _ = fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.ServiceName = "nebula-build-worker"

	log := logger.New(logger.Options{
		Level:       cfg.LogLevel,
		ServiceName: cfg.ServiceName,
		Environment: string(cfg.Env),
	})
	logger.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx = logger.WithLogger(ctx, log.With())

	dbCtx, dbCancel := context.WithTimeout(ctx, 30*time.Second)
	defer dbCancel()
	pool, err := database.Connect(dbCtx, database.Config{
		DSN:             cfg.Database.DSN(),
		MaxConns:        cfg.Database.MaxConns,
		MinConns:        cfg.Database.MinConns,
		MaxConnLifetime: cfg.Database.MaxConnLifetime,
	})
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}
	defer pool.Close()

	producer := platformqueue.NewAsynqProducer(cfg.Redis.Address(), cfg.Redis.Password, cfg.Redis.DB)
	defer func() { _ = producer.Close() }()

	w := platformqueue.NewAsynqWorker(cfg.Redis.Address(), cfg.Redis.Password, cfg.Redis.DB)
	w.Register(platformqueue.JobTypeBuildRun, buildHandler(pool, cfg, producer))

	log.Info("build-worker.ready")
	err = w.Run(ctx)
	stop()
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func buildHandler(pool *pgxpool.Pool, cfg config.Config, prod platformqueue.Producer) platformqueue.HandlerFunc {
	repo := projectsinfra.NewRepository(pool)
	host, _ := os.Hostname()

	return func(ctx context.Context, job platformqueue.Job) error {
		var p jobs.BuildRunPayload
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return fmt.Errorf("payload json: %w", err)
		}
		buildID, err := uuid.Parse(p.BuildID)
		if err != nil {
			return err
		}
		depID, err := uuid.Parse(p.DeploymentID)
		if err != nil {
			return err
		}

		log := slog.Default().With("build_id", p.BuildID, "deployment_id", p.DeploymentID)
		log.Info("build.start")

		bc, err := repo.LoadBuildJobContext(ctx, buildID)
		if err != nil {
			return err
		}

		_ = repo.UpdateBuildFields(ctx, buildID, host, "cloning", ptr("docker"), nil, nil)
		workDir, err := os.MkdirTemp("", "nebula-build-*")
		if err != nil {
			return fail(repo, ctx, host, depID, buildID, fmt.Errorf("tmpdir: %w", err))
		}
		defer func() { _ = os.RemoveAll(workDir) }()

		branch := strings.TrimSpace(bc.Ref)
		if branch == "" {
			branch = "main"
		}
		g := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", branch, strings.TrimSpace(bc.RepoURL), ".")
		g.Dir = workDir
		out, err := g.CombinedOutput()
		if err != nil {
			log.Error("git.clone", "err", err, "out", string(out))
			return fail(repo, ctx, host, depID, buildID, fmt.Errorf("git clone: %w — %s", err, shorten(string(out))))
		}

		_ = repo.UpdateBuildFields(ctx, buildID, host, "building", ptr("docker"), nil, nil)
		img := strings.TrimSpace(p.ImageRef)
		dfile := filepath.Join(workDir, dockerfile(bc.BuildConfig))
		if _, err := os.Stat(dfile); err != nil {
			return fail(repo, ctx, host, depID, buildID, fmt.Errorf("%s not found", filepath.Base(dfile)))
		}

		build := exec.CommandContext(ctx, "docker", "build", "-t", img, "-f", filepath.Base(dfile), ".")
		build.Dir = workDir
		build.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
		o2, err := build.CombinedOutput()
		if err != nil {
			log.Error("docker.build", "err", err, "out", string(o2))
			return fail(repo, ctx, host, depID, buildID, fmt.Errorf("docker build: %w — %s", err, shorten(string(o2))))
		}

		_ = repo.UpdateBuildFields(ctx, buildID, host, "pushing", ptr("docker"), nil, nil)

		psh := exec.CommandContext(ctx, "docker", "push", img)
		o3, err := psh.CombinedOutput()
		if err != nil {
			log.Error("docker.push", "err", err, "out", string(o3))
			return fail(repo, ctx, host, depID, buildID, fmt.Errorf("docker push: %w — %s", err, shorten(string(o3))))
		}

		zero := 0
		if err := repo.UpdateBuildFields(ctx, buildID, host, "success", ptr("docker"), nil, &zero); err != nil {
			return err
		}

		im := img
		if err := repo.UpdateDeploymentWorker(ctx, depID, "deploying", &im, nil); err != nil {
			log.Warn("deployment.deploying_marker", "err", err)
		}

		pl := jobs.DeployRunPayload{
			DeploymentID:   depID.String(),
			ServiceID:      bc.ServiceID.String(),
			OrganizationID: bc.OrganizationID.String(),
			ProjectID:      bc.ProjectID.String(),
			OrgSlug:        bc.OrgSlug,
			ProjectSlug:    bc.ProjectSlug,
			ServiceSlug:    bc.ServiceSlug,
			ImageRef:       img,
			ListenPort:     listen(bc.RuntimeConfig),
			BaseDomain:     cfg.Runtime.BaseDomain,
		}
		if _, err := prod.Enqueue(ctx, platformqueue.Job{
			Type:       platformqueue.JobTypeDeployRun,
			Payload:    platformqueue.MustMarshalJSON(&pl),
			MaxRetries: 2,
			Timeout:    platformqueue.DefaultDeployJobTimeout(),
			Queue:      "critical",
		}); err != nil {
			return fmt.Errorf("enqueue deploy: %w", err)
		}
		log.Info("build.done", "image", img)
		return nil
	}
}

func dockerfile(rc json.RawMessage) string {
	var m map[string]any
	_ = json.Unmarshal(rc, &m)
	if v, ok := m["dockerfile_path"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return "Dockerfile"
}

func listen(rt json.RawMessage) int {
	var m map[string]any
	_ = json.Unmarshal(rt, &m)
	if v, ok := m["listen_port"].(float64); ok && int(v) > 0 {
		return int(v)
	}
	if v, ok := m["port"].(float64); ok && int(v) > 0 {
		return int(v)
	}
	return 8080
}

func ptr(s string) *string { return &s }

func shorten(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 4000 {
		return s[:4000] + "…"
	}
	return s
}

func fail(repo *projectsinfra.Repository, ctx context.Context, wid string, dep, build uuid.UUID, err error) error {
	msg := err.Error()
	code := 1
	_ = repo.UpdateBuildFields(ctx, build, wid, "failed", nil, &msg, &code)
	_ = repo.UpdateDeploymentWorker(ctx, dep, "failed", nil, &msg)
	if sid, e2 := repo.DeploymentServiceID(ctx, dep); e2 == nil {
		_ = repo.PatchServiceRuntimeImage(ctx, sid, nil, "failed")
	}
	return err
}
