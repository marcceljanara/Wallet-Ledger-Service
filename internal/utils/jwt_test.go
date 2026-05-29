package utils

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestJWTGenerationAndParsing(t *testing.T) {
	secret := "my-secret-key-123"
	userID := uuid.New()
	email := "test@example.com"
	role := "USER"
	expiration := 10 * time.Minute

	token, err := GenerateToken(userID, email, role, secret, expiration)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := ParseToken(token, secret)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
}

func TestJWTExpiredToken(t *testing.T) {
	secret := "my-secret-key-123"
	userID := uuid.New()
	email := "test@example.com"
	role := "USER"
	expiration := -1 * time.Minute

	token, err := GenerateToken(userID, email, role, secret, expiration)
	assert.NoError(t, err)

	_, err = ParseToken(token, secret)
	assert.Error(t, err)
}

func TestJWTInvalidSecret(t *testing.T) {
	secret := "my-secret-key-123"
	userID := uuid.New()
	email := "test@example.com"
	role := "USER"
	expiration := 10 * time.Minute

	token, err := GenerateToken(userID, email, role, secret, expiration)
	assert.NoError(t, err)

	_, err = ParseToken(token, "wrong-secret")
	assert.Error(t, err)
}
