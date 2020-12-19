package secretdb

import (
	"context"
	"database/sql"
	"time"
)

// TeamModel represent the team model
type TeamModel struct {
	ID          string
	AccessToken string
	Scope       string
	Name        string
	Paid        bool
}

// teamRepository represent the teamRepository model
type teamRepository struct {
	db *sql.DB
}

type TeamRepository interface {
	Close()
	FindByID(httpContext context.Context, id string) (TeamModel, error)
	Find(httpContext context.Context) ([]TeamModel, error)
	Create(httpContext context.Context, team *TeamModel) error
	Update(httpContext context.Context, team *TeamModel) error
	Delete(httpContext context.Context, id string) error
}

// NewTeamsRepository will create a variable that represent the Repository struct
func NewTeamsRepository(db *sql.DB) TeamRepository {
	return &teamRepository{db}
}

// Close attaches the provider and close the connection
func (r *teamRepository) Close() {
	r.db.Close()
}

// FindByID attaches the user repository and find data based on id
func (r *teamRepository) FindByID(httpContext context.Context, id string) (TeamModel, error) {
	team := TeamModel{}

	ctx, cancel := context.WithTimeout(httpContext, 5*time.Second)
	defer cancel()

	err := r.db.QueryRowContext(ctx, "SELECT id, access_token, scope, name, paid FROM teams WHERE id = $1", id).Scan(
		&team.ID,
		&team.AccessToken,
		&team.Scope,
		&team.Name,
		&team.Paid,
	)
	if err != nil {
		return team, err
	}
	return team, nil
}

// Find attaches the user repository and find all data
func (r *teamRepository) Find(httpContext context.Context) ([]TeamModel, error) {
	//TODO IMPLEMENT APM CONTEXT ATTACHMENT

	teams := []TeamModel{}

	ctx, cancel := context.WithTimeout(httpContext, 5*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, "SELECT id, access_token, scope, name, paid FROM teams")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		team := TeamModel{}
		err = rows.Scan(
			&team.ID,
			&team.AccessToken,
			&team.Scope,
			&team.Name,
			&team.Paid,
		)

		if err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}

	return teams, nil
}

// Create attaches the team repository and creating the data
func (r *teamRepository) Create(httpContext context.Context, team *TeamModel) error {
	ctx, cancel := context.WithTimeout(httpContext, 5*time.Second)
	defer cancel()

	query := "INSERT INTO teams (id, access_token, scope, name, paid) VALUES ($1, $2, $3, $4, $5)"
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(
		ctx,
		team.ID,
		team.AccessToken,
		team.Scope,
		team.Name,
		team.Paid,
	)
	return err
}

// Update attaches the team repository and update data based on id
func (r *teamRepository) Update(httpContext context.Context, team *TeamModel) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := "UPDATE teams SET id = $1, access_token = $2, scope = $3, name = $4, paid = $5 WHERE id = $1"
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(
		ctx,
		team.ID,
		team.AccessToken,
		team.Scope,
		team.Name,
		team.Paid,
	)
	return err
}

// Delete attaches the team repository and delete data based on id
func (r *teamRepository) Delete(httpContext context.Context, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := "DELETE FROM teams WHERE id = $1"
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, id)
	return err
}
