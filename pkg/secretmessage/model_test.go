package secretmessage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSecret_DefaultExpiry(t *testing.T) {
	id := "abc123"
	value := "mysecret"
	secret := NewSecret(id, value)

	assert.Equal(t, id, secret.ID)
	assert.Equal(t, value, secret.Value)
	assert.WithinDuration(t, time.Now().AddDate(0, 0, 7), secret.ExpiresAt, time.Second*2)
}

func TestNewSecret_WithExpiryDate(t *testing.T) {
	id := "abc123"
	value := "mysecret"
	expiry := time.Now().AddDate(0, 0, 3)
	secret := NewSecret(id, value, WithExpiryDate(expiry))

	assert.Equal(t, id, secret.ID)
	assert.Equal(t, value, secret.Value)
	assert.WithinDuration(t, expiry, secret.ExpiresAt, time.Second*2)
}

func TestNewSecret_ExpiryMoreThan30Days(t *testing.T) {
	id := "abc123"
	value := "mysecret"
	expiry := time.Now().AddDate(0, 0, 40)
	secret := NewSecret(id, value, WithExpiryDate(expiry))

	maxExpiry := time.Now().AddDate(0, 0, 30)
	assert.WithinDuration(t, maxExpiry, secret.ExpiresAt, time.Second*2)
}

func TestNewSecret_ExpiryInPast(t *testing.T) {
	id := "abc123"
	value := "mysecret"
	expiry := time.Now().Add(-time.Hour * 24)
	secret := NewSecret(id, value, WithExpiryDate(expiry))

	assert.WithinDuration(t, time.Now(), secret.ExpiresAt, time.Second*2)
}
