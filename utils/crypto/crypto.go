package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func GenerateSecureRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

func GenerateOAuth2Credentials(clientIDLength, clientSecretLength int) (string, string, error) {
	// Set defaults if zero values are passed
	if clientIDLength <= 0 {
		clientIDLength = 32
	}
	if clientSecretLength <= 0 {
		clientSecretLength = 64
	}

	// Generate Client ID (hex encoded)
	clientIDBytes := make([]byte, clientIDLength/2) // Hex encoding doubles the length
	_, err := rand.Read(clientIDBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate client ID: %w", err)
	}
	clientID := hex.EncodeToString(clientIDBytes)

	// Generate Client Secret (base64 URL-safe encoded)
	secretBytesLength := clientSecretLength * 3 / 4 // Base64 encoding increases length by ~33%
	secretBytes := make([]byte, secretBytesLength)
	_, err = rand.Read(secretBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate client secret: %w", err)
	}

	clientSecret := base64.URLEncoding.EncodeToString(secretBytes)
	clientSecret = strings.TrimRight(clientSecret, "=") // Remove padding

	return clientID, clientSecret, nil
}

func HashPassword(password string) string {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hashedPassword)
}

func ComparePassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
