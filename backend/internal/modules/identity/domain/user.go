// Package domain holds the Identity bounded-context entities, value
// objects, and repository ports. It must remain free of any infrastructure
// concerns (database drivers, HTTP, JWT, hashing libraries) — those live
// in `infrastructure` and adapt to the ports defined here.
package domain

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nebulacloud/nebula/internal/platform/auth"
)

// User is the aggregate root for the Identity context.
type User struct {
	ID            uuid.UUID
	Email         string
	PasswordHash  string
	DisplayName   string
	AvatarURL     string
	IsActive      bool
	IsAdmin       bool
	EmailVerified bool
	MFAEnabled    bool
	LastLoginAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CanLogin reports whether the user is allowed to authenticate.
func (u User) CanLogin() bool { return u.IsActive }

// NormaliseEmail lower-cases and trims an email so equality lookups behave.
// The email column is CITEXT so the database matches case-insensitively
// either way; the helper keeps logs and audit entries consistent.
func NormaliseEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// Session represents a single refresh-token-bearing session for a user.
type Session struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	RefreshTokenHash  []byte
	IP                string
	UserAgent         string
	ExpiresAt         time.Time
	RevokedAt         *time.Time
	CreatedAt         time.Time
}

// IsActive reports whether the session is still usable at the supplied
// instant. Both expiry and revocation are considered.
func (s Session) IsActive(at time.Time) bool {
	if s.RevokedAt != nil {
		return false
	}
	return at.Before(s.ExpiresAt)
}

// Membership ties a user to an organisation with a role. The Identity module
// owns memberships only insofar as login + RBAC require them; the Projects
// module is the authoritative writer.
type Membership struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Role           auth.Role
}

// ----------------------------------------------------------------------------
// Ports
// ----------------------------------------------------------------------------

// UserRepository persists User aggregates.
type UserRepository interface {
	Create(ctx context.Context, u User) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByID(ctx context.Context, id uuid.UUID) (User, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error
}

// SessionRepository persists Session aggregates.
type SessionRepository interface {
	Create(ctx context.Context, s Session) (Session, error)
	FindByRefreshHash(ctx context.Context, hash []byte) (Session, error)
	Revoke(ctx context.Context, id uuid.UUID, at time.Time) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID, at time.Time) error
}

// MembershipRepository reads org memberships for the logged-in user.
type MembershipRepository interface {
	ListForUser(ctx context.Context, userID uuid.UUID) ([]Membership, error)
	Find(ctx context.Context, userID, orgID uuid.UUID) (Membership, error)
}

// PasswordHasher abstracts the password derivation function (Argon2id in
// production). The port lives in the domain so application code never
// imports a crypto library.
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Verify(plain, hash string) (bool, error)
}

// AccessTokenIssuer abstracts JWT signing / verification.
type AccessTokenIssuer interface {
	Issue(ctx context.Context, claims AccessClaims) (token string, err error)
	Verify(ctx context.Context, token string) (AccessClaims, error)
}

// AccessClaims is the verified content of an access token.
type AccessClaims struct {
	Subject        string
	Email          string
	OrganizationID string
	Role           auth.Role
	SessionID      string
	TokenID        string
	IssuedAt       time.Time
	ExpiresAt      time.Time
}

// RefreshTokenGenerator yields a fresh opaque refresh token plus its hash.
type RefreshTokenGenerator interface {
	Generate() (token string, hash []byte, err error)
	Hash(token string) ([]byte, error)
}

// AuditRecorder is implemented by the audit module; the identity module
// keeps a thin port to avoid a hard dependency.
type AuditRecorder interface {
	Record(ctx context.Context, action string, actorID *uuid.UUID, metadata map[string]any)
}
