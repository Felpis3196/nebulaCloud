// Package application hosts projects / deployments use cases (Phase 2+).
package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nebulacloud/nebula/internal/jobs"
	"github.com/nebulacloud/nebula/internal/modules/audit"
	projectsinfra "github.com/nebulacloud/nebula/internal/modules/projects/infrastructure"
	"github.com/nebulacloud/nebula/internal/platform/auth"
	"github.com/nebulacloud/nebula/internal/platform/config"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
	"github.com/nebulacloud/nebula/internal/platform/logger"
	"github.com/nebulacloud/nebula/internal/platform/queue"
	"github.com/nebulacloud/nebula/internal/platform/secrets"
	"github.com/nebulacloud/nebula/internal/platform/telemetry"
)

// Service orchestrates control-plane project operations and build enqueue.
type Service struct {
	repo     *projectsinfra.Repository
	sealer   secrets.Sealer
	producer queue.Producer
	recorder *audit.Recorder
	build    config.BuildConfig
	runtime  config.RuntimeConfig
	lokiURL  string
	promURL  string
	env      config.Environment
	ghSecret string
}

// New constructs a workspace Service.
func New(
	repo *projectsinfra.Repository,
	sealer secrets.Sealer,
	producer queue.Producer,
	recorder *audit.Recorder,
	cfg config.Config,
) *Service {
	return &Service{
		repo:     repo,
		sealer:   sealer,
		producer: producer,
		recorder: recorder,
		build:    cfg.Build,
		runtime:  cfg.Runtime,
		lokiURL:  cfg.Observability.LokiURL,
		promURL:  cfg.Observability.PrometheusURL,
		env:      cfg.Env,
		ghSecret: cfg.GitHub.WebhookSecret,
	}
}

func actorFromPrincipal(p auth.Principal) uuid.UUID {
	id, err := uuid.Parse(strings.TrimSpace(p.UserID))
	if err != nil {
		return uuid.Nil
	}
	return id
}

func (s *Service) authorizeOrg(ctx context.Context, actor auth.Principal, orgID uuid.UUID, min auth.Role) error {
	userID := actorFromPrincipal(actor)
	if userID == uuid.Nil {
		return platformerrors.Unauthorized("invalid principal")
	}
	mr, err := s.repo.MemberRole(ctx, userID, orgID)
	if err != nil {
		return err
	}
	if !mr.AtLeast(min) {
		return platformerrors.Forbidden("insufficient privileges")
	}
	return nil
}

func slugOK(slug string) bool {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if len(slug) < 2 || len(slug) > 63 {
		return false
	}
	for _, c := range slug {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			continue
		}
		return false
	}
	return slug[0] != '-' && slug[len(slug)-1] != '-'
}

func sanitizeImagePart(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "_", "-")
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' {
			return r
		}
		return '-'
	}, s)
}

func dockerfileFromBuildCfg(raw json.RawMessage) string {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if v, ok := m["dockerfile_path"].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return "Dockerfile"
}

func listenPortFromRuntime(raw json.RawMessage) int {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if v, ok := m["listen_port"].(float64); ok && int(v) > 0 {
		return int(v)
	}
	if v, ok := m["port"].(float64); ok && int(v) > 0 {
		return int(v)
	}
	return 8080
}

func replicasFromRuntime(raw json.RawMessage) int {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if v, ok := m["replicas"].(float64); ok && int(v) > 0 {
		return int(v)
	}
	return 1
}

func noopGitSHA(sha string) bool {
	s := strings.TrimSpace(strings.ToLower(sha))
	if s == "" {
		return true
	}
	for _, c := range s {
		if c != '0' {
			return false
		}
	}
	return true
}

// normalizeRegistryHost keeps host:port valid for docker image references (do not mangle ':').
func normalizeRegistryHost(reg string) string {
	reg = strings.TrimSuffix(strings.TrimSpace(strings.ToLower(reg)), "/")
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' || r == ':' {
			return r
		}
		return '-'
	}, reg)
}

func composeImageRef(reg string, orgSlug, projSlug, svcSlug string, deploymentID uuid.UUID) string {
	reg = normalizeRegistryHost(reg)
	tag := strings.ToLower(deploymentID.String()[:12])
	return fmt.Sprintf("%s/%s/%s/%s:%s",
		reg, sanitizeImagePart(orgSlug),
		sanitizeImagePart(projSlug), sanitizeImagePart(svcSlug), tag)
}

// ListOrganizations returns orgs visible to caller.
func (s *Service) ListOrganizations(ctx context.Context, actor auth.Principal) ([]projectsinfra.OrganizationRow, error) {
	userID := actorFromPrincipal(actor)
	if userID == uuid.Nil {
		return nil, platformerrors.Unauthorized("invalid principal")
	}
	return s.repo.ListOrganizations(ctx, userID)
}

// CreateOrganization creates org + membership.
func (s *Service) CreateOrganization(ctx context.Context, actor auth.Principal, slug, name string) (projectsinfra.OrganizationRow, error) {
	userID := actorFromPrincipal(actor)
	if userID == uuid.Nil {
		return projectsinfra.OrganizationRow{}, platformerrors.Unauthorized("invalid principal")
	}
	name = strings.TrimSpace(name)
	if !slugOK(slug) || name == "" {
		return projectsinfra.OrganizationRow{}, platformerrors.Validation("invalid slug or name")
	}
	row, err := s.repo.CreateOrganization(ctx, userID, strings.ToLower(slug), name)
	if err != nil {
		return projectsinfra.OrganizationRow{}, err
	}
	u := userID
	s.recorder.Record(ctx, "organization.created", &u, map[string]any{"org_id": row.ID.String(), "slug": row.Slug})
	return row, nil
}

// ListProjects for org with ACL.
func (s *Service) ListProjects(ctx context.Context, actor auth.Principal, orgID uuid.UUID) ([]projectsinfra.ProjectRow, error) {
	if err := s.authorizeOrg(ctx, actor, orgID, auth.RoleViewer); err != nil {
		return nil, err
	}
	return s.repo.ListProjects(ctx, orgID)
}

// CreateProject with ACL.
func (s *Service) CreateProject(ctx context.Context, actor auth.Principal, orgID uuid.UUID, slug, name string, description, repoURL *string, branch string) (projectsinfra.ProjectRow, error) {
	if err := s.authorizeOrg(ctx, actor, orgID, auth.RoleDeveloper); err != nil {
		return projectsinfra.ProjectRow{}, err
	}
	if !slugOK(slug) || strings.TrimSpace(name) == "" {
		return projectsinfra.ProjectRow{}, platformerrors.Validation("invalid slug or name")
	}
	return s.repo.CreateProject(ctx, orgID, strings.ToLower(slug), name, description, repoURL, branch)
}

// GetProject resolves an ACL-checked project row.
func (s *Service) GetProject(ctx context.Context, actor auth.Principal, projectID uuid.UUID) (projectsinfra.ProjectRow, error) {
	p, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return projectsinfra.ProjectRow{}, err
	}
	if err := s.authorizeOrg(ctx, actor, p.OrganizationID, auth.RoleViewer); err != nil {
		return projectsinfra.ProjectRow{}, err
	}
	return p, nil
}

// UpdateProject patches name / description / repo URL / branch.
func (s *Service) UpdateProject(ctx context.Context, actor auth.Principal, projectID uuid.UUID, name, description, repoURL, branch *string) (projectsinfra.ProjectRow, error) {
	p, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return projectsinfra.ProjectRow{}, err
	}
	if err := s.authorizeOrg(ctx, actor, p.OrganizationID, auth.RoleDeveloper); err != nil {
		return projectsinfra.ProjectRow{}, err
	}
	return s.repo.UpdateProject(ctx, projectID, name, description, repoURL, branch)
}

func (s *Service) authorizeService(ctx context.Context, actor auth.Principal, serviceID uuid.UUID, min auth.Role) (projectsinfra.ServiceRow, error) {
	svc, err := s.repo.GetService(ctx, serviceID)
	if err != nil {
		return projectsinfra.ServiceRow{}, err
	}
	project, err := s.repo.GetProject(ctx, svc.ProjectID)
	if err != nil {
		return projectsinfra.ServiceRow{}, err
	}
	if err := s.authorizeOrg(ctx, actor, project.OrganizationID, min); err != nil {
		return projectsinfra.ServiceRow{}, err
	}
	return svc, nil
}

// ListServices ACL.
func (s *Service) ListServices(ctx context.Context, actor auth.Principal, projectID uuid.UUID) ([]projectsinfra.ServiceRow, error) {
	p, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := s.authorizeOrg(ctx, actor, p.OrganizationID, auth.RoleViewer); err != nil {
		return nil, err
	}
	return s.repo.ListServices(ctx, projectID)
}

// CreateService ACL.
func (s *Service) CreateService(ctx context.Context, actor auth.Principal, projectID uuid.UUID, slug, name, typ string) (projectsinfra.ServiceRow, error) {
	p, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return projectsinfra.ServiceRow{}, err
	}
	if err := s.authorizeOrg(ctx, actor, p.OrganizationID, auth.RoleDeveloper); err != nil {
		return projectsinfra.ServiceRow{}, err
	}
	if !slugOK(slug) || strings.TrimSpace(name) == "" {
		return projectsinfra.ServiceRow{}, platformerrors.Validation("invalid slug or name")
	}
	if typ == "" {
		typ = "web"
	}
	return s.repo.CreateService(ctx, projectID, strings.ToLower(slug), name, typ)
}

// GetService ACL.
func (s *Service) GetService(ctx context.Context, actor auth.Principal, serviceID uuid.UUID) (projectsinfra.ServiceRow, error) {
	return s.authorizeService(ctx, actor, serviceID, auth.RoleViewer)
}

// AuthorizeServiceDeveloper ensures developer+ access (terminal, domains).
func (s *Service) AuthorizeServiceDeveloper(ctx context.Context, actor auth.Principal, serviceID uuid.UUID) error {
	_, err := s.authorizeService(ctx, actor, serviceID, auth.RoleDeveloper)
	return err
}

// UpdateService merges JSON patches into build/runtime configs.
func (s *Service) UpdateService(ctx context.Context, actor auth.Principal, serviceID uuid.UUID, name *string, buildPatch, rtPatch json.RawMessage) (projectsinfra.ServiceRow, error) {
	if _, err := s.authorizeService(ctx, actor, serviceID, auth.RoleDeveloper); err != nil {
		return projectsinfra.ServiceRow{}, err
	}
	if len(rtPatch) > 0 && string(rtPatch) != "null" {
		var pm map[string]any
		if json.Unmarshal(rtPatch, &pm) == nil {
			if _, ok := pm["listen_port"]; ok {
				pm["listen_port_auto"] = false
				if b, err := json.Marshal(pm); err == nil {
					rtPatch = b
				}
			}
		}
	}
	return s.repo.UpdateService(ctx, serviceID, name, buildPatch, rtPatch)
}

// ListEnv returns ciphertext rows — handlers project previews only.
func (s *Service) ListEnv(ctx context.Context, actor auth.Principal, serviceID uuid.UUID) ([]projectsinfra.EnvVarRow, error) {
	if _, err := s.authorizeService(ctx, actor, serviceID, auth.RoleViewer); err != nil {
		return nil, err
	}
	return s.repo.ListEnvVars(ctx, serviceID)
}

// UpsertEnvVar encrypts plaintext.
func (s *Service) UpsertEnvVar(ctx context.Context, actor auth.Principal, serviceID uuid.UUID, key, value string, secret bool) (projectsinfra.EnvVarRow, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return projectsinfra.EnvVarRow{}, platformerrors.Validation("env key required")
	}
	if _, err := s.authorizeService(ctx, actor, serviceID, auth.RoleDeveloper); err != nil {
		return projectsinfra.EnvVarRow{}, err
	}
	enc, err := s.sealer.Seal([]byte(value))
	if err != nil {
		return projectsinfra.EnvVarRow{}, platformerrors.Internal("seal secret").WithCause(err)
	}
	return s.repo.UpsertEnvVar(ctx, serviceID, key, enc, secret)
}

// DeleteEnvVar removes a DB row.
func (s *Service) DeleteEnvVar(ctx context.Context, actor auth.Principal, serviceID uuid.UUID, key string) error {
	if _, err := s.authorizeService(ctx, actor, serviceID, auth.RoleDeveloper); err != nil {
		return err
	}
	return s.repo.DeleteEnvVar(ctx, serviceID, key)
}

// ListDeployments with ACL filtering.
func (s *Service) ListDeployments(ctx context.Context, actor auth.Principal, serviceID *uuid.UUID, projectID *uuid.UUID, limit int) ([]projectsinfra.DeploymentJoined, error) {
	if serviceID != nil {
		svc, err := s.repo.GetService(ctx, *serviceID)
		if err != nil {
			return nil, err
		}
		project, err := s.repo.GetProject(ctx, svc.ProjectID)
		if err != nil {
			return nil, err
		}
		if err := s.authorizeOrg(ctx, actor, project.OrganizationID, auth.RoleViewer); err != nil {
			return nil, err
		}
		return s.repo.ListDeploymentsJoined(ctx, serviceID, nil, limit)
	}
	if projectID != nil {
		p, err := s.repo.GetProject(ctx, *projectID)
		if err != nil {
			return nil, err
		}
		if err := s.authorizeOrg(ctx, actor, p.OrganizationID, auth.RoleViewer); err != nil {
			return nil, err
		}
		return s.repo.ListDeploymentsJoined(ctx, nil, projectID, limit)
	}
	return nil, platformerrors.Validation("project_id or service_id is required")
}

// GetDeployment enforces ACL through joined org.
func (s *Service) GetDeployment(ctx context.Context, actor auth.Principal, deploymentID uuid.UUID) (projectsinfra.DeploymentJoined, error) {
	dj, err := s.repo.GetDeploymentJoined(ctx, deploymentID)
	if err != nil {
		return projectsinfra.DeploymentJoined{}, err
	}
	svc, err := s.repo.GetService(ctx, dj.ServiceID)
	if err != nil {
		return projectsinfra.DeploymentJoined{}, err
	}
	project, err := s.repo.GetProject(ctx, svc.ProjectID)
	if err != nil {
		return projectsinfra.DeploymentJoined{}, err
	}
	if err := s.authorizeOrg(ctx, actor, project.OrganizationID, auth.RoleViewer); err != nil {
		return projectsinfra.DeploymentJoined{}, err
	}
	return dj, nil
}

// enqueueBuild persists rows and publishes an asynq build job.
func (s *Service) enqueueBuild(ctx context.Context, svc projectsinfra.ServiceRow,
	project projectsinfra.ProjectRow, org projectsinfra.OrganizationRow,
	trigger string, triggeredBy *uuid.UUID, sha, msg, ref *string,
) (uuid.UUID, uuid.UUID, error) {
	if project.RepoURL == nil || strings.TrimSpace(*project.RepoURL) == "" {
		return uuid.Nil, uuid.Nil, platformerrors.Validation("project.repo_url must be set before deploying")
	}
	deploymentID, err := s.repo.InsertDeployment(ctx, svc.ID, triggeredBy, trigger)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if sha != nil || msg != nil || ref != nil {
		if err := s.repo.PatchDeploymentGitMeta(ctx, deploymentID, sha, msg, ref); err != nil {
			return uuid.Nil, uuid.Nil, err
		}
	}
	buildID, err := s.repo.InsertBuild(ctx, deploymentID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if err := s.repo.UpdateDeploymentWorker(ctx, deploymentID, "building", nil, nil); err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if err := s.repo.PatchServiceRuntimeImage(ctx, svc.ID, nil, "building"); err != nil {
		logger.FromContext(ctx).Warn("enqueueBuild.service_status", "error", err)
	}
	imageRef := composeImageRef(s.build.RegistryURL, org.Slug, project.Slug, svc.Slug, deploymentID)
	payload := jobs.BuildRunPayload{
		BuildID:          buildID.String(),
		DeploymentID:     deploymentID.String(),
		ServiceID:        svc.ID.String(),
		OrganizationID:   project.OrganizationID.String(),
		ProjectID:        project.ID.String(),
		RepoURL:          strings.TrimSpace(*project.RepoURL),
		Ref:              strings.TrimSpace(project.DefaultBranch),
		RegistryURL:      s.build.RegistryURL,
		RegistryInsecure: s.build.RegistryInsecure,
		ImageRef:         imageRef,
		DockerfilePath:   dockerfileFromBuildCfg(svc.BuildConfig),
	}
	if _, err = s.producer.Enqueue(ctx, queue.Job{
		Type:       queue.JobTypeBuildRun,
		Payload:    queue.MustMarshalJSON(&payload),
		MaxRetries: 2,
		Timeout:    queue.DefaultBuildJobTimeout(),
		Queue:      queue.QueueBuild,
	}); err != nil {
		return uuid.Nil, uuid.Nil, platformerrors.Internal("enqueue build").WithCause(err)
	}
	return deploymentID, buildID, nil
}

// StartDeployment creates a queued deployment initiated by dashboard user.
func (s *Service) StartDeployment(ctx context.Context, actor auth.Principal, serviceID uuid.UUID) (deploymentID uuid.UUID, err error) {
	svc, err := s.authorizeService(ctx, actor, serviceID, auth.RoleDeveloper)
	if err != nil {
		return uuid.Nil, err
	}
	project, err := s.repo.GetProject(ctx, svc.ProjectID)
	if err != nil {
		return uuid.Nil, err
	}
	org, err := s.repo.GetOrganization(ctx, project.OrganizationID)
	if err != nil {
		return uuid.Nil, err
	}
	uid := actorFromPrincipal(actor)
	depID, _, err := s.enqueueBuild(ctx, svc, project, org, "manual", &uid, nil, nil, nil)
	return depID, err
}

// StartDeploymentWebhook is invoked after GitHub webhook verification.
func (s *Service) StartDeploymentWebhook(ctx context.Context, serviceID uuid.UUID, sha, msg, ref *string) (uuid.UUID, error) {
	if sha == nil || noopGitSHA(*sha) {
		return uuid.Nil, nil
	}
	sTrim := strings.TrimSpace(*sha)
	dup, err := s.repo.HasRecentWebhookDeployment(ctx, serviceID, sTrim, time.Now().UTC().Add(-60*time.Second))
	if err != nil {
		return uuid.Nil, err
	}
	if dup {
		return uuid.Nil, nil
	}
	svc, err := s.repo.GetService(ctx, serviceID)
	if err != nil {
		return uuid.Nil, err
	}
	project, err := s.repo.GetProject(ctx, svc.ProjectID)
	if err != nil {
		return uuid.Nil, err
	}
	org, err := s.repo.GetOrganization(ctx, project.OrganizationID)
	if err != nil {
		return uuid.Nil, err
	}
	shaNorm := sTrim
	depID, _, err := s.enqueueBuild(ctx, svc, project, org, "webhook", nil, &shaNorm, msg, ref)
	return depID, err
}

// DispatchGitPush resolves projects by repo URL and enqueues webhook deploys.
// installationID, when non-nil and non-zero, restricts matches to projects with that GitHub App installation.
func (s *Service) DispatchGitPush(ctx context.Context, normalizedRepo, sha, msg, gitRef string, installationID *int64) int {
	projs, err := s.repo.FindProjectsByRepoURL(ctx, normalizedRepo, gitRef, installationID)
	if err != nil {
		logger.FromContext(ctx).Warn("dispatch.push.list_failed", "error", err)
		return 0
	}
	var n int
	for _, proj := range projs {
		srvs, err := s.repo.ListServices(ctx, proj.ID)
		if err != nil {
			logger.FromContext(ctx).Warn("dispatch.push.services_failed", "error", err)
			continue
		}
		refCopy := gitRef
		shaCopy := sha
		msgCopy := msg
		for _, svc := range srvs {
			depID, err := s.StartDeploymentWebhook(ctx, svc.ID, &shaCopy, &msgCopy, &refCopy)
			if err != nil {
				logger.FromContext(ctx).Warn("dispatch.push.enqueue_failed", "service_id", svc.ID.String(), "error", err)
				continue
			}
			if depID != uuid.Nil {
				n++
			}
		}
	}
	return n
}

// SetProjectGitHubInstallation persists the GitHub App installation id (used after verified OAuth / App flow).
func (s *Service) SetProjectGitHubInstallation(ctx context.Context, actor auth.Principal, projectID uuid.UUID, installationID int64) (projectsinfra.ProjectRow, error) {
	if installationID <= 0 {
		return projectsinfra.ProjectRow{}, platformerrors.Validation("installation_id must be positive")
	}
	p, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return projectsinfra.ProjectRow{}, err
	}
	if err := s.authorizeOrg(ctx, actor, p.OrganizationID, auth.RoleDeveloper); err != nil {
		return projectsinfra.ProjectRow{}, err
	}
	return s.repo.SetProjectGitHubInstallation(ctx, projectID, installationID)
}

// WebhookSecret for GitHub HMAC verification.
func (s *Service) WebhookSecret() string { return s.ghSecret }

func (s *Service) Env() config.Environment { return s.env }

// QueryLogs proxies Loki.
func (s *Service) QueryLogs(ctx context.Context, actor auth.Principal, serviceID uuid.UUID, window time.Duration, limit int) ([]telemetry.LogLine, error) {
	if _, err := s.authorizeService(ctx, actor, serviceID, auth.RoleViewer); err != nil {
		return nil, err
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	return telemetry.QueryLokiServiceLogs(ctx, s.lokiURL, serviceID.String(), window, limit)
}

// QueryMetrics returns per-service container CPU/memory from cAdvisor/Prometheus.
func (s *Service) QueryMetrics(ctx context.Context, actor auth.Principal, serviceID uuid.UUID, window time.Duration) ([]telemetry.MetricSeries, error) {
	if _, err := s.authorizeService(ctx, actor, serviceID, auth.RoleViewer); err != nil {
		return nil, err
	}
	if window <= 0 {
		window = time.Hour
	}
	sid := serviceID.String()
	cpuExpr := `rate(container_cpu_usage_seconds_total{container_label_nebula_service="` + sid + `"}[5m])`
	memExpr := `container_memory_usage_bytes{container_label_nebula_service="` + sid + `"}`
	empty := []telemetry.MetricSeries{
		{Name: "cpu_usage", Unit: "cores", Points: []telemetry.MetricPoint{}},
		{Name: "memory_bytes", Unit: "bytes", Points: []telemetry.MetricPoint{}},
	}
	cpuPts, _, err := telemetry.QueryPrometheusRange(ctx, s.promURL, cpuExpr, window, time.Minute)
	if err != nil {
		return empty, nil
	}
	memPts, _, err := telemetry.QueryPrometheusRange(ctx, s.promURL, memExpr, window, time.Minute)
	if err != nil {
		memPts = nil
	}
	if len(cpuPts) == 0 && len(memPts) == 0 {
		return empty, nil
	}
	return []telemetry.MetricSeries{
		{Name: "cpu_usage", Unit: "cores", Points: cpuPts},
		{Name: "memory_bytes", Unit: "bytes", Points: memPts},
	}, nil
}

// CountServices validates viewer access before counting services.
func (s *Service) CountServices(ctx context.Context, actor auth.Principal, projectID uuid.UUID) (int, error) {
	p, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return 0, err
	}
	if err := s.authorizeOrg(ctx, actor, p.OrganizationID, auth.RoleViewer); err != nil {
		return 0, err
	}
	return s.repo.CountServicesForProject(ctx, projectID)
}

// ListenPort parses listen port from service runtime JSON.
func ListenPort(rt json.RawMessage) int { return listenPortFromRuntime(rt) }

// Replicas parses replica count from service runtime JSON.
func Replicas(rt json.RawMessage) int { return replicasFromRuntime(rt) }
