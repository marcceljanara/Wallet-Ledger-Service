package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPasswordHashing(t *testing.T) {
	password := "SecretPass123"
	hash, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	assert.True(t, CheckPassword(password, hash))
	assert.False(t, CheckPassword("WrongPass", hash))
}
