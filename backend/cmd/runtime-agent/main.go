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
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nebulacloud/nebula/internal/jobs"
	projectsinfra "github.com/nebulacloud/nebula/internal/modules/projects/infrastructure"
	"github.com/nebulacloud/nebula/internal/platform/config"
	"github.com/nebulacloud/nebula/internal/platform/database"
	"github.com/nebulacloud/nebula/internal/platform/logstream"
	"github.com/nebulacloud/nebula/internal/platform/logger"
	platformqueue "github.com/nebulacloud/nebula/internal/platform/queue"
	platformredis "github.com/nebulacloud/nebula/internal/platform/redis"
	"github.com/nebulacloud/nebula/internal/platform/routing"
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

	logPub := logstream.NewPublisher(cfg.Redis.Address(), cfg.Redis.Password, cfg.Redis.DB)
	defer func() { _ = logPub.Close() }()

	repo := projectsinfra.NewRepository(pool)
	w := platformqueue.NewAsynqWorker(cfg.Redis.Address(), cfg.Redis.Password, cfg.Redis.DB, platformqueue.DeployWorkerQueues(), deployExhaustedHandler(repo))
	w.Register(platformqueue.JobTypeDeployRun, deployHandler(pool, cfg, logPub))

	log.Info("runtime-agent.ready", "docker_host", cfg.Runtime.DockerHost, "base_domain", cfg.Runtime.BaseDomain)

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

func deployHandler(pool *pgxpool.Pool, cfg config.Config, logPub *logstream.Publisher) platformqueue.HandlerFunc {
	repo := projectsinfra.NewRepository(pool)

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
		publishDeployLog(ctx, logPub, p.DeploymentID, "deploy.start image="+strings.TrimSpace(p.ImageRef), "info")

		listenPort := listenOr(p.ListenPort)
		routesDir := strings.TrimSpace(cfg.Runtime.TraefikUserRoutesDir)
		route := routing.NewDeployRoute(routesDir, p.ServiceSlug, p.ProjectSlug, p.BaseDomain, svcID, listenPort)

		img := strings.TrimSpace(p.ImageRef)
		publishDeployLog(ctx, logPub, p.DeploymentID, "pulling image", "info")
		pull := exec.CommandContext(ctx, "docker", "pull", img)
		o, err := pull.CombinedOutput()
		if err != nil {
			pullErr := fmt.Errorf("docker pull: %w — %s", err, strings.TrimSpace(string(o)))
			publishDeployLog(ctx, logPub, p.DeploymentID, "deploy failed: "+shorten(pullErr.Error()), "error")
			return fail(repo, ctx, routesDir, route.RouteID, depID, svcID, pullErr)
		}
		publishDeployLog(ctx, logPub, p.DeploymentID, "image pulled", "info")

		publishDeployLog(ctx, logPub, p.DeploymentID, "replacing previous container if any", "info")
		if err := ensureContainerSlotFree(ctx, route.ContainerName, svcID); err != nil {
			slotErr := fmt.Errorf("free container slot: %w", err)
			publishDeployLog(ctx, logPub, p.DeploymentID, "deploy failed: "+shorten(slotErr.Error()), "error")
			return fail(repo, ctx, routesDir, route.RouteID, depID, svcID, slotErr)
		}

		netName := strings.TrimSpace(cfg.Runtime.Network)
		if netName == "" {
			netName = "nebula_platform"
		}
		defaultRule := fmt.Sprintf("Host(`%s`)", route.Host)

		runArgs := []string{
			"run", "-d",
			"--name", route.ContainerName,
			"--network", netName,
			"--restart", "unless-stopped",
			"--label", "traefik.enable=true",
			"--label", "traefik.docker.network="+netName,
			"--label", fmt.Sprintf("traefik.http.routers.%s.rule=%s", route.RouteID, defaultRule),
			"--label", fmt.Sprintf("traefik.http.routers.%s.entrypoints=web", route.RouteID),
			"--label", fmt.Sprintf("traefik.http.routers.%s.service=%s", route.RouteID, route.RouteID),
			"--label", fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port=%d", route.RouteID, listenPort),
			"--label", fmt.Sprintf("nebula_service=%s", svcID.String()),
			"--label", fmt.Sprintf("nebula_deployment_id=%s", depID.String()),
			"--label", fmt.Sprintf("nebula_project=%s", p.ProjectSlug),
			"--label", fmt.Sprintf("nebula_org=%s", p.OrgSlug),
		}
		domains, _ := repo.ListDomainsByService(ctx, svcID)
		for i, d := range domains {
			if d.SSLStatus != "issued" {
				continue
			}
			customID := fmt.Sprintf("%s-d%d", route.RouteID, i)
			customRule := fmt.Sprintf("Host(`%s`)", d.Hostname)
			runArgs = append(runArgs,
				"--label", fmt.Sprintf("traefik.http.routers.%s.rule=%s", customID, customRule),
				"--label", fmt.Sprintf("traefik.http.routers.%s.entrypoints=websecure", customID),
				"--label", fmt.Sprintf("traefik.http.routers.%s.tls=true", customID),
				"--label", fmt.Sprintf("traefik.http.routers.%s.tls.certresolver=letsencrypt", customID),
				"--label", fmt.Sprintf("traefik.http.routers.%s.service=%s", customID, route.RouteID),
			)
		}
		runArgs = append(runArgs, img)

		publishDeployLog(ctx, logPub, p.DeploymentID, "starting container", "info")
		o2, err := dockerRun(ctx, runArgs)
		if err != nil && isNameConflict(err, string(o2)) {
			publishDeployLog(ctx, logPub, p.DeploymentID, "container name conflict, retrying cleanup", "warn")
			if cleanErr := ensureContainerSlotFree(ctx, route.ContainerName, svcID); cleanErr != nil {
				runErr := fmt.Errorf("docker run: %w — %s (cleanup: %v)", err, strings.TrimSpace(string(o2)), cleanErr)
				publishDeployLog(ctx, logPub, p.DeploymentID, "deploy failed: "+shorten(runErr.Error()), "error")
				return fail(repo, ctx, routesDir, route.RouteID, depID, svcID, runErr)
			}
			o2, err = dockerRun(ctx, runArgs)
		}
		if err != nil {
			runErr := fmt.Errorf("docker run: %w — %s", err, strings.TrimSpace(string(o2)))
			publishDeployLog(ctx, logPub, p.DeploymentID, "deploy failed: "+shorten(runErr.Error()), "error")
			return fail(repo, ctx, routesDir, route.RouteID, depID, svcID, runErr)
		}
		_ = o2

		publishDeployLog(ctx, logPub, p.DeploymentID, "waiting for container health", "info")
		if err := routing.WaitContainerReady(ctx, route.ContainerName, listenPort, 60*time.Second); err != nil {
			publishDeployLog(ctx, logPub, p.DeploymentID, "health check: "+shorten(err.Error()), "warn")
		}

		publishDeployLog(ctx, logPub, p.DeploymentID, "registering traefik route (file provider)", "info")
		verify := routing.VerifyOptions{
			TraefikAPIURL: cfg.Runtime.TraefikAPIURL,
			Retries:       15,
			RetryDelay:    3 * time.Second,
			InitialDelay:  2 * time.Second,
		}
		routeErr := routing.SyncDeployRoute(ctx, routesDir, route, verify)
		if routeErr != nil {
			publishDeployLog(ctx, logPub, p.DeploymentID, "traefik reload nudge (file provider lag)", "warn")
			_ = nudgeTraefikReload(ctx)
			verify.InitialDelay = 1 * time.Second
			routeErr = routing.SyncDeployRoute(ctx, routesDir, route, verify)
		}
		if routeErr != nil {
			wrapped := fmt.Errorf("traefik route: %w", routeErr)
			publishDeployLog(ctx, logPub, p.DeploymentID, "deploy failed: "+shorten(wrapped.Error()), "error")
			_ = exec.CommandContext(ctx, "docker", "rm", "-f", route.ContainerName).Run()
			return fail(repo, ctx, routesDir, route.RouteID, depID, svcID, wrapped)
		}
		publishDeployLog(ctx, logPub, p.DeploymentID, "traefik route verified", "info")

		removeOrphanServiceContainers(ctx, svcID, route.ContainerName)

		im := img
		if err := repo.UpdateDeploymentWorker(ctx, depID, "running", &im, nil); err != nil {
			log.Warn("deployment.running", "err", err)
		}
		if err := repo.PatchServiceRuntimeImage(ctx, svcID, &im, "running"); err != nil {
			log.Warn("service.running", "err", err)
		}
		publishDeployLog(ctx, logPub, p.DeploymentID, "deploy complete url="+route.PublicURL, "info")
		log.Info("deploy.done", "url", route.PublicURL, "route_file", route.FilePath, "port", listenPort)
		return nil
	}
}

func publishDeployLog(ctx context.Context, pub *logstream.Publisher, deploymentID, msg, level string) {
	if pub == nil {
		return
	}
	_ = pub.Publish(ctx, logstream.Line{
		DeploymentID: deploymentID,
		Message:      msg,
		Level:        level,
		Source:       "runtime",
		TS:           time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func shorten(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 4000 {
		return s[:4000] + "…"
	}
	return s
}

// nudgeTraefikReload asks the Traefik container to reload file config (helps Docker Desktop file watch).
func nudgeTraefikReload(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", "name=traefik", "--format", "{{.Names}}")
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	names := strings.Fields(strings.TrimSpace(string(out)))
	if len(names) == 0 {
		return fmt.Errorf("traefik container not found")
	}
	// SIGHUP is commonly used to reload config; harmless if ignored.
	return exec.CommandContext(ctx, "docker", "kill", "-s", "HUP", names[0]).Run()
}

func removeOrphanServiceContainers(ctx context.Context, serviceID uuid.UUID, keepName string) {
	filter := "label=nebula_service=" + serviceID.String()
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", filter, "--format", "{{.Names}}")
	out, err := cmd.Output()
	if err != nil {
		return
	}
	for _, name := range strings.Fields(strings.TrimSpace(string(out))) {
		if name == "" || name == keepName {
			continue
		}
		_ = exec.CommandContext(ctx, "docker", "rm", "-f", name).Run()
	}
}

func listenOr(p int) int {
	if p <= 0 {
		return 8080
	}
	return p
}

func fail(repo *projectsinfra.Repository, ctx context.Context, routesDir, routeID string, depID, svcID uuid.UUID, err error) error {
	routing.RemoveUserRoute(routesDir, routeID)
	msg := err.Error()
	_ = repo.UpdateDeploymentWorker(ctx, depID, "failed", nil, &msg)
	_ = repo.PatchServiceRuntimeImage(ctx, svcID, nil, "failed")
	return err
}

// deployExhaustedHandler marks deployments failed when asynq gives up retrying deploy.run.
func deployExhaustedHandler(repo *projectsinfra.Repository) asynq.ErrorHandler {
	return asynq.ErrorHandlerFunc(func(ctx context.Context, t *asynq.Task, err error) {
		if t.Type() != jobs.TaskDeploy || err == nil {
			return
		}
		retry, _ := asynq.GetRetryCount(ctx)
		max, _ := asynq.GetMaxRetry(ctx)
		if retry < max {
			return
		}
		var p jobs.DeployRunPayload
		if json.Unmarshal(t.Payload(), &p) != nil {
			return
		}
		depID, parseErr := uuid.Parse(p.DeploymentID)
		if parseErr != nil {
			return
		}
		svcID, parseErr := uuid.Parse(p.ServiceID)
		if parseErr != nil {
			return
		}
		_ = fail(repo, ctx, "", "", depID, svcID, err)
	})
}
