package util

import (
	"crypto/rand"
	"fmt"

	"github.com/google/uuid"
)

// GenerateUUID generates a random UUID string
func GenerateUUID() string {
	return uuid.New().String()
}

// SimpleUUID generates a simple random UUID (fallback)
func SimpleUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to simple random if crypto fails
		return fmt.Sprintf("%x", b)
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
