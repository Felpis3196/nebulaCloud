package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/nebulacloud/nebula/internal/modules/identity/domain"
	"github.com/nebulacloud/nebula/internal/platform/auth"
)

// JWTIssuer implements domain.AccessTokenIssuer using HS256 (rotatable to
// RS256 by swapping the underlying key + signing method).
type JWTIssuer struct {
	secret []byte
	issuer string
	method jwt.SigningMethod
}

// NewJWTIssuer constructs a JWTIssuer. Returns an error if secret is empty.
func NewJWTIssuer(secret, issuer string) (*JWTIssuer, error) {
	if len(secret) < 16 {
		return nil, errors.New("jwt: secret too short (min 16 chars)")
	}
	return &JWTIssuer{
		secret: []byte(secret),
		issuer: issuer,
		method: jwt.SigningMethodHS256,
	}, nil
}

// nebulaClaims is the on-the-wire claim set. Keeping it private prevents
// callers from depending on JWT-library internals.
type nebulaClaims struct {
	jwt.RegisteredClaims
	Email          string    `json:"email"`
	OrganizationID string    `json:"org,omitempty"`
	Role           auth.Role `json:"role"`
	SessionID      string    `json:"sid,omitempty"`
}

// Issue serialises the supplied claims into a signed JWT.
func (j *JWTIssuer) Issue(_ context.Context, claims domain.AccessClaims) (string, error) {
	tok := jwt.NewWithClaims(j.method, nebulaClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   claims.Subject,
			ID:        claims.TokenID,
			IssuedAt:  jwt.NewNumericDate(claims.IssuedAt),
			ExpiresAt: jwt.NewNumericDate(claims.ExpiresAt),
			NotBefore: jwt.NewNumericDate(claims.IssuedAt.Add(-1 * time.Second)),
		},
		Email:          claims.Email,
		OrganizationID: claims.OrganizationID,
		Role:           claims.Role,
		SessionID:      claims.SessionID,
	})
	signed, err := tok.SignedString(j.secret)
	if err != nil {
		return "", fmt.Errorf("jwt: sign: %w", err)
	}
	return signed, nil
}

// Verify validates a token and returns the parsed claims.
func (j *JWTIssuer) Verify(_ context.Context, token string) (domain.AccessClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &nebulaClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != j.method.Alg() {
			return nil, fmt.Errorf("jwt: unexpected alg %q", t.Method.Alg())
		}
		return j.secret, nil
	}, jwt.WithIssuer(j.issuer), jwt.WithExpirationRequired(), jwt.WithLeeway(5*time.Second))
	if err != nil {
		return domain.AccessClaims{}, err
	}
	claims, ok := parsed.Claims.(*nebulaClaims)
	if !ok || !parsed.Valid {
		return domain.AccessClaims{}, errors.New("jwt: invalid claims")
	}
	if !claims.Role.IsValid() {
		return domain.AccessClaims{}, errors.New("jwt: invalid role")
	}
	return domain.AccessClaims{
		Subject:        claims.Subject,
		Email:          claims.Email,
		OrganizationID: claims.OrganizationID,
		Role:           claims.Role,
		SessionID:      claims.SessionID,
		TokenID:        claims.ID,
		IssuedAt:       claims.IssuedAt.Time,
		ExpiresAt:      claims.ExpiresAt.Time,
	}, nil
}
