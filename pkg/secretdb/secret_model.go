package secretdb

import (
	"context"
	"database/sql"
	"time"
)

// SecretModel represent the secret model
type SecretModel struct {
	ID        string
	CreatedAt time.Time
	ExpiresAt time.Time
	Value     string
}

// secretRepository represent the secretRepository model
type secretRepository struct {
	db *sql.DB
}

type SecretRepository interface {
	Close()
	FindByID(httpContext context.Context, id string) (*SecretModel, error)
	Find(httpContext context.Context) ([]SecretModel, error)
	Create(httpContext context.Context, secret *SecretModel) error
	Update(httpContext context.Context, secret *SecretModel) error
	Delete(httpContext context.Context, id string) error
}

// NewSecretsRepository will create a variable that represent the Repository struct
func NewSecretsRepository(db *sql.DB) SecretRepository {
	return &secretRepository{db}
}

// Close attaches the provider and close the connection
func (r *secretRepository) Close() {
	r.db.Close()
}

// FindByID attaches the user repository and find data based on id
func (r *secretRepository) FindByID(httpContext context.Context, id string) (*SecretModel, error) {
	secret := new(SecretModel)

	ctx, cancel := context.WithTimeout(httpContext, 5*time.Second)
	defer cancel()

	err := r.db.QueryRowContext(ctx, "SELECT id, created_at, expires_at, value FROM secrets WHERE id = $1", id).Scan(
		&secret.ID,
		&secret.CreatedAt,
		&secret.ExpiresAt,
		&secret.Value,
	)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// Find attaches the user repository and find all data
func (r *secretRepository) Find(httpContext context.Context) ([]SecretModel, error) {
	//TODO IMPLEMENT APM CONTEXT ATTACHMENT

	secrets := []SecretModel{}

	ctx, cancel := context.WithTimeout(httpContext, 5*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, "SELECT id, created_at, expires_at, value FROM secrets")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		secret := SecretModel{}
		err = rows.Scan(
			&secret.ID,
			&secret.CreatedAt,
			&secret.ExpiresAt,
			&secret.Value,
		)

		if err != nil {
			return nil, err
		}
		secrets = append(secrets, secret)
	}

	return secrets, nil
}

// Create attaches the secret repository and creating the data
func (r *secretRepository) Create(httpContext context.Context, secret *SecretModel) error {
	ctx, cancel := context.WithTimeout(httpContext, 5*time.Second)
	defer cancel()

	query := "INSERT INTO secrets (id, created_at, expires_at, value) VALUES ($1, $2, $3, $4)"
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, secret.ID, secret.CreatedAt, secret.ExpiresAt, secret.Value)
	return err
}

// Update attaches the secret repository and update data based on id
func (r *secretRepository) Update(httpContext context.Context, secret *SecretModel) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := "UPDATE secrets SET id = $1, created_at = $2, expires_at = $3, value = $4 WHERE id = $1"
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, secret.ID, secret.CreatedAt, secret.ExpiresAt, secret.Value)
	return err
}

// Delete attaches the secret repository and delete data based on id
func (r *secretRepository) Delete(httpContext context.Context, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := "DELETE FROM secrets WHERE id = $1"
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, id)
	return err
}
