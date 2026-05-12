package interfaces

import (
	"encoding/json"
	"net/http"

	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
	"github.com/nebulacloud/nebula/internal/platform/httpx"
)

func (h *Handler) githubOAuthStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.Error(w, platformerrors.Validation("method not allowed"))
		return
	}
	writeGitHubNotImplemented(w)
}

func (h *Handler) githubOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.Error(w, platformerrors.Validation("method not allowed"))
		return
	}
	writeGitHubNotImplemented(w)
}

func writeGitHubNotImplemented(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(httpx.Response{
		Data: map[string]string{
			"status":  "not_implemented",
			"message": "GitHub OAuth for NebulaCloud is planned for Phase 3 (App + user token exchange).",
		},
	})
}
