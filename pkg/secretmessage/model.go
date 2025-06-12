package secretmessage

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

type Secret struct {
	gorm.Model
	ID        string
	ExpiresAt time.Time
	Value     string
}

type SecretOption func(*Secret) *Secret

type Team struct {
	gorm.Model
	ID          string
	AccessToken string
	Scope       string
	Name        string
	Paid        sql.NullBool `gorm:"default:false"`
}

func WithExpiryDate(expiryDate time.Time) SecretOption {
	return func(s *Secret) *Secret {
		s.ExpiresAt = expiryDate
		return s
	}
}

func NewSecret(id string, value string, opts ...SecretOption) *Secret {
	secret := &Secret{
		ID:    id,
		Value: value,
	}

	for _, opt := range opts {
		opt(secret)
	}

	if secret.ExpiresAt.IsZero() {
		// Default to 7 days expiry if not provided
		secret.ExpiresAt = time.Now().AddDate(0, 0, 7)
	}
	// if expiry date is more than 30 days in the future, set it to 30 days
	if secret.ExpiresAt.After(time.Now().AddDate(0, 0, 30)) {
		secret.ExpiresAt = time.Now().AddDate(0, 0, 30)
	}

	// If expiry date is in the past, set it to now
	if secret.ExpiresAt.Before(time.Now()) {
		secret.ExpiresAt = time.Now()
	}
	return secret
}
