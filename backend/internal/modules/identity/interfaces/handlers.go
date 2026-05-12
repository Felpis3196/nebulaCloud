package interfaces

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/nebulacloud/nebula/internal/modules/identity/application"
	"github.com/nebulacloud/nebula/internal/platform/auth"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
	"github.com/nebulacloud/nebula/internal/platform/httpx"
)

// Handler exposes the Identity HTTP routes.
type Handler struct {
	svc *application.Service
}

// NewHandler returns a Handler bound to the given service.
func NewHandler(svc *application.Service) *Handler { return &Handler{svc: svc} }

// Mount installs the auth routes under r at "/auth/...".
func (h *Handler) Mount(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.register)
		r.Post("/login", h.login)
		r.Post("/refresh", h.refresh)
		r.Post("/logout", h.logout)
		r.Get("/github", h.githubOAuthStart)
		r.Get("/github/callback", h.githubOAuthCallback)
	})
}

// MountAuthenticated installs routes that require a valid bearer token.
func (h *Handler) MountAuthenticated(r chi.Router) {
	r.Get("/me", h.me)
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	user, err := h.svc.Register(r.Context(), application.RegisterCommand{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, toUserDTO(user))
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	pair, err := h.svc.Login(r.Context(), application.LoginCommand{
		Email:     req.Email,
		Password:  req.Password,
		IP:        clientIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, toTokenPairDTO(pair))
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	pair, err := h.svc.Refresh(r.Context(), application.RefreshCommand{
		RefreshToken: req.RefreshToken,
		IP:           clientIP(r),
		UserAgent:    r.UserAgent(),
	})
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, toTokenPairDTO(pair))
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	_ = httpx.DecodeJSON(r, &req) // body is optional

	if err := h.svc.Logout(r.Context(), application.LogoutCommand{RefreshToken: req.RefreshToken}); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	user, err := h.svc.Me(r.Context(), principal)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, toUserDTO(user))
}

func clientIP(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		if i := strings.IndexByte(v, ','); i > 0 {
			return strings.TrimSpace(v[:i])
		}
		return strings.TrimSpace(v)
	}
	if v := r.Header.Get("X-Real-Ip"); v != "" {
		return v
	}
	return r.RemoteAddr
}
