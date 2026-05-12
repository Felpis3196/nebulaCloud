package infrastructure

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

// RefreshGenerator implements domain.RefreshTokenGenerator. It produces a
// 256-bit URL-safe opaque token and exposes its sha256 hash for storage.
//
// Storing the hash (not the token) means a database compromise never leaks
// usable refresh tokens.
type RefreshGenerator struct{}

// NewRefreshGenerator returns a RefreshGenerator.
func NewRefreshGenerator() *RefreshGenerator { return &RefreshGenerator{} }

// Generate returns (raw, hash, error).
func (g *RefreshGenerator) Generate() (string, []byte, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, fmt.Errorf("refresh: read random: %w", err)
	}
	raw := base64.RawURLEncoding.EncodeToString(buf)
	hash, err := g.Hash(raw)
	if err != nil {
		return "", nil, err
	}
	return raw, hash, nil
}

// Hash returns the sha256 of the raw token.
func (g *RefreshGenerator) Hash(token string) ([]byte, error) {
	if token == "" {
		return nil, errors.New("refresh: empty token")
	}
	sum := sha256.Sum256([]byte(token))
	return sum[:], nil
}
