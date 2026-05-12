package infrastructure

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
)

// BuildJobContext is everything required to run a Docker build worker job.
type BuildJobContext struct {
	BuildID        uuid.UUID
	DeploymentID   uuid.UUID
	ServiceID      uuid.UUID
	OrganizationID uuid.UUID
	ProjectID      uuid.UUID
	OrgSlug        string
	ProjectSlug    string
	ServiceSlug    string
	RepoURL        string
	Ref            string
	BuildConfig    json.RawMessage
	RuntimeConfig  json.RawMessage
}

// LoadBuildJobContext loads build + related rows.
func (r *Repository) LoadBuildJobContext(ctx context.Context, buildID uuid.UUID) (BuildJobContext, error) {
	const q = `
		SELECT b.id, d.id, s.id, p.organization_id, p.id,
	       o.slug, p.slug, s.slug,
		       COALESCE(p.repo_url, '') AS repo_url,
		       p.default_branch,
		       COALESCE(s.build_config::text, '{}')::jsonb,
		       COALESCE(s.runtime_config::text, '{}')::jsonb
		FROM builds b
		INNER JOIN deployments d ON d.id = b.deployment_id
		INNER JOIN services s ON s.id = d.service_id
		INNER JOIN projects p ON p.id = s.project_id
		INNER JOIN organizations o ON o.id = p.organization_id
		WHERE b.id = $1`

	var bc BuildJobContext
	if err := r.pool.QueryRow(ctx, q, buildID).Scan(
		&bc.BuildID, &bc.DeploymentID, &bc.ServiceID, &bc.OrganizationID, &bc.ProjectID,
		&bc.OrgSlug, &bc.ProjectSlug, &bc.ServiceSlug,
		&bc.RepoURL, &bc.Ref, &bc.BuildConfig, &bc.RuntimeConfig,
	); err != nil {
		if err == pgx.ErrNoRows {
			return BuildJobContext{}, platformerrors.NotFound("build not found")
		}
		return BuildJobContext{}, platformerrors.Internal("load build ctx").WithCause(err)
	}
	return bc, nil
}
