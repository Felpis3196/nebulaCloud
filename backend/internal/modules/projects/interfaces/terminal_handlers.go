package interfaces

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/nebulacloud/nebula/internal/platform/auth"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
	"github.com/nebulacloud/nebula/internal/platform/httpx"
	"github.com/nebulacloud/nebula/internal/platform/terminal"
)

const terminalSessionTimeout = 30 * time.Minute

type terminalResizeMsg struct {
	Type string `json:"type"`
	Cols uint   `json:"cols"`
	Rows uint   `json:"rows"`
}

// serviceTerminal upgrades to WebSocket and attaches docker exec to the service container.
func (h *Handler) serviceTerminal(w http.ResponseWriter, r *http.Request) {
	if !h.terminalOn {
		httpx.Error(w, platformerrors.Forbidden("web terminal is disabled"))
		return
	}
	pr, err := h.wsPrincipal(r)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	sid, ok := parseUUID(chi.URLParam(r, "serviceID"), w)
	if !ok {
		return
	}
	if err := h.svc.AuthorizeServiceDeveloper(r.Context(), pr, sid); err != nil {
		httpx.Error(w, err)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(r.Context(), terminalSessionTimeout)
	defer cancel()

	sess, err := terminal.Open(ctx, sid.String())
	if err != nil {
		_ = conn.WriteJSON(map[string]string{"error": err.Error()})
		return
	}
	defer sess.Close()

	h.auditTerminal(r.Context(), pr, sid, "terminal.connect")
	defer h.auditTerminal(context.Background(), pr, sid, "terminal.disconnect")

	attach, err := sess.Attach(ctx)
	if err != nil {
		_ = conn.WriteJSON(map[string]string{"error": err.Error()})
		return
	}
	defer attach.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			n, err := attach.Reader.Read(buf)
			if n > 0 {
				if werr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			cancel()
			break
		}
		var resize terminalResizeMsg
		if json.Unmarshal(msg, &resize) == nil && resize.Type == "resize" && resize.Cols > 0 && resize.Rows > 0 {
			_ = sess.Resize(ctx, resize.Cols, resize.Rows)
			continue
		}
		if _, err := attach.Conn.Write(msg); err != nil {
			cancel()
			break
		}
	}
	wg.Wait()
}

func (h *Handler) auditTerminal(ctx context.Context, actor auth.Principal, serviceID uuid.UUID, action string) {
	if h.auditFn == nil {
		return
	}
	h.auditFn(ctx, actor, serviceID, action)
}
