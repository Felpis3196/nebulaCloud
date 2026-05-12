package httpx

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// routePattern returns the chi route template (e.g. "/projects/{id}") if
// the request is being served by a chi router; otherwise it falls back to
// the raw URL path. Used by metrics middleware to keep cardinality low.
func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if p := rctx.RoutePattern(); p != "" {
			return p
		}
	}
	return r.URL.Path
}
