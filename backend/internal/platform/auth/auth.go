// Package auth defines the platform's identity primitives: roles, claims,
// and the AuthContext that flows through every authenticated request.
//
// Concrete JWT signing + password hashing implementations live in
// modules/identity/infrastructure so this package stays free of
// crypto-library dependencies.
package auth

import "context"

// Role enumerates the RBAC roles supported by NebulaCloud.
type Role string

const (
	RoleAdmin     Role = "admin"
	RoleDeveloper Role = "developer"
	RoleViewer    Role = "viewer"
)

// Rank assigns an ordering to roles so middleware can write predicates like
// `role.AtLeast(RoleDeveloper)`.
func (r Role) Rank() int {
	switch r {
	case RoleAdmin:
		return 30
	case RoleDeveloper:
		return 20
	case RoleViewer:
		return 10
	default:
		return 0
	}
}

// AtLeast reports whether r is at least as privileged as required.
func (r Role) AtLeast(required Role) bool { return r.Rank() >= required.Rank() }

// IsValid returns true when r is a known role.
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleDeveloper, RoleViewer:
		return true
	}
	return false
}

// Principal is the authenticated identity attached to a request context.
type Principal struct {
	UserID         string
	Email          string
	OrganizationID string
	Role           Role
	SessionID      string
	TokenID        string // jti — for revocation lookups
}

// HasRole returns true if the principal has at least the required role.
func (p Principal) HasRole(required Role) bool { return p.Role.AtLeast(required) }

type ctxKey struct{}

// WithPrincipal attaches a principal to ctx.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// PrincipalFromContext retrieves the principal attached to ctx.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}
