package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nebulacloud/nebula/internal/platform/auth"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
)

// Repository is Postgres persistence for orgs, projects, services, env, deployments.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Repository.
func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// --- membership ---

// MemberRole returns the user's role in an org or not_found.
func (r *Repository) MemberRole(ctx context.Context, userID, orgID uuid.UUID) (auth.Role, error) {
	const q = `SELECT role FROM memberships WHERE user_id = $1 AND organization_id = $2`
	var roleStr string
	err := r.pool.QueryRow(ctx, q, userID, orgID).Scan(&roleStr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", platformerrors.NotFound("membership not found")
		}
		return "", platformerrors.Internal("membership lookup").WithCause(err)
	}
	return auth.Role(roleStr), nil
}

// --- organizations ---

type OrganizationRow struct {
	ID        uuid.UUID
	Slug      string
	Name      string
	Plan      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListOrganizations returns orgs visible to user.
func (r *Repository) ListOrganizations(ctx context.Context, userID uuid.UUID) ([]OrganizationRow, error) {
	const q = `
		SELECT o.id, o.slug, o.name, o.plan, o.created_at, o.updated_at
		FROM organizations o
		INNER JOIN memberships m ON m.organization_id = o.id
		WHERE m.user_id = $1
		ORDER BY o.name
	`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, platformerrors.Internal("list organizations").WithCause(err)
	}
	defer rows.Close()
	var out []OrganizationRow
	for rows.Next() {
		var o OrganizationRow
		if err := rows.Scan(&o.ID, &o.Slug, &o.Name, &o.Plan, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, platformerrors.Internal("scan organization").WithCause(err)
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// CreateOrganization inserts org + admin membership transactionally.
func (r *Repository) CreateOrganization(ctx context.Context, ownerID uuid.UUID, slug, name string) (OrganizationRow, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return OrganizationRow{}, platformerrors.Internal("begin tx").WithCause(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	orgID := uuid.New()
	const insOrg = `
		INSERT INTO organizations (id, slug, name, owner_id, plan, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'free', NOW(), NOW())
		RETURNING id, slug, name, plan, created_at, updated_at`
	var row OrganizationRow
	if err := tx.QueryRow(ctx, insOrg, orgID, slug, name, ownerID).Scan(
		&row.ID, &row.Slug, &row.Name, &row.Plan, &row.CreatedAt, &row.UpdatedAt,
	); err != nil {
		if isUniqueViolation(err) {
			return OrganizationRow{}, platformerrors.Conflict("organization slug already taken")
		}
		return OrganizationRow{}, platformerrors.Internal("create organization").WithCause(err)
	}
	const insMem = `
		INSERT INTO memberships (organization_id, user_id, role, created_at)
		VALUES ($1, $2, 'admin', NOW())`
	if _, err := tx.Exec(ctx, insMem, orgID, ownerID); err != nil {
		return OrganizationRow{}, platformerrors.Internal("create membership").WithCause(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return OrganizationRow{}, platformerrors.Internal("commit organization").WithCause(err)
	}
	return row, nil
}

// GetOrganization fetches org by id.
func (r *Repository) GetOrganization(ctx context.Context, id uuid.UUID) (OrganizationRow, error) {
	const q = `
		SELECT id, slug, name, plan, created_at, updated_at
		FROM organizations WHERE id = $1`
	var row OrganizationRow
	if err := r.pool.QueryRow(ctx, q, id).Scan(
		&row.ID, &row.Slug, &row.Name, &row.Plan, &row.CreatedAt, &row.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrganizationRow{}, platformerrors.NotFound("organization not found")
		}
		return OrganizationRow{}, platformerrors.Internal("get organization").WithCause(err)
	}
	return row, nil
}

// --- projects ---

type ProjectRow struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	Slug            string
	Name            string
	Description     *string
	RepoURL         *string
	DefaultBranch   string
	GitHubInstallID *int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ListProjects by org id.
func (r *Repository) ListProjects(ctx context.Context, orgID uuid.UUID) ([]ProjectRow, error) {
	const q = `
		SELECT p.id, p.organization_id, p.slug, p.name, p.description,
		       p.repo_url, p.default_branch, p.github_installation_id, p.created_at, p.updated_at
		FROM projects p
		WHERE p.organization_id = $1
		ORDER BY p.name`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, platformerrors.Internal("list projects").WithCause(err)
	}
	defer rows.Close()
	var out []ProjectRow
	for rows.Next() {
		var p ProjectRow
		if err := rows.Scan(
			&p.ID, &p.OrganizationID, &p.Slug, &p.Name, &p.Description,
			&p.RepoURL, &p.DefaultBranch, &p.GitHubInstallID, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("scan project").WithCause(err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CountServicesForProject returns count of services.
func (r *Repository) CountServicesForProject(ctx context.Context, projectID uuid.UUID) (int, error) {
	const q = `SELECT COUNT(*) FROM services WHERE project_id = $1`
	var n int
	if err := r.pool.QueryRow(ctx, q, projectID).Scan(&n); err != nil {
		return 0, platformerrors.Internal("count services").WithCause(err)
	}
	return n, nil
}

// GetProject fetches project and ensures belongs to org.
func (r *Repository) GetProject(ctx context.Context, projectID uuid.UUID) (ProjectRow, error) {
	const q = `
		SELECT p.id, p.organization_id, p.slug, p.name, p.description,
		       p.repo_url, p.default_branch, p.github_installation_id, p.created_at, p.updated_at
		FROM projects p WHERE p.id = $1`
	var p ProjectRow
	if err := r.pool.QueryRow(ctx, q, projectID).Scan(
		&p.ID, &p.OrganizationID, &p.Slug, &p.Name, &p.Description,
		&p.RepoURL, &p.DefaultBranch, &p.GitHubInstallID, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProjectRow{}, platformerrors.NotFound("project not found")
		}
		return ProjectRow{}, platformerrors.Internal("get project").WithCause(err)
	}
	return p, nil
}

// CreateProject inserts a project row.
func (r *Repository) CreateProject(ctx context.Context, orgID uuid.UUID, slug, name string, description *string, repoURL *string, branch string) (ProjectRow, error) {
	if branch == "" {
		branch = "main"
	}
	id := uuid.New()
	const q = `
		INSERT INTO projects (id, organization_id, slug, name, description, repo_url, default_branch, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, organization_id, slug, name, description,
		          repo_url, default_branch, github_installation_id, created_at, updated_at`
	var p ProjectRow
	if err := r.pool.QueryRow(ctx, q, id, orgID, slug, name, description, repoURL, branch).Scan(
		&p.ID, &p.OrganizationID, &p.Slug, &p.Name, &p.Description,
		&p.RepoURL, &p.DefaultBranch, &p.GitHubInstallID, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		if isUniqueViolation(err) {
			return ProjectRow{}, platformerrors.Conflict("project slug already taken in this organization")
		}
		return ProjectRow{}, platformerrors.Internal("create project").WithCause(err)
	}
	return p, nil
}

// UpdateProject patches name, description, repo URL, branch.
func (r *Repository) UpdateProject(ctx context.Context, projectID uuid.UUID, name, description, repoURL, branch *string) (ProjectRow, error) {
	p, err := r.GetProject(ctx, projectID)
	if err != nil {
		return ProjectRow{}, err
	}
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" {
			return ProjectRow{}, platformerrors.Validation("name cannot be empty")
		}
		p.Name = n
	}
	if description != nil {
		p.Description = description
	}
	if repoURL != nil {
		u := strings.TrimSpace(*repoURL)
		if u == "" {
			p.RepoURL = nil
		} else {
			p.RepoURL = &u
		}
	}
	if branch != nil && *branch != "" {
		p.DefaultBranch = *branch
	}
	const q = `
		UPDATE projects SET name = $2, description = $3, repo_url = $4, default_branch = $5, updated_at = NOW()
		WHERE id = $1
		RETURNING id, organization_id, slug, name, description,
		          repo_url, default_branch, github_installation_id, created_at, updated_at`
	if err := r.pool.QueryRow(ctx, q, projectID, p.Name, p.Description, p.RepoURL, p.DefaultBranch).Scan(
		&p.ID, &p.OrganizationID, &p.Slug, &p.Name, &p.Description,
		&p.RepoURL, &p.DefaultBranch, &p.GitHubInstallID, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return ProjectRow{}, platformerrors.Internal("update project").WithCause(err)
	}
	return p, nil
}

// SetProjectGitHubInstallation sets github_installation_id for a project row.
func (r *Repository) SetProjectGitHubInstallation(ctx context.Context, projectID uuid.UUID, installationID int64) (ProjectRow, error) {
	const q = `
		UPDATE projects SET github_installation_id = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, organization_id, slug, name, description,
		          repo_url, default_branch, github_installation_id, created_at, updated_at`
	var p ProjectRow
	if err := r.pool.QueryRow(ctx, q, projectID, installationID).Scan(
		&p.ID, &p.OrganizationID, &p.Slug, &p.Name, &p.Description,
		&p.RepoURL, &p.DefaultBranch, &p.GitHubInstallID, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProjectRow{}, platformerrors.NotFound("project not found")
		}
		return ProjectRow{}, platformerrors.Internal("set github installation").WithCause(err)
	}
	return p, nil
}

// FindProjectsByRepoURL returns matching projects across all orgs (webhook matcher).
// When matchInstallationID is non-nil and > 0, only projects with the same github_installation_id match.
func (r *Repository) FindProjectsByRepoURL(ctx context.Context, normalizedURL string, branchRef string, matchInstallationID *int64) ([]ProjectRow, error) {
	const q = `
		SELECT p.id, p.organization_id, p.slug, p.name, p.description,
		       p.repo_url, p.default_branch, p.github_installation_id, p.created_at, p.updated_at
		FROM projects p
		WHERE p.repo_url IS NOT NULL`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, platformerrors.Internal("scan projects repo").WithCause(err)
	}
	defer rows.Close()
	var out []ProjectRow
	br := strings.TrimPrefix(branchRef, "refs/heads/")
	for rows.Next() {
		var p ProjectRow
		if err := rows.Scan(
			&p.ID, &p.OrganizationID, &p.Slug, &p.Name, &p.Description,
			&p.RepoURL, &p.DefaultBranch, &p.GitHubInstallID, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("scan project row").WithCause(err)
		}
		if p.RepoURL == nil {
			continue
		}
		if NormalizeRepoURL(*p.RepoURL) != normalizedURL {
			continue
		}
		if br != "" && p.DefaultBranch != "" && br != p.DefaultBranch {
			continue
		}
		if matchInstallationID != nil && *matchInstallationID != 0 {
			if p.GitHubInstallID == nil || *p.GitHubInstallID != *matchInstallationID {
				continue
			}
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// --- services ---

type ServiceRow struct {
	ID            uuid.UUID
	ProjectID     uuid.UUID
	Slug          string
	Name          string
	Type          string
	Status        string
	BuildConfig   json.RawMessage
	RuntimeConfig json.RawMessage
	CurrentImage  *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ListServices ...
func (r *Repository) ListServices(ctx context.Context, projectID uuid.UUID) ([]ServiceRow, error) {
	const q = `
		SELECT id, project_id, slug, name, type, status,
		       COALESCE(build_config::text, '{}')::jsonb,
		       COALESCE(runtime_config::text, '{}')::jsonb,
		       current_image, created_at, updated_at
		FROM services WHERE project_id = $1 ORDER BY name`
	rows, err := r.pool.Query(ctx, q, projectID)
	if err != nil {
		return nil, platformerrors.Internal("list services").WithCause(err)
	}
	defer rows.Close()
	return scanServiceRows(rows)
}

func scanServiceRows(rows pgx.Rows) ([]ServiceRow, error) {
	var out []ServiceRow
	for rows.Next() {
		var s ServiceRow
		var buildB, rtB []byte
		if err := rows.Scan(
			&s.ID, &s.ProjectID, &s.Slug, &s.Name, &s.Type, &s.Status,
			&buildB, &rtB, &s.CurrentImage, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("scan service").WithCause(err)
		}
		s.BuildConfig = json.RawMessage(buildB)
		s.RuntimeConfig = json.RawMessage(rtB)
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetService ...
func (r *Repository) GetService(ctx context.Context, id uuid.UUID) (ServiceRow, error) {
	const q = `
		SELECT id, project_id, slug, name, type, status,
		       COALESCE(build_config::text, '{}')::jsonb,
		       COALESCE(runtime_config::text, '{}')::jsonb,
		       current_image, created_at, updated_at
		FROM services WHERE id = $1`
	var s ServiceRow
	var buildB, rtB []byte
	if err := r.pool.QueryRow(ctx, q, id).Scan(
		&s.ID, &s.ProjectID, &s.Slug, &s.Name, &s.Type, &s.Status,
		&buildB, &rtB, &s.CurrentImage, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ServiceRow{}, platformerrors.NotFound("service not found")
		}
		return ServiceRow{}, platformerrors.Internal("get service").WithCause(err)
	}
	s.BuildConfig = json.RawMessage(buildB)
	s.RuntimeConfig = json.RawMessage(rtB)
	return s, nil
}

// CreateService ...
func (r *Repository) CreateService(ctx context.Context, projectID uuid.UUID, slug, name, typ string) (ServiceRow, error) {
	if typ == "" {
		typ = "web"
	}
	id := uuid.New()
	buildJSON := json.RawMessage(`{}`)
	runtimeJSON := json.RawMessage(`{"listen_port":8080,"replicas":1}`)
	const q = `
		INSERT INTO services (id, project_id, slug, name, type, status, build_config, runtime_config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'idle', $6, $7, NOW(), NOW())
		RETURNING id, project_id, slug, name, type, status,
		          COALESCE(build_config::text,'{}')::jsonb,
		          COALESCE(runtime_config::text,'{}')::jsonb,
		          current_image, created_at, updated_at`
	var s ServiceRow
	var buildB, rtB []byte
	if err := r.pool.QueryRow(ctx, q, id, projectID, slug, name, typ, buildJSON, runtimeJSON).Scan(
		&s.ID, &s.ProjectID, &s.Slug, &s.Name, &s.Type, &s.Status,
		&buildB, &rtB, &s.CurrentImage, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		if isUniqueViolation(err) {
			return ServiceRow{}, platformerrors.Conflict("service slug already taken in this project")
		}
		return ServiceRow{}, platformerrors.Internal("create service").WithCause(err)
	}
	s.BuildConfig = json.RawMessage(buildB)
	s.RuntimeConfig = json.RawMessage(rtB)
	return s, nil
}

// UpdateService merges name and configs.
func (r *Repository) UpdateService(ctx context.Context, serviceID uuid.UUID, name *string, buildMerge, runtimeMerge json.RawMessage) (ServiceRow, error) {
	s, err := r.GetService(ctx, serviceID)
	if err != nil {
		return ServiceRow{}, err
	}
	if name != nil {
		s.Name = strings.TrimSpace(*name)
	}
	if len(buildMerge) > 0 && string(buildMerge) != "null" {
		s.BuildConfig, err = mergeJSON(s.BuildConfig, buildMerge)
		if err != nil {
			return ServiceRow{}, platformerrors.Validation("invalid build_config").WithCause(err)
		}
	}
	if len(runtimeMerge) > 0 && string(runtimeMerge) != "null" {
		s.RuntimeConfig, err = mergeJSON(s.RuntimeConfig, runtimeMerge)
		if err != nil {
			return ServiceRow{}, platformerrors.Validation("invalid runtime_config").WithCause(err)
		}
	}
	const q = `
		UPDATE services SET name = $2, build_config = $3, runtime_config = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING id, project_id, slug, name, type, status,
		          COALESCE(build_config::text,'{}')::jsonb,
		          COALESCE(runtime_config::text,'{}')::jsonb,
		          current_image, created_at, updated_at`
	var buildB, rtB []byte
	if err := r.pool.QueryRow(ctx, q, serviceID, s.Name, s.BuildConfig, s.RuntimeConfig).Scan(
		&s.ID, &s.ProjectID, &s.Slug, &s.Name, &s.Type, &s.Status,
		&buildB, &rtB, &s.CurrentImage, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return ServiceRow{}, platformerrors.Internal("update service").WithCause(err)
	}
	s.BuildConfig = json.RawMessage(buildB)
	s.RuntimeConfig = json.RawMessage(rtB)
	return s, nil
}

func mergeJSON(base json.RawMessage, patch json.RawMessage) (json.RawMessage, error) {
	var bm, pm map[string]any
	if err := json.Unmarshal(base, &bm); err != nil {
		bm = map[string]any{}
	}
	if err := json.Unmarshal(patch, &pm); err != nil {
		return nil, err
	}
	for k, v := range pm {
		bm[k] = v
	}
	b, err := json.Marshal(bm)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// PatchServiceRuntimeImage sets current_image and optionally status for deploy completion.
func (r *Repository) PatchServiceRuntimeImage(ctx context.Context, serviceID uuid.UUID, image *string, status string) error {
	const q = `UPDATE services SET current_image = COALESCE($2, current_image), status = $3, updated_at = NOW() WHERE id = $1`
	if _, err := r.pool.Exec(ctx, q, serviceID, image, status); err != nil {
		return platformerrors.Internal("update service image").WithCause(err)
	}
	return nil
}

// --- env vars ---

type EnvVarRow struct {
	ID        uuid.UUID
	ServiceID uuid.UUID
	Key       string
	ValueEnc  []byte
	IsSecret  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListEnvVars ...
func (r *Repository) ListEnvVars(ctx context.Context, serviceID uuid.UUID) ([]EnvVarRow, error) {
	const q = `
		SELECT id, service_id, key, value_enc, is_secret, created_at, updated_at
		FROM env_vars WHERE service_id = $1 ORDER BY key`
	rows, err := r.pool.Query(ctx, q, serviceID)
	if err != nil {
		return nil, platformerrors.Internal("list env vars").WithCause(err)
	}
	defer rows.Close()
	var out []EnvVarRow
	for rows.Next() {
		var e EnvVarRow
		if err := rows.Scan(&e.ID, &e.ServiceID, &e.Key, &e.ValueEnc, &e.IsSecret, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, platformerrors.Internal("scan env var").WithCause(err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpsertEnvVar ...
func (r *Repository) UpsertEnvVar(ctx context.Context, serviceID uuid.UUID, key string, valueEnc []byte, secret bool) (EnvVarRow, error) {
	id := uuid.New()
	const q = `
		INSERT INTO env_vars (id, service_id, key, value_enc, is_secret, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (service_id, key) DO UPDATE SET
		  value_enc = EXCLUDED.value_enc,
		  is_secret = EXCLUDED.is_secret,
		  updated_at = NOW()
		RETURNING id, service_id, key, value_enc, is_secret, created_at, updated_at`
	var e EnvVarRow
	if err := r.pool.QueryRow(ctx, q, id, serviceID, key, valueEnc, secret).Scan(
		&e.ID, &e.ServiceID, &e.Key, &e.ValueEnc, &e.IsSecret, &e.CreatedAt, &e.UpdatedAt,
	); err != nil {
		return EnvVarRow{}, platformerrors.Internal("upsert env var").WithCause(err)
	}
	return e, nil
}

// DeleteEnvVar ...
func (r *Repository) DeleteEnvVar(ctx context.Context, serviceID uuid.UUID, key string) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM env_vars WHERE service_id = $1 AND key = $2`, serviceID, key)
	if err != nil {
		return platformerrors.Internal("delete env var").WithCause(err)
	}
	if cmd.RowsAffected() == 0 {
		return platformerrors.NotFound("env var not found")
	}
	return nil
}

// --- deployments / builds ---

type DeploymentRow struct {
	ID           uuid.UUID
	ServiceID    uuid.UUID
	TriggeredBy  *uuid.UUID
	Trigger      string
	CommitSHA    *string
	CommitMsg    *string
	Ref          *string
	ImageRef     *string
	Status       string
	ErrorMsg     *string
	Metadata     json.RawMessage
	CreatedAt    time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
}

func (r *Repository) InsertDeployment(ctx context.Context, svcID uuid.UUID, userID *uuid.UUID, trigger string) (uuid.UUID, error) {
	id := uuid.New()
	const q = `
		INSERT INTO deployments (id, service_id, triggered_by, trigger, status, created_at)
		VALUES ($1, $2, $3, $4, 'queued', NOW())`
	if _, err := r.pool.Exec(ctx, q, id, svcID, userID, trigger); err != nil {
		return uuid.Nil, platformerrors.Internal("insert deployment").WithCause(err)
	}
	return id, nil
}

// DeploymentServiceID returns owning service UUID.
func (r *Repository) DeploymentServiceID(ctx context.Context, depID uuid.UUID) (uuid.UUID, error) {
	var sid uuid.UUID
	if err := r.pool.QueryRow(ctx, `SELECT service_id FROM deployments WHERE id = $1`, depID).Scan(&sid); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, platformerrors.NotFound("deployment not found")
		}
		return uuid.Nil, platformerrors.Internal("deployment lookup").WithCause(err)
	}
	return sid, nil
}

// PatchDeploymentGitMeta sets git-related fields before the worker consumes the row.
func (r *Repository) PatchDeploymentGitMeta(ctx context.Context, depID uuid.UUID, sha, msg, ref *string) error {
	const q = `
		UPDATE deployments SET
		  commit_sha = COALESCE($2, commit_sha),
		  commit_message = COALESCE($3, commit_message),
		  ref = COALESCE($4, ref)
		WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, depID, sha, msg, ref)
	if err != nil {
		return platformerrors.Internal("patch deployment git meta").WithCause(err)
	}
	return nil
}

func (r *Repository) InsertBuild(ctx context.Context, deploymentID uuid.UUID) (uuid.UUID, error) {
	id := uuid.New()
	const q = `
		INSERT INTO builds (id, deployment_id, status, created_at)
		VALUES ($1, $2, 'queued', NOW())`
	if _, err := r.pool.Exec(ctx, q, id, deploymentID); err != nil {
		return uuid.Nil, platformerrors.Internal("insert build").WithCause(err)
	}
	return id, nil
}

func (r *Repository) UpdateDeploymentWorker(ctx context.Context, depID uuid.UUID, status string, imageRef, errMsg *string) error {
	terminal := status == "failed" || status == "canceled" || status == "running" || status == "rolled_back"
	const q = `
		UPDATE deployments SET
		  status = $2,
		  image_ref = COALESCE($3, image_ref),
		  error_message = COALESCE($4, error_message),
		  started_at = COALESCE(started_at,
		     CASE WHEN $2 IN ('building','pushing','deploying','running','failed','canceled') THEN NOW() END),
		  finished_at = CASE WHEN $5 THEN NOW() ELSE finished_at END
		WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, depID, status, imageRef, errMsg, terminal)
	if err != nil {
		return platformerrors.Internal("update deployment").WithCause(err)
	}
	return nil
}

func (r *Repository) UpdateBuildFields(ctx context.Context, buildID uuid.UUID, workerID string, status string,
	detectedStack, errMsg *string, exitCode *int,
) error {
	const q = `
		UPDATE builds SET
		  worker_id = COALESCE(NULLIF($2,''), worker_id),
		  status = $3,
		  detected_stack = COALESCE($4, detected_stack),
		  error_message = COALESCE($5, error_message),
		  exit_code = COALESCE($6, exit_code),
		  started_at = COALESCE(started_at, CASE WHEN $3 != 'queued' THEN NOW() ELSE started_at END),
		  finished_at = CASE WHEN $3 IN ('success','failed','canceled') THEN NOW() ELSE finished_at END
		WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, buildID, workerID, status, detectedStack, errMsg, exitCode)
	if err != nil {
		return platformerrors.Internal("update build").WithCause(err)
	}
	return nil
}

// ListDeploymentsJoined returns deployment rows enriched for dashboard.
func (r *Repository) ListDeploymentsJoined(ctx context.Context, serviceID *uuid.UUID, projectID *uuid.UUID, limit int) ([]DeploymentJoined, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	where := []string{"1=1"}
	args := []any{}
	n := 1
	if serviceID != nil {
		where = append(where, fmt.Sprintf("d.service_id = $%d", n))
		args = append(args, *serviceID)
		n++
	}
	if projectID != nil {
		where = append(where, fmt.Sprintf("p.id = $%d", n))
		args = append(args, *projectID)
		n++
	}
	args = append(args, limit)
	limitParam := n

	q := fmt.Sprintf(`
		SELECT d.id, d.service_id, s.name AS service_name, p.id AS project_id, p.name AS project_name,
		       d.triggered_by, u.email AS trigger_email, d.trigger, d.status,
		       d.commit_sha, d.commit_message, d.ref, d.image_ref,
		       COALESCE((EXTRACT(EPOCH FROM (d.finished_at - d.started_at)) * 1000)::bigint, 0) AS duration_ms,
		       d.created_at, d.started_at, d.finished_at
		FROM deployments d
		INNER JOIN services s ON s.id = d.service_id
		INNER JOIN projects p ON p.id = s.project_id
		LEFT JOIN users u ON u.id = d.triggered_by
		WHERE %s
		ORDER BY d.created_at DESC
		LIMIT $%d`, strings.Join(where, " AND "), limitParam)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, platformerrors.Internal("list deployments").WithCause(err)
	}
	defer rows.Close()
	var out []DeploymentJoined
	for rows.Next() {
		var dj DeploymentJoined
		var trig *uuid.UUID
		var trigEmail *string
		if err := rows.Scan(
			&dj.ID, &dj.ServiceID, &dj.ServiceName, &dj.ProjectID, &dj.ProjectName,
			&trig, &trigEmail, &dj.Trigger, &dj.Status,
			&dj.CommitSHA, &dj.CommitMsg, &dj.Ref, &dj.ImageRef, &dj.DurationMs,
			&dj.CreatedAt, &dj.StartedAt, &dj.FinishedAt,
		); err != nil {
			return nil, platformerrors.Internal("scan deployment").WithCause(err)
		}
		dj.TriggeredBy = trig
		dj.TriggerEmail = trigEmail
		out = append(out, dj)
	}
	return out, rows.Err()
}

// GetDeploymentJoined fetches one deployment plus join metadata.
func (r *Repository) GetDeploymentJoined(ctx context.Context, id uuid.UUID) (DeploymentJoined, error) {
	const q = `
		SELECT d.id, d.service_id, s.name AS service_name, p.id AS project_id, p.name AS project_name,
		       d.triggered_by, u.email AS trigger_email, d.trigger, d.status,
		       d.commit_sha, d.commit_message, d.ref, d.image_ref,
		       COALESCE((EXTRACT(EPOCH FROM (d.finished_at - d.started_at)) * 1000)::bigint, 0) AS duration_ms,
		       d.created_at, d.started_at, d.finished_at
		FROM deployments d
		INNER JOIN services s ON s.id = d.service_id
		INNER JOIN projects p ON p.id = s.project_id
		LEFT JOIN users u ON u.id = d.triggered_by
		WHERE d.id = $1`
	var dj DeploymentJoined
	var trig *uuid.UUID
	var trigEmail *string
	if err := r.pool.QueryRow(ctx, q, id).Scan(
		&dj.ID, &dj.ServiceID, &dj.ServiceName, &dj.ProjectID, &dj.ProjectName,
		&trig, &trigEmail, &dj.Trigger, &dj.Status,
		&dj.CommitSHA, &dj.CommitMsg, &dj.Ref, &dj.ImageRef, &dj.DurationMs,
		&dj.CreatedAt, &dj.StartedAt, &dj.FinishedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DeploymentJoined{}, platformerrors.NotFound("deployment not found")
		}
		return DeploymentJoined{}, platformerrors.Internal("get deployment").WithCause(err)
	}
	dj.TriggeredBy = trig
	dj.TriggerEmail = trigEmail
	return dj, nil
}

// DeploymentJoined augments deployments for API projections.
type DeploymentJoined struct {
	ID            uuid.UUID
	ServiceID     uuid.UUID
	ServiceName   string
	ProjectID     uuid.UUID
	ProjectName   string
	TriggeredBy   *uuid.UUID
	TriggerEmail  *string
	Trigger       string
	Status        string
	CommitSHA     *string
	CommitMsg     *string
	Ref           *string
	ImageRef      *string
	DurationMs    int64
	CreatedAt     time.Time
	StartedAt     *time.Time
	FinishedAt    *time.Time
}

func isUniqueViolation(err error) bool {
	const sqlStateUnique = "23505"
	type pgErr interface{ SQLState() string }
	var pe pgErr
	if errors.As(err, &pe) {
		return pe.SQLState() == sqlStateUnique
	}
	return false
}

// NormalizeRepoURL lowercases scheme/host and trims .git for stable matching.
func NormalizeRepoURL(raw string) string {
	u := strings.TrimSpace(strings.ToLower(raw))
	u = strings.TrimSuffix(u, ".git")
	u = strings.TrimSuffix(u, "/")
	if parsed, err := url.Parse(strings.Replace(u, "git@", "https://", 1)); err == nil && parsed.Host != "" {
		return parsed.Scheme + "://" + parsed.Host + strings.TrimSuffix(parsed.Path, ".git")
	}
	return u
}
