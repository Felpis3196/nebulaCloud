package interfaces

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	projectsinfra "github.com/nebulacloud/nebula/internal/modules/projects/infrastructure"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
	"github.com/nebulacloud/nebula/internal/platform/httpx"
	"github.com/nebulacloud/nebula/internal/platform/logger"
)

// GithubWebhook verifies the X-Hub-Signature-256 payload and triggers builds on pushes.
func (h *Handler) GithubWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.Error(w, platformerrors.Validation("method not allowed"))
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		httpx.Error(w, platformerrors.Validation("read body"))
		return
	}

	secret := h.svc.WebhookSecret()
	env := h.svc.Env()
	if env.IsProduction() && strings.TrimSpace(secret) == "" {
		httpx.Error(w, platformerrors.Unauthorized("webhook disabled"))
		return
	}
	if strings.TrimSpace(secret) != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		if sig == "" {
			httpx.Error(w, platformerrors.Unauthorized("missing signature"))
			return
		}
		if !verifyGithubHMAC(body, sig, secret) {
			logger.FromContext(r.Context()).Warn("github_webhook.signature_mismatch")
			httpx.Error(w, platformerrors.Unauthorized("invalid signature"))
			return
		}
	}

	event := strings.TrimSpace(strings.ToLower(r.Header.Get("X-GitHub-Event")))
	switch event {
	case "ping":
		httpx.NoContent(w)
		return
	case "push":
	default:
		httpx.OK(w, map[string]string{"status": "ignored", "reason": event})
		return
	}

	var envelope struct {
		Ref    string `json:"ref"`
		After  string `json:"after"`
		HeadCommit struct {
			ID      string `json:"id"`
			Message string `json:"message"`
		} `json:"head_commit"`
		Installation *struct {
			ID int64 `json:"id"`
		} `json:"installation"`
		Repo struct {
			CloneURL string `json:"clone_url"`
			GitURL   string `json:"git_url"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		httpx.Error(w, platformerrors.Validation("invalid JSON"))
		return
	}
	rawURL := strings.TrimSpace(envelope.Repo.CloneURL)
	if rawURL == "" {
		rawURL = strings.TrimSpace(envelope.Repo.GitURL)
	}
	norm := projectsinfra.NormalizeRepoURL(rawURL)
	sha := strings.TrimSpace(envelope.After)
	if noopPushSHA(sha) {
		sha = strings.TrimSpace(envelope.HeadCommit.ID)
	}
	if noopPushSHA(sha) {
		httpx.OK(w, map[string]any{"status": "ignored", "reason": "noop_commit"})
		return
	}
	msg := strings.TrimSpace(envelope.HeadCommit.Message)
	ref := envelope.Ref

	var instPtr *int64
	if envelope.Installation != nil && envelope.Installation.ID != 0 {
		id := envelope.Installation.ID
		instPtr = &id
	}

	n := h.svc.DispatchGitPush(r.Context(), norm, sha, msg, ref, instPtr)
	httpx.OK(w, map[string]any{"status": "ok", "enqueued_services": n})
}

// noopPushSHA is true for missing commits (branch delete) or GitHub's all-zero placeholder.
func noopPushSHA(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return true
	}
	for _, c := range s {
		if c != '0' {
			return false
		}
	}
	return true
}

func verifyGithubHMAC(payload []byte, hexHeader string, secret string) bool {
	if !strings.HasPrefix(hexHeader, "sha256=") {
		return false
	}
	supplied := hexHeader[len("sha256="):]
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(supplied)), []byte(strings.ToLower(expected)))
}
