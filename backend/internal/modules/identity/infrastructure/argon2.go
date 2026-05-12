package infrastructure

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2idHasher implements domain.PasswordHasher using Argon2id with the
// recommended OWASP 2023 parameters. The encoded format follows the
// canonical "$argon2id$v=19$m=...,t=...,p=...$salt$hash" string so values
// can be migrated in/out of the platform.
type Argon2idHasher struct {
	pepper  string
	timeCost   uint32
	memoryKiB  uint32 // memory in KiB
	threads    uint8
	saltLen    uint32
	keyLen     uint32
}

// NewArgon2idHasher returns a hasher with safe defaults. The pepper is a
// process-wide secret combined with the password before hashing so a
// database leak alone is not enough to mount an offline attack.
func NewArgon2idHasher(pepper string) *Argon2idHasher {
	return &Argon2idHasher{
		pepper:    pepper,
		timeCost:  3,
		memoryKiB: 64 * 1024, // 64 MiB
		threads:   4,
		saltLen:   16,
		keyLen:    32,
	}
}

// Hash returns the encoded hash for the given plaintext.
func (h *Argon2idHasher) Hash(plain string) (string, error) {
	salt := make([]byte, h.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("argon2: read salt: %w", err)
	}
	digest := argon2.IDKey([]byte(h.pepper+plain), salt, h.timeCost, h.memoryKiB, h.threads, h.keyLen)
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.memoryKiB, h.timeCost, h.threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(digest),
	)
	return encoded, nil
}

// Verify reports whether the plaintext matches the encoded hash.
func (h *Argon2idHasher) Verify(plain, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("argon2: unsupported hash format")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("argon2: parse version: %w", err)
	}
	var memory, timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads); err != nil {
		return false, fmt.Errorf("argon2: parse params: %w", err)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("argon2: decode salt: %w", err)
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("argon2: decode hash: %w", err)
	}
	candidate := argon2.IDKey([]byte(h.pepper+plain), salt, timeCost, memory, threads, uint32(len(expected)))
	if subtle.ConstantTimeCompare(candidate, expected) == 1 {
		return true, nil
	}
	return false, nil
}
