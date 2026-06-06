package interfaces

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
	"github.com/nebulacloud/nebula/internal/platform/httpx"
)

const (
	githubOAuthStateCookie    = "nebula_github_oauth_state"
	githubOAuthReturnCookie   = "nebula_github_oauth_return"
	githubOAuthTokenCookie    = "nebula_github_oauth_token"
	githubOAuthCookieMaxAge   = 600
	githubOAuthTokenMaxAge    = 3600
)

func (h *Handler) githubOAuthStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.Error(w, platformerrors.Validation("method not allowed"))
		return
	}
	if strings.TrimSpace(h.ghClientID) == "" || strings.TrimSpace(h.ghRedirect) == "" {
		writeGitHubNotImplemented(w)
		return
	}
	state, err := randomOAuthState()
	if err != nil {
		httpx.Error(w, platformerrors.Internal("oauth state"))
		return
	}
	returnTo := sanitizeOAuthReturn(h.appURL, r.URL.Query().Get("return_to"))
	http.SetCookie(w, &http.Cookie{
		Name:     githubOAuthStateCookie,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   githubOAuthCookieMaxAge,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     githubOAuthReturnCookie,
		Value:    returnTo,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   githubOAuthCookieMaxAge,
	})
	q := url.Values{}
	q.Set("client_id", h.ghClientID)
	q.Set("redirect_uri", h.ghRedirect)
	q.Set("scope", "read:user repo")
	q.Set("state", state)
	http.Redirect(w, r, "https://github.com/login/oauth/authorize?"+q.Encode(), http.StatusFound)
}

func (h *Handler) githubOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.Error(w, platformerrors.Validation("method not allowed"))
		return
	}
	if strings.TrimSpace(h.ghClientID) == "" {
		writeGitHubNotImplemented(w)
		return
	}
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	cookie, err := r.Cookie(githubOAuthStateCookie)
	if err != nil || cookie.Value != state || state == "" {
		redirectOAuthError(w, h.appURL, "invalid_state")
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		redirectOAuthError(w, h.appURL, "missing_code")
		return
	}
	token, err := exchangeGitHubCode(r, h.ghClientID, h.ghClientSecret, h.ghRedirect, code)
	if err != nil {
		redirectOAuthError(w, h.appURL, "token_exchange")
		return
	}
	returnTo := h.appURL + "/auth/github/callback"
	if rc, err := r.Cookie(githubOAuthReturnCookie); err == nil && rc.Value != "" {
		returnTo = sanitizeOAuthReturn(h.appURL, rc.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     githubOAuthTokenCookie,
		Value:    token.AccessToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   githubOAuthTokenMaxAge,
	})
	u, _ := url.Parse(returnTo)
	q := u.Query()
	q.Set("github", "connected")
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (h *Handler) githubOAuthRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.Error(w, platformerrors.Validation("method not allowed"))
		return
	}
	if strings.TrimSpace(h.ghClientID) == "" {
		writeGitHubNotImplemented(w)
		return
	}
	cookie, err := r.Cookie(githubOAuthTokenCookie)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		httpx.Error(w, platformerrors.Unauthorized("github not connected"))
		return
	}
	repos, err := listGitHubRepos(r, cookie.Value)
	if err != nil {
		httpx.Error(w, platformerrors.Internal("list github repos").WithCause(err))
		return
	}
	httpx.OK(w, repos)
}

type githubTokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type githubRepoDTO struct {
	FullName      string `json:"full_name"`
	HTMLURL       string `json:"html_url"`
	CloneURL      string `json:"clone_url"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
}

func exchangeGitHubCode(r *http.Request, clientID, clientSecret, redirectURI, code string) (githubTokenResp, error) {
	body := url.Values{}
	body.Set("client_id", clientID)
	body.Set("client_secret", clientSecret)
	body.Set("code", code)
	body.Set("redirect_uri", redirectURI)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(body.Encode()))
	if err != nil {
		return githubTokenResp{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return githubTokenResp{}, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode/100 != 2 {
		return githubTokenResp{}, fmt.Errorf("github status %d: %s", res.StatusCode, string(raw))
	}
	var out githubTokenResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return githubTokenResp{}, err
	}
	if out.AccessToken == "" {
		return githubTokenResp{}, fmt.Errorf("empty access_token")
	}
	return out, nil
}

func listGitHubRepos(r *http.Request, accessToken string) ([]githubRepoDTO, error) {
	req, err := http.NewRequestWithContext(
		r.Context(),
		http.MethodGet,
		"https://api.github.com/user/repos?per_page=100&sort=updated",
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode/100 != 2 {
		return nil, fmt.Errorf("github api status %d: %s", res.StatusCode, string(raw))
	}
	var payload []struct {
		FullName      string `json:"full_name"`
		HTMLURL       string `json:"html_url"`
		CloneURL      string `json:"clone_url"`
		DefaultBranch string `json:"default_branch"`
		Private       bool   `json:"private"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	out := make([]githubRepoDTO, 0, len(payload))
	for _, p := range payload {
		if p.FullName == "" || p.HTMLURL == "" {
			continue
		}
		clone := p.CloneURL
		if clone == "" {
			clone = p.HTMLURL + ".git"
		}
		branch := p.DefaultBranch
		if branch == "" {
			branch = "main"
		}
		out = append(out, githubRepoDTO{
			FullName:      p.FullName,
			HTMLURL:       p.HTMLURL,
			CloneURL:      clone,
			DefaultBranch: branch,
			Private:       p.Private,
		})
	}
	return out, nil
}

func sanitizeOAuthReturn(appURL, raw string) string {
	appURL = strings.TrimRight(strings.TrimSpace(appURL), "/")
	if appURL == "" {
		appURL = "http://localhost:3000"
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return appURL + "/auth/github/callback"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return appURL + "/auth/github/callback"
	}
	base, err := url.Parse(appURL)
	if err != nil {
		return appURL + "/auth/github/callback"
	}
	if !strings.EqualFold(u.Scheme, base.Scheme) || !strings.EqualFold(u.Host, base.Host) {
		return appURL + "/auth/github/callback"
	}
	return strings.TrimRight(raw, "/")
}

func redirectOAuthError(w http.ResponseWriter, appURL, code string) {
	target := sanitizeOAuthReturn(appURL, "") + "?github=error&reason=" + url.QueryEscape(code)
	w.Header().Set("Location", target)
	w.WriteHeader(http.StatusFound)
}

func randomOAuthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func writeGitHubNotImplemented(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(httpx.Response{
		Data: map[string]string{
			"status":  "not_implemented",
			"message": "Set NEBULA_GITHUB_APP_CLIENT_ID and NEBULA_GITHUB_OAUTH_REDIRECT_URL to enable GitHub OAuth.",
		},
	})
}
