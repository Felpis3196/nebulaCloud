package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
)

// DomainRow is a custom hostname bound to a service.
type DomainRow struct {
	ID                uuid.UUID
	ServiceID         uuid.UUID
	Hostname          string
	IsPrimary         bool
	SSLStatus         string
	VerificationToken string
	VerifiedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ListDomainsByService returns domains for a service ordered by created_at.
func (r *Repository) ListDomainsByService(ctx context.Context, serviceID uuid.UUID) ([]DomainRow, error) {
	const q = `
		SELECT id, service_id, hostname, is_primary, ssl_status, verification_token, verified_at, created_at, updated_at
		FROM domains WHERE service_id = $1 ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, q, serviceID)
	if err != nil {
		return nil, platformerrors.Internal("list domains").WithCause(err)
	}
	defer rows.Close()
	var out []DomainRow
	for rows.Next() {
		var d DomainRow
		if err := rows.Scan(&d.ID, &d.ServiceID, &d.Hostname, &d.IsPrimary, &d.SSLStatus, &d.VerificationToken, &d.VerifiedAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, platformerrors.Internal("scan domain").WithCause(err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// CreateDomain inserts a domain row.
func (r *Repository) CreateDomain(ctx context.Context, serviceID uuid.UUID, hostname, token string) (DomainRow, error) {
	const q = `
		INSERT INTO domains (service_id, hostname, verification_token, ssl_status)
		VALUES ($1, $2, $3, 'pending')
		RETURNING id, service_id, hostname, is_primary, ssl_status, verification_token, verified_at, created_at, updated_at
	`
	var d DomainRow
	err := r.pool.QueryRow(ctx, q, serviceID, hostname, token).Scan(
		&d.ID, &d.ServiceID, &d.Hostname, &d.IsPrimary, &d.SSLStatus, &d.VerificationToken, &d.VerifiedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return DomainRow{}, platformerrors.Conflict("hostname already in use")
		}
		return DomainRow{}, platformerrors.Internal("create domain").WithCause(err)
	}
	return d, nil
}

// GetDomain loads a domain by id.
func (r *Repository) GetDomain(ctx context.Context, id uuid.UUID) (DomainRow, error) {
	const q = `
		SELECT id, service_id, hostname, is_primary, ssl_status, verification_token, verified_at, created_at, updated_at
		FROM domains WHERE id = $1
	`
	var d DomainRow
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&d.ID, &d.ServiceID, &d.Hostname, &d.IsPrimary, &d.SSLStatus, &d.VerificationToken, &d.VerifiedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DomainRow{}, platformerrors.NotFound("domain not found")
		}
		return DomainRow{}, platformerrors.Internal("get domain").WithCause(err)
	}
	return d, nil
}

// DeleteDomain removes a domain.
func (r *Repository) DeleteDomain(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM domains WHERE id = $1`, id)
	if err != nil {
		return platformerrors.Internal("delete domain").WithCause(err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound("domain not found")
	}
	return nil
}

// MarkDomainVerified sets ssl_status=issued and verified_at=now.
func (r *Repository) MarkDomainVerified(ctx context.Context, id uuid.UUID) (DomainRow, error) {
	const q = `
		UPDATE domains SET ssl_status = 'issued', verified_at = NOW(), updated_at = NOW()
		WHERE id = $1
		RETURNING id, service_id, hostname, is_primary, ssl_status, verification_token, verified_at, created_at, updated_at
	`
	var d DomainRow
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&d.ID, &d.ServiceID, &d.Hostname, &d.IsPrimary, &d.SSLStatus, &d.VerificationToken, &d.VerifiedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DomainRow{}, platformerrors.NotFound("domain not found")
		}
		return DomainRow{}, platformerrors.Internal("verify domain").WithCause(err)
	}
	return d, nil
}
