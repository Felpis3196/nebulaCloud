// Command runtime-agent consumes deploy.run jobs (docker pull/run + Traefik labels).
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
	cfg.ServiceName = "nebula-runtime-agent"

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

	w := platformqueue.NewAsynqWorker(cfg.Redis.Address(), cfg.Redis.Password, cfg.Redis.DB)
	w.Register(platformqueue.JobTypeDeployRun, deployHandler(pool, cfg))

	log.Info("runtime-agent.ready", "docker_host", cfg.Runtime.DockerHost, "base_domain", cfg.Runtime.BaseDomain)

	err = w.Run(ctx)
	stop()
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func deployHandler(pool *pgxpool.Pool, cfg config.Config) platformqueue.HandlerFunc {
	repo := projectsinfra.NewRepository(pool)
	host, _ := os.Hostname()

	return func(ctx context.Context, job platformqueue.Job) error {
		var p jobs.DeployRunPayload
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return fmt.Errorf("deploy payload: %w", err)
		}
		depID, err := uuid.Parse(p.DeploymentID)
		if err != nil {
			return err
		}
		svcID, err := uuid.Parse(p.ServiceID)
		if err != nil {
			return err
		}

		log := slog.Default().With("deployment_id", p.DeploymentID, "service_id", p.ServiceID)
		log.Info("deploy.start", "image", p.ImageRef)

		pruneLabeledServiceContainers(ctx, svcID)

		cname := stableContainerName(svcID)
		img := strings.TrimSpace(p.ImageRef)
		pull := exec.CommandContext(ctx, "docker", "pull", img)
		o, err := pull.CombinedOutput()
		if err != nil {
			return fail(repo, ctx, host, depID, svcID, fmt.Errorf("docker pull: %w — %s", err, strings.TrimSpace(string(o))))
		}

		hostRule := sanitize(p.ServiceSlug) + "." + sanitize(p.ProjectSlug) + "." + sanitize(p.BaseDomain)
		routeID := "nebula-" + shortID(svcID)
		rule := fmt.Sprintf("Host(`%s`)", hostRule)

		runArgs := []string{
			"run", "-d",
			"--name", cname,
			"--network", "nebula_platform",
			"--restart", "unless-stopped",
			"--label", "traefik.enable=true",
			"--label", "traefik.docker.network=nebula_platform",
			"--label", fmt.Sprintf("traefik.http.routers.%s.rule=%s", routeID, rule),
			"--label", fmt.Sprintf("traefik.http.routers.%s.entrypoints=web", routeID),
			"--label", fmt.Sprintf("traefik.http.routers.%s.service=%s", routeID, routeID),
			"--label", fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port=%d", routeID, listenOr(p.ListenPort)),
			"--label", fmt.Sprintf("nebula_service=%s", svcID.String()),
			"--label", fmt.Sprintf("nebula_deployment_id=%s", depID.String()),
			"--label", fmt.Sprintf("nebula_project=%s", p.ProjectSlug),
			"--label", fmt.Sprintf("nebula_org=%s", p.OrgSlug),
			img,
		}

		run := exec.CommandContext(ctx, "docker", runArgs...)
		o2, err := run.CombinedOutput()
		if err != nil {
			return fail(repo, ctx, host, depID, svcID, fmt.Errorf("docker run: %w — %s", err, strings.TrimSpace(string(o2))))
		}
		_ = o2

		im := img
		if err := repo.UpdateDeploymentWorker(ctx, depID, "running", &im, nil); err != nil {
			log.Warn("deployment.running", "err", err)
		}
		if err := repo.PatchServiceRuntimeImage(ctx, svcID, &im, "running"); err != nil {
			log.Warn("service.running", "err", err)
		}
		log.Info("deploy.done", "url", "http://"+hostRule)
		return nil
	}
}

func pruneLabeledServiceContainers(ctx context.Context, serviceID uuid.UUID) {
	filter := "label=nebula_service=" + serviceID.String()
	cmd := exec.CommandContext(ctx, "docker", "ps", "-aq", "--filter", filter)
	out, err := cmd.Output()
	if err != nil {
		return
	}
	for _, id := range strings.Fields(strings.TrimSpace(string(out))) {
		if id == "" {
			continue
		}
		_ = exec.CommandContext(ctx, "docker", "rm", "-f", id).Run()
	}
}

func stableContainerName(svcID uuid.UUID) string {
	s := strings.ReplaceAll(strings.ToLower(svcID.String()), "-", "")
	if len(s) > 12 {
		s = s[:12]
	}
	return "nebula-svc-" + s
}

func shortID(id uuid.UUID) string {
	s := strings.ReplaceAll(strings.ToLower(id.String()), "-", "")
	if len(s) < 8 {
		return s + "00000000"
	}
	return s[:8]
}

func sanitize(s string) string {
	var b strings.Builder
	s = strings.ToLower(strings.TrimSpace(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
			continue
		}
		if r == '_' {
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "svc"
	}
	return out
}

func listenOr(p int) int {
	if p <= 0 {
		return 8080
	}
	return p
}

func fail(repo *projectsinfra.Repository, ctx context.Context, wid string, depID, svcID uuid.UUID, err error) error {
	msg := err.Error()
	_ = repo.UpdateDeploymentWorker(ctx, depID, "failed", nil, &msg)
	_ = repo.PatchServiceRuntimeImage(ctx, svcID, nil, "failed")
	return err
}
