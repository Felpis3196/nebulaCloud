package interfaces

import (
	"encoding/json"
	"net/http"

	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
	"github.com/nebulacloud/nebula/internal/platform/httpx"
)

// StubGithubInstallationLink reserves POST /integrations/github/installation for Phase 3.
// SetProjectGitHubInstallation exists on the service for verified callbacks.
func (h *Handler) StubGithubInstallationLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.Error(w, platformerrors.Validation("method not allowed"))
		return
	}
	if _, ok := principal(r); !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(httpx.Response{
		Data: map[string]string{
			"status":  "not_implemented",
			"message": "GitHub App installation linking requires the Phase 3 OAuth callback. Service.SetProjectGitHubInstallation is available for verified flows.",
		},
	})
}
