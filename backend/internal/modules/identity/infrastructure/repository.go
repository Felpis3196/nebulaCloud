package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nebulacloud/nebula/internal/modules/identity/domain"
	"github.com/nebulacloud/nebula/internal/platform/auth"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
)

// PostgresUserRepo implements domain.UserRepository on top of pgxpool.Pool.
type PostgresUserRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepo constructs a PostgresUserRepo.
func NewPostgresUserRepo(pool *pgxpool.Pool) *PostgresUserRepo {
	return &PostgresUserRepo{pool: pool}
}

// Create inserts a new user. Email uniqueness is enforced at the DB level
// (CITEXT UNIQUE); we translate the resulting violation into a Conflict.
func (r *PostgresUserRepo) Create(ctx context.Context, u domain.User) (domain.User, error) {
	const q = `
		INSERT INTO users
		  (id, email, password_hash, display_name, avatar_url, is_active, is_admin,
		   email_verified, mfa_enabled, last_login_at, created_at, updated_at)
		VALUES
		  ($1, $2, $3, NULLIF($4,''), NULLIF($5,''), $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, email, password_hash, COALESCE(display_name,''), COALESCE(avatar_url,''),
		          is_active, is_admin, email_verified, mfa_enabled, last_login_at,
		          created_at, updated_at
	`
	row := r.pool.QueryRow(ctx, q,
		u.ID, u.Email, u.PasswordHash, u.DisplayName, u.AvatarURL,
		u.IsActive, u.IsAdmin, u.EmailVerified, u.MFAEnabled,
		u.LastLoginAt, u.CreatedAt, u.UpdatedAt,
	)
	out, err := scanUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, platformerrors.Conflict("email already registered")
		}
		return domain.User{}, platformerrors.Internal("create user").WithCause(err)
	}
	return out, nil
}

// FindByEmail looks up a user by their CITEXT-normalised email.
func (r *PostgresUserRepo) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	const q = `
		SELECT id, email, password_hash, COALESCE(display_name,''), COALESCE(avatar_url,''),
		       is_active, is_admin, email_verified, mfa_enabled, last_login_at,
		       created_at, updated_at
		FROM users
		WHERE email = $1
	`
	row := r.pool.QueryRow(ctx, q, email)
	out, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, platformerrors.NotFound("user not found")
		}
		return domain.User{}, platformerrors.Internal("find user by email").WithCause(err)
	}
	return out, nil
}

// FindByID looks up a user by primary key.
func (r *PostgresUserRepo) FindByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	const q = `
		SELECT id, email, password_hash, COALESCE(display_name,''), COALESCE(avatar_url,''),
		       is_active, is_admin, email_verified, mfa_enabled, last_login_at,
		       created_at, updated_at
		FROM users WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, q, id)
	out, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, platformerrors.NotFound("user not found")
		}
		return domain.User{}, platformerrors.Internal("find user by id").WithCause(err)
	}
	return out, nil
}

// UpdateLastLogin stamps last_login_at for the given user.
func (r *PostgresUserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error {
	const q = `UPDATE users SET last_login_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id, at)
	return err
}

// PostgresSessionRepo implements domain.SessionRepository.
type PostgresSessionRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresSessionRepo constructs a PostgresSessionRepo.
func NewPostgresSessionRepo(pool *pgxpool.Pool) *PostgresSessionRepo {
	return &PostgresSessionRepo{pool: pool}
}

// Create persists a new session.
func (r *PostgresSessionRepo) Create(ctx context.Context, s domain.Session) (domain.Session, error) {
	const q = `
		INSERT INTO sessions (id, user_id, refresh_token_hash, ip, user_agent, expires_at, created_at)
		VALUES ($1, $2, $3, NULLIF($4,'')::inet, NULLIF($5,''), $6, $7)
		RETURNING id, user_id, refresh_token_hash, COALESCE(host(ip),''),
		          COALESCE(user_agent,''), expires_at, revoked_at, created_at
	`
	row := r.pool.QueryRow(ctx, q,
		s.ID, s.UserID, s.RefreshTokenHash, s.IP, s.UserAgent, s.ExpiresAt, s.CreatedAt,
	)
	out, err := scanSession(row)
	if err != nil {
		return domain.Session{}, platformerrors.Internal("create session").WithCause(err)
	}
	return out, nil
}

// FindByRefreshHash returns the session matching the supplied hash.
func (r *PostgresSessionRepo) FindByRefreshHash(ctx context.Context, hash []byte) (domain.Session, error) {
	const q = `
		SELECT id, user_id, refresh_token_hash, COALESCE(host(ip),''),
		       COALESCE(user_agent,''), expires_at, revoked_at, created_at
		FROM sessions
		WHERE refresh_token_hash = $1
	`
	row := r.pool.QueryRow(ctx, q, hash)
	out, err := scanSession(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, platformerrors.NotFound("session not found")
		}
		return domain.Session{}, platformerrors.Internal("find session").WithCause(err)
	}
	return out, nil
}

// Revoke marks the supplied session as revoked.
func (r *PostgresSessionRepo) Revoke(ctx context.Context, id uuid.UUID, at time.Time) error {
	const q = `UPDATE sessions SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, q, id, at)
	return err
}

// RevokeAllForUser revokes every active session belonging to the user.
func (r *PostgresSessionRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID, at time.Time) error {
	const q = `UPDATE sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, q, userID, at)
	return err
}

// PostgresMembershipRepo implements domain.MembershipRepository.
type PostgresMembershipRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresMembershipRepo constructs a PostgresMembershipRepo.
func NewPostgresMembershipRepo(pool *pgxpool.Pool) *PostgresMembershipRepo {
	return &PostgresMembershipRepo{pool: pool}
}

// ListForUser returns every membership for the user.
func (r *PostgresMembershipRepo) ListForUser(ctx context.Context, userID uuid.UUID) ([]domain.Membership, error) {
	const q = `SELECT organization_id, user_id, role FROM memberships WHERE user_id = $1`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, platformerrors.Internal("list memberships").WithCause(err)
	}
	defer rows.Close()
	var out []domain.Membership
	for rows.Next() {
		var m domain.Membership
		var role string
		if err := rows.Scan(&m.OrganizationID, &m.UserID, &role); err != nil {
			return nil, platformerrors.Internal("scan membership").WithCause(err)
		}
		m.Role = auth.Role(role)
		out = append(out, m)
	}
	return out, rows.Err()
}

// Find returns a single membership.
func (r *PostgresMembershipRepo) Find(ctx context.Context, userID, orgID uuid.UUID) (domain.Membership, error) {
	const q = `SELECT organization_id, user_id, role FROM memberships WHERE user_id = $1 AND organization_id = $2`
	row := r.pool.QueryRow(ctx, q, userID, orgID)
	var m domain.Membership
	var role string
	if err := row.Scan(&m.OrganizationID, &m.UserID, &role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Membership{}, platformerrors.NotFound("membership not found")
		}
		return domain.Membership{}, platformerrors.Internal("find membership").WithCause(err)
	}
	m.Role = auth.Role(role)
	return m, nil
}

// ----------------------------------------------------------------------------
// scanners
// ----------------------------------------------------------------------------

type rowScanner interface {
	Scan(dst ...any) error
}

func scanUser(row rowScanner) (domain.User, error) {
	var u domain.User
	var lastLogin *time.Time
	if err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.IsActive, &u.IsAdmin, &u.EmailVerified, &u.MFAEnabled,
		&lastLogin, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return domain.User{}, err
	}
	u.LastLoginAt = lastLogin
	return u, nil
}

func scanSession(row rowScanner) (domain.Session, error) {
	var s domain.Session
	var revoked *time.Time
	if err := row.Scan(
		&s.ID, &s.UserID, &s.RefreshTokenHash, &s.IP, &s.UserAgent,
		&s.ExpiresAt, &revoked, &s.CreatedAt,
	); err != nil {
		return domain.Session{}, err
	}
	s.RevokedAt = revoked
	return s, nil
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
