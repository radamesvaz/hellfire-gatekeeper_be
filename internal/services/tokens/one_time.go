package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// OneTimeTokenManager abstracts one-time token generation/hashing.
type OneTimeTokenManager interface {
	Generate() (plain string, hash string, err error)
	Hash(rawToken string) string
}

// SHA256OneTimeTokenManager implements secure random token + SHA-256 hashing.
type SHA256OneTimeTokenManager struct {
	tokenBytes int
}

func NewSHA256OneTimeTokenManager(tokenBytes int) *SHA256OneTimeTokenManager {
	if tokenBytes <= 0 {
		tokenBytes = 32
	}
	return &SHA256OneTimeTokenManager{tokenBytes: tokenBytes}
}

func (m *SHA256OneTimeTokenManager) Generate() (string, string, error) {
	raw := make([]byte, m.tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	plain := strings.ToUpper(hex.EncodeToString(raw))
	return plain, m.Hash(plain), nil
}

func (m *SHA256OneTimeTokenManager) Hash(rawToken string) string {
	normalized := strings.TrimSpace(strings.ToUpper(rawToken))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
