package interfaces_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/nebulacloud/nebula/internal/modules/identity/application"
	"github.com/nebulacloud/nebula/internal/modules/identity/domain"
	identityif "github.com/nebulacloud/nebula/internal/modules/identity/interfaces"
	identityinfra "github.com/nebulacloud/nebula/internal/modules/identity/infrastructure"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
)

func TestHTTPRegisterLoginMe(t *testing.T) {
	router := newTestAuthRouter(t, "", "", "", "http://localhost:3000")

	regBody := `{"email":"http@nebula.test","password":"valid-password-12","display_name":"HTTP"}`
	reg := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regBody))
	reg.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, reg)
	if rr.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", rr.Code, rr.Body.String())
	}

	dup := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regBody))
	dup.Header.Set("Content-Type", "application/json")
	dupRR := httptest.NewRecorder()
	router.ServeHTTP(dupRR, dup)
	if dupRR.Code != http.StatusConflict {
		t.Fatalf("duplicate register status=%d", dupRR.Code)
	}

	loginBody := `{"email":"http@nebula.test","password":"valid-password-12"}`
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	login.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	router.ServeHTTP(loginRR, login)
	if loginRR.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", loginRR.Code, loginRR.Body.String())
	}
	var loginEnv struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginRR.Body.Bytes(), &loginEnv); err != nil {
		t.Fatal(err)
	}
	if loginEnv.Data.AccessToken == "" {
		t.Fatal("missing access_token")
	}

	me := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	me.Header.Set("Authorization", "Bearer "+loginEnv.Data.AccessToken)
	meRR := httptest.NewRecorder()
	router.ServeHTTP(meRR, me)
	if meRR.Code != http.StatusOK {
		t.Fatalf("me status=%d body=%s", meRR.Code, meRR.Body.String())
	}

	unauth := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	unauthRR := httptest.NewRecorder()
	router.ServeHTTP(unauthRR, unauth)
	if unauthRR.Code != http.StatusUnauthorized {
		t.Fatalf("me unauth status=%d", unauthRR.Code)
	}
}

func TestGitHubOAuthNotConfigured(t *testing.T) {
	router := newTestAuthRouter(t, "", "", "", "http://localhost:3000")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/github", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("github start status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGitHubOAuthStartRedirect(t *testing.T) {
	router := newTestAuthRouter(t, "test-client-id", "secret", "http://localhost/callback", "http://localhost:3000")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/github", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d body=%s", rr.Code, rr.Body.String())
	}
	loc := rr.Header().Get("Location")
	if loc == "" || !strings.Contains(loc, "github.com/login/oauth/authorize") {
		t.Fatalf("bad location: %q", loc)
	}
}

func TestGitHubOAuthCallbackInvalidState(t *testing.T) {
	router := newTestAuthRouter(t, "id", "secret", "http://localhost/cb", "http://localhost:3000")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/github/callback?code=abc&state=wrong", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("callback status=%d body=%s", rr.Code, rr.Body.String())
	}
	loc := rr.Header().Get("Location")
	if loc == "" || !strings.Contains(loc, "github=error") {
		t.Fatalf("expected redirect with github=error, got %q", loc)
	}
}

func newTestAuthRouter(t *testing.T, ghID, ghSecret, ghRedirect, appURL string) http.Handler {
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
	h := identityif.NewHandler(svc, ghID, ghSecret, ghRedirect, appURL)
	r := chi.NewRouter()
	r.Route("/api/v1", func(api chi.Router) {
		h.Mount(api)
		api.Group(func(authed chi.Router) {
			authed.Use(identityif.Authenticator(svc))
			h.MountAuthenticated(authed)
		})
	})
	return r
}

// Reuse lightweight fakes (duplicated from application tests to avoid export cycles).
type memUserRepo struct{ byEmail map[string]domain.User }

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
func (m *memUserRepo) UpdateLastLogin(context.Context, uuid.UUID, time.Time) error { return nil }

type memSessionRepo struct{ sessions map[uuid.UUID]domain.Session }

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
