package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

const RefreshCookieName = "refresh_token"

func GenerateRefreshTokenValue() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func HashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
