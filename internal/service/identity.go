package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func generateSubject() (string, error) {
	randomBytes := make([]byte, 24)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate subject: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}
