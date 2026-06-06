package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nebulacloud/nebula/internal/modules/identity/application"
	"github.com/nebulacloud/nebula/internal/modules/identity/domain"
	identityinfra "github.com/nebulacloud/nebula/internal/modules/identity/infrastructure"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
)

func TestRegisterLoginFlow(t *testing.T) {
	svc := newTestIdentityService(t)
	ctx := context.Background()

	u, err := svc.Register(ctx, application.RegisterCommand{
		Email:       "dev@nebula.test",
		Password:    "valid-password-12",
		DisplayName: "Dev",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if u.Email != "dev@nebula.test" {
		t.Fatalf("email: got %q", u.Email)
	}

	_, err = svc.Register(ctx, application.RegisterCommand{
		Email:    "dev@nebula.test",
		Password: "valid-password-12",
	})
	if platformerrors.KindOf(err) != platformerrors.KindConflict {
		t.Fatalf("expected conflict, got %v", err)
	}

	pair, err := svc.Login(ctx, application.LoginCommand{
		Email:    "dev@nebula.test",
		Password: "valid-password-12",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected token pair")
	}

	_, err = svc.Login(ctx, application.LoginCommand{
		Email:    "dev@nebula.test",
		Password: "wrong-password-12",
	})
	if platformerrors.KindOf(err) != platformerrors.KindUnauthorized {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	svc := newTestIdentityService(t)
	_, err := svc.Register(context.Background(), application.RegisterCommand{
		Email:    "x@y.z",
		Password: "short",
	})
	if platformerrors.KindOf(err) != platformerrors.KindValidation {
		t.Fatalf("expected validation, got %v", err)
	}
}

func newTestIdentityService(t *testing.T) *application.Service {
	t.Helper()
	hasher := identityinfra.NewArgon2idHasher("test-pepper")
	issuer, err := identityinfra.NewJWTIssuer("test-jwt-secret-min-16-chars", "nebula-test")
	if err != nil {
		t.Fatal(err)
	}
	svc, err := application.New(application.Config{
		Users:       &memUserRepo{byEmail: map[string]domain.User{}},
		Sessions:    &memSessionRepo{sessions: map[uuid.UUID]domain.Session{}},
		Memberships: &memMembershipRepo{},
		Hasher:      hasher,
		Tokens:      issuer,
		Refresh:     identityinfra.NewRefreshGenerator(),
		Audit:       noopAudit{},
		AccessTTL:   15 * time.Minute,
		RefreshTTL:  24 * time.Hour,
		Clock:       time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

type memUserRepo struct {
	byEmail map[string]domain.User
}

func (m *memUserRepo) Create(_ context.Context, u domain.User) (domain.User, error) {
	if _, ok := m.byEmail[u.Email]; ok {
		return domain.User{}, platformerrors.Conflict("email already registered")
	}
	m.byEmail[u.Email] = u
	return u, nil
}

func (m *memUserRepo) FindByEmail(_ context.Context, email string) (domain.User, error) {
	u, ok := m.byEmail[domain.NormaliseEmail(email)]
	if !ok {
		return domain.User{}, platformerrors.NotFound("user not found")
	}
	return u, nil
}

func (m *memUserRepo) FindByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	for _, u := range m.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return domain.User{}, platformerrors.NotFound("user not found")
}

func (m *memUserRepo) UpdateLastLogin(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

type memSessionRepo struct {
	sessions map[uuid.UUID]domain.Session
}

func (m *memSessionRepo) Create(_ context.Context, s domain.Session) (domain.Session, error) {
	m.sessions[s.ID] = s
	return s, nil
}

func (m *memSessionRepo) FindByRefreshHash(_ context.Context, hash []byte) (domain.Session, error) {
	for _, s := range m.sessions {
		if string(s.RefreshTokenHash) == string(hash) {
			return s, nil
		}
	}
	return domain.Session{}, platformerrors.NotFound("session not found")
}

func (m *memSessionRepo) Revoke(_ context.Context, id uuid.UUID, at time.Time) error {
	s, ok := m.sessions[id]
	if !ok {
		return platformerrors.NotFound("session not found")
	}
	s.RevokedAt = &at
	m.sessions[id] = s
	return nil
}

func (m *memSessionRepo) RevokeAllForUser(_ context.Context, userID uuid.UUID, at time.Time) error {
	for id, s := range m.sessions {
		if s.UserID == userID {
			s.RevokedAt = &at
			m.sessions[id] = s
		}
	}
	return nil
}

type memMembershipRepo struct{}

func (m *memMembershipRepo) ListForUser(context.Context, uuid.UUID) ([]domain.Membership, error) {
	return nil, nil
}

func (m *memMembershipRepo) Find(context.Context, uuid.UUID, uuid.UUID) (domain.Membership, error) {
	return domain.Membership{}, platformerrors.NotFound("membership not found")
}

type noopAudit struct{}

func (noopAudit) Record(context.Context, string, *uuid.UUID, map[string]any) {}
