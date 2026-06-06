package interfaces_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	identityapp "github.com/nebulacloud/nebula/internal/modules/identity/application"
	identityinfra "github.com/nebulacloud/nebula/internal/modules/identity/infrastructure"
	identityif "github.com/nebulacloud/nebula/internal/modules/identity/interfaces"
	projectsapp "github.com/nebulacloud/nebula/internal/modules/projects/application"
	projectsinfra "github.com/nebulacloud/nebula/internal/modules/projects/infrastructure"
	projectsif "github.com/nebulacloud/nebula/internal/modules/projects/interfaces"
	"github.com/nebulacloud/nebula/internal/platform/config"
	"github.com/nebulacloud/nebula/internal/platform/database"
)

// TestOrgProjectMembershipHTTP exercises org membership ACL over HTTP.
// Requires NEBULA_TEST_DSN (e.g. postgres://nebula:nebula@localhost:5432/nebula?sslmode=disable).
func TestOrgProjectMembershipHTTP(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()
	pool, err := database.Connect(ctx, database.Config{DSN: dsn, MaxConns: 4, MinConns: 1})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	router := mountProjectsTestRouter(t, pool)

	emailA := "member-a-" + uuid.NewString()[:8] + "@nebula.test"
	emailB := "member-b-" + uuid.NewString()[:8] + "@nebula.test"
	pass := "valid-password-12"

	tokenA := registerAndLogin(t, router, emailA, pass)
	tokenB := registerAndLogin(t, router, emailB, pass)

	orgBody := `{"slug":"org-` + uuid.NewString()[:8] + `","name":"Test Org"}`
	orgReq := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBufferString(orgBody))
	orgReq.Header.Set("Content-Type", "application/json")
	orgReq.Header.Set("Authorization", "Bearer "+tokenA)
	orgRR := httptest.NewRecorder()
	router.ServeHTTP(orgRR, orgReq)
	if orgRR.Code != http.StatusCreated {
		t.Fatalf("create org A status=%d body=%s", orgRR.Code, orgRR.Body.String())
	}
	var orgEnv struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(orgRR.Body.Bytes(), &orgEnv); err != nil {
		t.Fatal(err)
	}
	orgID := orgEnv.Data.ID

	projBody := `{"slug":"app-1","name":"App","default_branch":"main"}`
	projReq := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+orgID+"/projects", bytes.NewBufferString(projBody))
	projReq.Header.Set("Content-Type", "application/json")
	projReq.Header.Set("Authorization", "Bearer "+tokenA)
	projRR := httptest.NewRecorder()
	router.ServeHTTP(projRR, projReq)
	if projRR.Code != http.StatusCreated {
		t.Fatalf("create project A status=%d body=%s", projRR.Code, projRR.Body.String())
	}
	var projEnv struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(projRR.Body.Bytes(), &projEnv); err != nil {
		t.Fatal(err)
	}
	projectID := projEnv.Data.ID

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID, nil)
	getReq.Header.Set("Authorization", "Bearer "+tokenA)
	getRR := httptest.NewRecorder()
	router.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get project status=%d want 200 body=%s", getRR.Code, getRR.Body.String())
	}

	patchBody := `{"repo_url":"https://github.com/docker/welcome-to-docker","default_branch":"main"}`
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/"+projectID, bytes.NewBufferString(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Authorization", "Bearer "+tokenA)
	patchRR := httptest.NewRecorder()
	router.ServeHTTP(patchRR, patchReq)
	if patchRR.Code != http.StatusOK {
		t.Fatalf("patch project status=%d want 200 body=%s", patchRR.Code, patchRR.Body.String())
	}
	var patched struct {
		Data struct {
			RepoURL string `json:"repo_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(patchRR.Body.Bytes(), &patched); err != nil {
		t.Fatal(err)
	}
	if patched.Data.RepoURL != "https://github.com/docker/welcome-to-docker" {
		t.Fatalf("repo_url=%q", patched.Data.RepoURL)
	}

	getDepReq := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID, nil)
	getDepReq.Header.Set("Authorization", "Bearer "+tokenA)
	getDepRR := httptest.NewRecorder()
	router.ServeHTTP(getDepRR, getDepReq)
	if getDepRR.Code != http.StatusOK {
		t.Fatalf("get project status=%d body=%s", getDepRR.Code, getDepRR.Body.String())
	}

	logsReq := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/00000000-0000-0000-0000-000000000099/build-logs", nil)
	logsReq.Header.Set("Authorization", "Bearer "+tokenA)
	logsRR := httptest.NewRecorder()
	router.ServeHTTP(logsRR, logsReq)
	if logsRR.Code != http.StatusNotFound {
		t.Fatalf("build-logs unknown deployment status=%d want 404 body=%s", logsRR.Code, logsRR.Body.String())
	}

	projReqB := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+orgID+"/projects", bytes.NewBufferString(projBody))
	projReqB.Header.Set("Content-Type", "application/json")
	projReqB.Header.Set("Authorization", "Bearer "+tokenB)
	projRRB := httptest.NewRecorder()
	router.ServeHTTP(projRRB, projReqB)
	if projRRB.Code != http.StatusNotFound {
		t.Fatalf("user B create project status=%d want 404 body=%s", projRRB.Code, projRRB.Body.String())
	}
	if !bytes.Contains(projRRB.Body.Bytes(), []byte("membership not found")) {
		t.Fatalf("expected membership not found, body=%s", projRRB.Body.String())
	}
}

func testDSN(t *testing.T) string {
	t.Helper()
	for _, key := range []string{"NEBULA_TEST_DSN", "TEST_DATABASE_URL"} {
		if v := getenv(key); v != "" {
			return v
		}
	}
	t.Skip("set NEBULA_TEST_DSN to run membership HTTP integration test")
	return ""
}

func getenv(k string) string {
	return os.Getenv(k)
}

func mountProjectsTestRouter(t *testing.T, pool *pgxpool.Pool) http.Handler {
	t.Helper()

	hasher := identityinfra.NewArgon2idHasher("test-pepper")
	issuer, err := identityinfra.NewJWTIssuer("test-jwt-secret-min-16-chars", "nebula-test")
	if err != nil {
		t.Fatal(err)
	}
	identitySvc, err := identityapp.New(identityapp.Config{
		Users:       identityinfra.NewPostgresUserRepo(pool),
		Sessions:    identityinfra.NewPostgresSessionRepo(pool),
		Memberships: identityinfra.NewPostgresMembershipRepo(pool),
		Hasher:      hasher,
		Tokens:      issuer,
		Refresh:     identityinfra.NewRefreshGenerator(),
		Audit:       noopAudit{},
		AccessTTL:   15 * time.Minute,
		RefreshTTL:  24 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	identityHandler := identityif.NewHandler(identitySvc, "", "", "", "http://localhost:3000")

	cfg := config.Config{
		Env: config.EnvDevelopment,
		Build: config.BuildConfig{
			RegistryURL:      "registry:5000",
			RegistryInsecure: true,
		},
		Runtime: config.RuntimeConfig{BaseDomain: "nebula.localhost"},
	}
	projectsSvc := projectsapp.New(projectsinfra.NewRepository(pool), nil, nil, nil, cfg)
	projectsHandler := projectsif.NewHandler(projectsSvc, nil, cfg.Runtime.BaseDomain, projectsif.HandlerOpts{})

	r := chi.NewRouter()
	r.Route("/api/v1", func(api chi.Router) {
		identityHandler.Mount(api)
		api.Group(func(authed chi.Router) {
			authed.Use(identityif.Authenticator(identitySvc))
			identityHandler.MountAuthenticated(authed)
			projectsHandler.Mount(authed)
		})
	})
	return r
}

func registerAndLogin(t *testing.T, router http.Handler, email, password string) string {
	t.Helper()
	regBody, _ := json.Marshal(map[string]string{
		"email": email, "password": password, "display_name": "Test",
	})
	reg := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBody))
	reg.Header.Set("Content-Type", "application/json")
	regRR := httptest.NewRecorder()
	router.ServeHTTP(regRR, reg)
	if regRR.Code != http.StatusCreated && regRR.Code != http.StatusConflict {
		t.Fatalf("register status=%d body=%s", regRR.Code, regRR.Body.String())
	}
	loginBody, _ := json.Marshal(map[string]string{"email": email, "password": password})
	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	login.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	router.ServeHTTP(loginRR, login)
	if loginRR.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", loginRR.Code, loginRR.Body.String())
	}
	var env struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginRR.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.AccessToken == "" {
		t.Fatal("empty access token")
	}
	return env.Data.AccessToken
}

type noopAudit struct{}

func (noopAudit) Record(context.Context, string, *uuid.UUID, map[string]any) {}
