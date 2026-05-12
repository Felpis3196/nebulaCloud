package interfaces

import (
	"net/http"
	"strings"

	"github.com/nebulacloud/nebula/internal/modules/identity/application"
	"github.com/nebulacloud/nebula/internal/platform/auth"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
	"github.com/nebulacloud/nebula/internal/platform/httpx"
)

// Authenticator returns middleware that requires a valid bearer token.
//
// The token is parsed and verified by the Identity application service; on
// success the resulting auth.Principal is stored on the request context so
// downstream handlers can pick it up via auth.PrincipalFromContext.
func Authenticator(svc *application.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				httpx.Error(w, platformerrors.Unauthorized("missing bearer token"))
				return
			}
			principal, err := svc.VerifyAccessToken(r.Context(), token)
			if err != nil {
				httpx.Error(w, err)
				return
			}
			ctx := auth.WithPrincipal(r.Context(), principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that rejects principals whose role is
// below the required minimum.
//
// Authenticator MUST be installed earlier in the chain.
func RequireRole(required auth.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := auth.PrincipalFromContext(r.Context())
			if !ok {
				httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
				return
			}
			if !principal.HasRole(required) {
				httpx.Error(w, platformerrors.Forbidden("insufficient privileges"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}
