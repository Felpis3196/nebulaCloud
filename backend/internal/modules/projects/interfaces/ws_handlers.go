package interfaces

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/nebulacloud/nebula/internal/platform/auth"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
	"github.com/nebulacloud/nebula/internal/platform/httpx"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // browser clients; CORS enforced at HTTP layer for API
	},
}

// streamLogs upgrades to WebSocket and fans out Redis pub/sub lines for a deployment.
// Query: deployment_id (required), token (optional JWT if Authorization header absent).
func (h *Handler) streamLogs(w http.ResponseWriter, r *http.Request) {
	pr, err := h.wsPrincipal(r)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	sid, ok := parseUUID(chi.URLParam(r, "serviceID"), w)
	if !ok {
		return
	}
	depRaw := strings.TrimSpace(r.URL.Query().Get("deployment_id"))
	if depRaw == "" {
		httpx.Error(w, platformerrors.Validation("deployment_id query param is required"))
		return
	}
	depID, err := uuid.Parse(depRaw)
	if err != nil {
		httpx.Error(w, platformerrors.Validation("invalid deployment_id"))
		return
	}
	dj, err := h.svc.GetDeployment(r.Context(), pr, depID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	if dj.ServiceID != sid {
		httpx.Error(w, platformerrors.Forbidden("deployment does not belong to service"))
		return
	}
	_ = pr

	if h.logSub == nil {
		httpx.Error(w, platformerrors.Internal("log stream unavailable"))
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	lines, err := h.logSub.Subscribe(ctx, depID.String())
	if err != nil {
		_ = conn.WriteJSON(map[string]string{"error": err.Error()})
		return
	}

	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				return
			}
			if err := conn.WriteJSON(line); err != nil {
				return
			}
		case <-ping.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *Handler) wsPrincipal(r *http.Request) (auth.Principal, error) {
	if pr, ok := auth.PrincipalFromContext(r.Context()); ok {
		return pr, nil
	}
	if h.verifyToken == nil {
		return auth.Principal{}, platformerrors.Unauthorized("not authenticated")
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		authz := r.Header.Get("Authorization")
		if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			token = strings.TrimSpace(authz[7:])
		}
	}
	if token == "" {
		return auth.Principal{}, platformerrors.Unauthorized("missing token")
	}
	return h.verifyToken(r.Context(), token)
}
