// Command build-worker drains asynq build jobs (git clone → stack detect →
// docker build or pack build → push → enqueue deploy.run).
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
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nebulacloud/nebula/internal/buildworker"
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

		dfRel := strings.TrimSpace(p.DockerfilePath)

		_ = repo.UpdateBuildFields(ctx, buildID, host, "cloning", nil, nil, nil)
		workDir, err := os.MkdirTemp("", "nebula-build-*")
		if err != nil {
			return fail(repo, ctx, host, depID, buildID, fmt.Errorf("tmpdir: %w", err))
		}
		defer func() { _ = os.RemoveAll(workDir) }()

		if err := buildworker.CloneSource(ctx, workDir, strings.TrimSpace(bc.RepoURL), bc.Ref, bc.CommitSHA); err != nil {
			log.Error("git.checkout", "err", err)
			return fail(repo, ctx, host, depID, buildID, err)
		}

		_ = repo.UpdateBuildFields(ctx, buildID, host, "detecting", nil, nil, nil)
		mode, detected, err := buildworker.DetectStack(workDir, dfRel)
		if err != nil {
			return fail(repo, ctx, host, depID, buildID, err)
		}

		img := strings.TrimSpace(p.ImageRef)

		_ = repo.UpdateBuildFields(ctx, buildID, host, "building", &detected, nil, nil)
		switch mode {
		case buildworker.ModeDockerfile:
			if err := buildworker.RunDockerBuild(ctx, workDir, img, dfRel); err != nil {
				log.Error("docker.build", "err", err)
				return fail(repo, ctx, host, depID, buildID, err)
			}
		case buildworker.ModeBuildpack:
			if err := buildworker.RunPackBuild(ctx, workDir, img, cfg.Build.BuilderImage); err != nil {
				log.Error("pack.build", "err", err)
				return fail(repo, ctx, host, depID, buildID, err)
			}
		default:
			return fail(repo, ctx, host, depID, buildID, fmt.Errorf("unknown build mode"))
		}

		_ = repo.UpdateBuildFields(ctx, buildID, host, "pushing", &detected, nil, nil)

		psh := exec.CommandContext(ctx, "docker", "push", img)
		o3, err := psh.CombinedOutput()
		if err != nil {
			log.Error("docker.push", "err", err, "out", string(o3))
			return fail(repo, ctx, host, depID, buildID, fmt.Errorf("docker push: %w — %s", err, shorten(string(o3))))
		}

		zero := 0
		if err := repo.UpdateBuildFields(ctx, buildID, host, "success", &detected, nil, &zero); err != nil {
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
