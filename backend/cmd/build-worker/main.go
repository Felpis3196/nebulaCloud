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
	"github.com/nebulacloud/nebula/internal/platform/logstream"
	"github.com/nebulacloud/nebula/internal/platform/logger"
	platformqueue "github.com/nebulacloud/nebula/internal/platform/queue"
	platformredis "github.com/nebulacloud/nebula/internal/platform/redis"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := runHealthcheck(); err != nil {
			os.Exit(1)
		}
		return
	}
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

	logPub := logstream.NewPublisher(cfg.Redis.Address(), cfg.Redis.Password, cfg.Redis.DB)
	defer func() { _ = logPub.Close() }()

	w := platformqueue.NewAsynqWorker(cfg.Redis.Address(), cfg.Redis.Password, cfg.Redis.DB, platformqueue.BuildWorkerQueues(), nil)
	w.Register(platformqueue.JobTypeBuildRun, buildHandler(pool, cfg, producer, logPub))

	log.Info("build-worker.ready")
	err = w.Run(ctx)
	stop()
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func runHealthcheck() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := database.Connect(ctx, database.Config{DSN: cfg.Database.DSN()})
	if err != nil {
		return err
	}
	defer pool.Close()
	rdb, err := platformredis.Connect(ctx, platformredis.Config{
		Addr: cfg.Redis.Address(), Password: cfg.Redis.Password, DB: cfg.Redis.DB,
	})
	if err != nil {
		return err
	}
	defer func() { _ = rdb.Close() }()
	return nil
}

func buildHandler(pool *pgxpool.Pool, cfg config.Config, prod platformqueue.Producer, logPub *logstream.Publisher) platformqueue.HandlerFunc {
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
		publishBuildLog(ctx, logPub, p.DeploymentID, "build.start", "info")

		bc, err := repo.LoadBuildJobContext(ctx, buildID)
		if err != nil {
			return err
		}
		ref := strings.TrimSpace(bc.Ref)
		if ref == "" {
			ref = "main"
		}
		publishBuildLog(ctx, logPub, p.DeploymentID,
			fmt.Sprintf("repo=%s ref=%s", strings.TrimSpace(bc.RepoURL), ref), "info")

		dfRel := strings.TrimSpace(p.DockerfilePath)

		_ = repo.UpdateBuildFields(ctx, buildID, host, "cloning", nil, nil, nil)
		publishBuildLog(ctx, logPub, p.DeploymentID, "cloning repository", "info")
		workDir, err := os.MkdirTemp("", "nebula-build-*")
		if err != nil {
			return fail(repo, ctx, host, depID, buildID, logPub, p.DeploymentID, fmt.Errorf("tmpdir: %w", err))
		}
		defer func() { _ = os.RemoveAll(workDir) }()

		if err := buildworker.CloneSource(ctx, workDir, strings.TrimSpace(bc.RepoURL), bc.Ref, bc.CommitSHA); err != nil {
			log.Error("git.checkout", "err", err)
			return fail(repo, ctx, host, depID, buildID, logPub, p.DeploymentID, err)
		}
		shaNote := ref
		if bc.CommitSHA != nil {
			if s := strings.TrimSpace(*bc.CommitSHA); s != "" {
				shaNote = s
			}
		}
		publishBuildLog(ctx, logPub, p.DeploymentID, "source cloned ("+shaNote+")", "info")

		_ = repo.UpdateBuildFields(ctx, buildID, host, "detecting", nil, nil, nil)
		publishBuildLog(ctx, logPub, p.DeploymentID, "detecting stack", "info")
		mode, detected, err := buildworker.DetectStack(workDir, dfRel)
		if err != nil {
			return fail(repo, ctx, host, depID, buildID, logPub, p.DeploymentID, err)
		}
		publishBuildLog(ctx, logPub, p.DeploymentID, "stack="+detected, "info")

		img := strings.TrimSpace(p.ImageRef)

		_ = repo.UpdateBuildFields(ctx, buildID, host, "building", &detected, nil, nil)
		publishBuildLog(ctx, logPub, p.DeploymentID, "building image ("+detected+")", "info")
		switch mode {
		case buildworker.ModeDockerfile:
			if err := buildworker.RunDockerBuild(ctx, workDir, img, dfRel); err != nil {
				log.Error("docker.build", "err", err)
				publishErrorOutput(ctx, logPub, p.DeploymentID, err.Error())
				return fail(repo, ctx, host, depID, buildID, logPub, p.DeploymentID, err)
			}
		case buildworker.ModeBuildpack:
			if err := buildworker.RunPackBuild(ctx, workDir, img, cfg.Build.BuilderImage); err != nil {
				log.Error("pack.build", "err", err)
				publishErrorOutput(ctx, logPub, p.DeploymentID, err.Error())
				return fail(repo, ctx, host, depID, buildID, logPub, p.DeploymentID, err)
			}
		default:
			return fail(repo, ctx, host, depID, buildID, logPub, p.DeploymentID, fmt.Errorf("unknown build mode"))
		}

		_ = repo.UpdateBuildFields(ctx, buildID, host, "pushing", &detected, nil, nil)
		publishBuildLog(ctx, logPub, p.DeploymentID, "pushing image to registry", "info")

		psh := exec.CommandContext(ctx, "docker", "push", img)
		o3, err := psh.CombinedOutput()
		if err != nil {
			log.Error("docker.push", "err", err, "out", string(o3))
			pushErr := fmt.Errorf("docker push: %w — %s", err, shorten(string(o3)))
			publishErrorOutput(ctx, logPub, p.DeploymentID, pushErr.Error())
			return fail(repo, ctx, host, depID, buildID, logPub, p.DeploymentID, pushErr)
		}

		zero := 0
		if err := repo.UpdateBuildFields(ctx, buildID, host, "success", &detected, nil, &zero); err != nil {
			return err
		}

		im := img
		if err := repo.UpdateDeploymentWorker(ctx, depID, "deploying", &im, nil); err != nil {
			log.Warn("deployment.deploying_marker", "err", err)
		}

		listenPort, portSource := buildworker.ResolveListenPort(bc.RuntimeConfig, detected, img, ctx)
		if err := repo.PatchServiceRuntimeConfig(ctx, bc.ServiceID, listenPort, detected); err != nil {
			log.Warn("service.runtime_config", "err", err)
		}
		publishBuildLog(ctx, logPub, p.DeploymentID, fmt.Sprintf("listen_port=%d (from %s)", listenPort, portSource), "info")

		pl := jobs.DeployRunPayload{
			DeploymentID:   depID.String(),
			ServiceID:      bc.ServiceID.String(),
			OrganizationID: bc.OrganizationID.String(),
			ProjectID:      bc.ProjectID.String(),
			OrgSlug:        bc.OrgSlug,
			ProjectSlug:    bc.ProjectSlug,
			ServiceSlug:    bc.ServiceSlug,
			ImageRef:       img,
			ListenPort:     listenPort,
			DetectedStack:  detected,
			BaseDomain:     cfg.Runtime.BaseDomain,
		}
		if _, err := prod.Enqueue(ctx, platformqueue.Job{
			Type:       platformqueue.JobTypeDeployRun,
			Payload:    platformqueue.MustMarshalJSON(&pl),
			MaxRetries: 2,
			Timeout:    platformqueue.DefaultDeployJobTimeout(),
			Queue:      platformqueue.QueueDeploy,
		}); err != nil {
			return fmt.Errorf("enqueue deploy: %w", err)
		}
		publishBuildLog(ctx, logPub, p.DeploymentID, "build complete", "info")
		publishBuildLog(ctx, logPub, p.DeploymentID, "deploy job queued", "info")
		log.Info("build.done", "image", img)
		return nil
	}
}

func shorten(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 4000 {
		return s[:4000] + "…"
	}
	return s
}

func publishBuildLog(ctx context.Context, pub *logstream.Publisher, deploymentID, msg, level string) {
	if pub == nil {
		return
	}
	_ = pub.Publish(ctx, logstream.Line{
		DeploymentID: deploymentID,
		Message:      msg,
		Level:        level,
		Source:       "build",
		TS:           time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func publishErrorOutput(ctx context.Context, pub *logstream.Publisher, deploymentID, raw string) {
	publishBuildLog(ctx, pub, deploymentID, "build failed: "+shorten(raw), "error")
	for _, line := range tailLines(raw, 40) {
		publishBuildLog(ctx, pub, deploymentID, line, "error")
	}
}

func tailLines(s string, max int) []string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) <= max {
		return lines
	}
	return lines[len(lines)-max:]
}

func fail(repo *projectsinfra.Repository, ctx context.Context, wid string, dep, build uuid.UUID, pub *logstream.Publisher, deploymentID string, err error) error {
	msg := err.Error()
	code := 1
	publishBuildLog(ctx, pub, deploymentID, "build failed: "+shorten(msg), "error")
	_ = repo.UpdateBuildFields(ctx, build, wid, "failed", nil, &msg, &code)
	_ = repo.UpdateDeploymentWorker(ctx, dep, "failed", nil, &msg)
	if sid, e2 := repo.DeploymentServiceID(ctx, dep); e2 == nil {
		_ = repo.PatchServiceRuntimeImage(ctx, sid, nil, "failed")
	}
	return err
}
