// Package application contains the Identity module's use cases. It speaks
// only domain types and ports defined in the domain package; concrete
// implementations are wired up by main.go via dependency injection.
package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nebulacloud/nebula/internal/modules/identity/domain"
	"github.com/nebulacloud/nebula/internal/platform/auth"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
	"github.com/nebulacloud/nebula/internal/platform/logger"
)

// Service is the public entry point for the Identity module's use cases.
//
// It is constructed by `New(...)` and then mounted onto the HTTP layer in
// `interfaces`. The service is safe to share across goroutines.
type Service struct {
	users        domain.UserRepository
	sessions     domain.SessionRepository
	memberships  domain.MembershipRepository
	hasher       domain.PasswordHasher
	tokens       domain.AccessTokenIssuer
	refresh      domain.RefreshTokenGenerator
	audit        domain.AuditRecorder
	clock        func() time.Time
	accessTTL    time.Duration
	refreshTTL   time.Duration
}

// Config bundles the constructor arguments. Using a struct keeps the
// signature stable as we add knobs later.
type Config struct {
	Users         domain.UserRepository
	Sessions      domain.SessionRepository
	Memberships   domain.MembershipRepository
	Hasher        domain.PasswordHasher
	Tokens        domain.AccessTokenIssuer
	Refresh       domain.RefreshTokenGenerator
	Audit         domain.AuditRecorder
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	Clock         func() time.Time // injected for tests; defaults to time.Now
}

// New builds a Service. Returns an error if any required dependency is nil.
func New(cfg Config) (*Service, error) {
	if cfg.Users == nil || cfg.Sessions == nil || cfg.Hasher == nil ||
		cfg.Tokens == nil || cfg.Refresh == nil {
		return nil, errors.New("identity: missing required dependencies")
	}
	if cfg.AccessTTL == 0 {
		cfg.AccessTTL = 15 * time.Minute
	}
	if cfg.RefreshTTL == 0 {
		cfg.RefreshTTL = 30 * 24 * time.Hour
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	return &Service{
		users:       cfg.Users,
		sessions:    cfg.Sessions,
		memberships: cfg.Memberships,
		hasher:      cfg.Hasher,
		tokens:      cfg.Tokens,
		refresh:     cfg.Refresh,
		audit:       cfg.Audit,
		clock:       cfg.Clock,
		accessTTL:   cfg.AccessTTL,
		refreshTTL:  cfg.RefreshTTL,
	}, nil
}

// ----------------------------------------------------------------------------
// Commands / queries
// ----------------------------------------------------------------------------

// RegisterCommand registers a new user.
type RegisterCommand struct {
	Email       string
	Password    string
	DisplayName string
}

// LoginCommand authenticates a user and issues a session.
type LoginCommand struct {
	Email     string
	Password  string
	IP        string
	UserAgent string
}

// RefreshCommand rotates a refresh token.
type RefreshCommand struct {
	RefreshToken string
	IP           string
	UserAgent    string
}

// LogoutCommand revokes a session.
type LogoutCommand struct {
	RefreshToken string
}

// TokenPair is what callers receive on successful login / refresh.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	AccessExpiry time.Time
	RefreshExpiry time.Time
	User         domain.User
}

// ----------------------------------------------------------------------------
// Use cases
// ----------------------------------------------------------------------------

// Register creates a new user with a hashed password.
func (s *Service) Register(ctx context.Context, cmd RegisterCommand) (domain.User, error) {
	email := domain.NormaliseEmail(cmd.Email)
	if email == "" || cmd.Password == "" {
		return domain.User{}, platformerrors.Validation("email and password are required")
	}
	if len(cmd.Password) < 12 {
		return domain.User{}, platformerrors.Validation("password must be at least 12 characters")
	}

	if existing, err := s.users.FindByEmail(ctx, email); err == nil && existing.ID != uuid.Nil {
		return domain.User{}, platformerrors.Conflict("email already registered")
	} else if err != nil && platformerrors.KindOf(err) != platformerrors.KindNotFound {
		return domain.User{}, err
	}

	hash, err := s.hasher.Hash(cmd.Password)
	if err != nil {
		return domain.User{}, platformerrors.Internal("hash password").WithCause(err)
	}

	now := s.clock()
	u := domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hash,
		DisplayName:  strings.TrimSpace(cmd.DisplayName),
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	created, err := s.users.Create(ctx, u)
	if err != nil {
		return domain.User{}, err
	}
	s.recordAudit(ctx, "user.registered", &created.ID, map[string]any{"email": email})
	return created, nil
}

// Login authenticates the user and issues a token pair.
//
// The returned error is intentionally generic on missing-user / wrong-password
// to avoid user-enumeration leaks; the audit trail still records detail.
func (s *Service) Login(ctx context.Context, cmd LoginCommand) (TokenPair, error) {
	email := domain.NormaliseEmail(cmd.Email)
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		s.recordAudit(ctx, "user.login.failed", nil, map[string]any{"email": email, "reason": "no_user"})
		return TokenPair{}, platformerrors.Unauthorized("invalid credentials")
	}
	if !user.CanLogin() {
		s.recordAudit(ctx, "user.login.failed", &user.ID, map[string]any{"email": email, "reason": "inactive"})
		return TokenPair{}, platformerrors.Forbidden("account is disabled")
	}

	ok, err := s.hasher.Verify(cmd.Password, user.PasswordHash)
	if err != nil {
		return TokenPair{}, platformerrors.Internal("verify password").WithCause(err)
	}
	if !ok {
		s.recordAudit(ctx, "user.login.failed", &user.ID, map[string]any{"email": email, "reason": "bad_password"})
		return TokenPair{}, platformerrors.Unauthorized("invalid credentials")
	}

	pair, err := s.issuePair(ctx, user, cmd.IP, cmd.UserAgent)
	if err != nil {
		return TokenPair{}, err
	}

	if err := s.users.UpdateLastLogin(ctx, user.ID, s.clock()); err != nil {
		logger.FromContext(ctx).Warn("identity.last_login.update_failed", "error", err)
	}
	s.recordAudit(ctx, "user.login", &user.ID, map[string]any{"email": email})
	return pair, nil
}

// Refresh validates the supplied refresh token, revokes it, and issues a new
// pair. Reuse of a previously rotated token is treated as compromise: every
// session for the user is revoked and an audit event is emitted.
func (s *Service) Refresh(ctx context.Context, cmd RefreshCommand) (TokenPair, error) {
	if cmd.RefreshToken == "" {
		return TokenPair{}, platformerrors.Unauthorized("missing refresh token")
	}
	hash, err := s.refresh.Hash(cmd.RefreshToken)
	if err != nil {
		return TokenPair{}, platformerrors.Unauthorized("invalid refresh token")
	}
	session, err := s.sessions.FindByRefreshHash(ctx, hash)
	if err != nil {
		return TokenPair{}, platformerrors.Unauthorized("invalid refresh token")
	}
	now := s.clock()
	if !session.IsActive(now) {
		// Possible reuse of a rotated token — kill every session for the user.
		_ = s.sessions.RevokeAllForUser(ctx, session.UserID, now)
		s.recordAudit(ctx, "user.refresh.reuse", &session.UserID, map[string]any{"session_id": session.ID})
		return TokenPair{}, platformerrors.Unauthorized("refresh token expired or revoked")
	}

	user, err := s.users.FindByID(ctx, session.UserID)
	if err != nil {
		return TokenPair{}, platformerrors.Unauthorized("invalid refresh token")
	}
	if !user.CanLogin() {
		_ = s.sessions.Revoke(ctx, session.ID, now)
		return TokenPair{}, platformerrors.Forbidden("account is disabled")
	}

	if err := s.sessions.Revoke(ctx, session.ID, now); err != nil {
		return TokenPair{}, platformerrors.Internal("revoke session").WithCause(err)
	}

	pair, err := s.issuePair(ctx, user, cmd.IP, cmd.UserAgent)
	if err != nil {
		return TokenPair{}, err
	}
	s.recordAudit(ctx, "user.refresh", &user.ID, nil)
	return pair, nil
}

// Logout revokes the supplied refresh-token's session.
func (s *Service) Logout(ctx context.Context, cmd LogoutCommand) error {
	if cmd.RefreshToken == "" {
		return nil
	}
	hash, err := s.refresh.Hash(cmd.RefreshToken)
	if err != nil {
		return nil
	}
	session, err := s.sessions.FindByRefreshHash(ctx, hash)
	if err != nil {
		return nil
	}
	now := s.clock()
	if err := s.sessions.Revoke(ctx, session.ID, now); err != nil {
		return platformerrors.Internal("revoke session").WithCause(err)
	}
	s.recordAudit(ctx, "user.logout", &session.UserID, nil)
	return nil
}

// VerifyAccessToken decodes and validates a bearer token, returning the
// principal that should flow on the request context.
func (s *Service) VerifyAccessToken(ctx context.Context, token string) (auth.Principal, error) {
	claims, err := s.tokens.Verify(ctx, token)
	if err != nil {
		return auth.Principal{}, platformerrors.Unauthorized("invalid access token").WithCause(err)
	}
	return auth.Principal{
		UserID:         claims.Subject,
		Email:          claims.Email,
		OrganizationID: claims.OrganizationID,
		Role:           claims.Role,
		SessionID:      claims.SessionID,
		TokenID:        claims.TokenID,
	}, nil
}

// Me returns the currently authenticated user.
func (s *Service) Me(ctx context.Context, principal auth.Principal) (domain.User, error) {
	id, err := uuid.Parse(principal.UserID)
	if err != nil {
		return domain.User{}, platformerrors.Unauthorized("invalid principal")
	}
	return s.users.FindByID(ctx, id)
}

// ----------------------------------------------------------------------------
// internals
// ----------------------------------------------------------------------------

func (s *Service) issuePair(ctx context.Context, user domain.User, ip, ua string) (TokenPair, error) {
	now := s.clock()

	role, orgID := s.resolvePrimaryMembership(ctx, user)

	access, accessExp, err := s.issueAccess(ctx, user, role, orgID, now)
	if err != nil {
		return TokenPair{}, err
	}

	rawRefresh, refreshHash, err := s.refresh.Generate()
	if err != nil {
		return TokenPair{}, platformerrors.Internal("generate refresh").WithCause(err)
	}
	refreshExp := now.Add(s.refreshTTL)

	session := domain.Session{
		ID:               uuid.New(),
		UserID:           user.ID,
		RefreshTokenHash: refreshHash,
		IP:               ip,
		UserAgent:        ua,
		ExpiresAt:        refreshExp,
		CreatedAt:        now,
	}
	if _, err := s.sessions.Create(ctx, session); err != nil {
		return TokenPair{}, platformerrors.Internal("persist session").WithCause(err)
	}

	return TokenPair{
		AccessToken:   access,
		RefreshToken:  rawRefresh,
		AccessExpiry:  accessExp,
		RefreshExpiry: refreshExp,
		User:          user,
	}, nil
}

func (s *Service) issueAccess(ctx context.Context, user domain.User, role auth.Role, orgID string, now time.Time) (string, time.Time, error) {
	exp := now.Add(s.accessTTL)
	tokenID := uuid.NewString()
	claims := domain.AccessClaims{
		Subject:        user.ID.String(),
		Email:          user.Email,
		OrganizationID: orgID,
		Role:           role,
		TokenID:        tokenID,
		IssuedAt:       now,
		ExpiresAt:      exp,
	}
	token, err := s.tokens.Issue(ctx, claims)
	if err != nil {
		return "", time.Time{}, platformerrors.Internal("issue access token").WithCause(err)
	}
	return token, exp, nil
}

// resolvePrimaryMembership picks a (role, org_id) for the access token.
// In the MVP we use the highest-privileged membership; multi-org switching
// is a Phase 2 concern.
func (s *Service) resolvePrimaryMembership(ctx context.Context, user domain.User) (auth.Role, string) {
	if user.IsAdmin {
		return auth.RoleAdmin, ""
	}
	if s.memberships == nil {
		return auth.RoleViewer, ""
	}
	memberships, err := s.memberships.ListForUser(ctx, user.ID)
	if err != nil || len(memberships) == 0 {
		return auth.RoleViewer, ""
	}
	best := memberships[0]
	for _, m := range memberships[1:] {
		if m.Role.Rank() > best.Role.Rank() {
			best = m
		}
	}
	return best.Role, best.OrganizationID.String()
}

func (s *Service) recordAudit(ctx context.Context, action string, actor *uuid.UUID, metadata map[string]any) {
	if s.audit == nil {
		return
	}
	s.audit.Record(ctx, action, actor, metadata)
}
