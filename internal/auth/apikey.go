package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// GenerateAPIKey generates a new API key with the given prefix
func GenerateAPIKey(prefix string, length int) (string, error) {
	// Generate random bytes
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Convert to hex string
	randomPart := hex.EncodeToString(bytes)

	// Combine prefix with random part
	apiKey := prefix + randomPart[:length]

	return apiKey, nil
}

// HashAPIKey creates a SHA-256 hash of the API key
func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

// ExtractKeyPrefix extracts the prefix from an API key (first 8 characters)
func ExtractKeyPrefix(apiKey string) string {
	if len(apiKey) < 8 {
		return apiKey
	}
	return apiKey[:8]
}

// ValidateAPIKeyFormat validates the format of an API key
func ValidateAPIKeyFormat(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key is empty")
	}

	if len(apiKey) < 16 {
		return fmt.Errorf("API key too short")
	}

	// Check if it has a valid prefix
	if !strings.HasPrefix(apiKey, "sk_") && !strings.HasPrefix(apiKey, "ak_") {
		return fmt.Errorf("invalid API key prefix")
	}

	return nil
}

// CompareAPIKey compares a raw API key with a hashed version
func CompareAPIKey(rawKey, hashedKey string) bool {
	return HashAPIKey(rawKey) == hashedKey
}