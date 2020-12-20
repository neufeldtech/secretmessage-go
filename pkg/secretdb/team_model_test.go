package secretdb

import (
	"context"
	"database/sql"
	"log"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func Test_teamRepository_FindByID(t *testing.T) {

	teamID := "TABC123"
	accessToken := "xoxb-123"
	scope := "chat:write,chat:read"
	name := "myteam"
	paid := true

	type args struct {
		httpContext context.Context
		id          string
	}
	tests := []struct {
		name     string
		setup    func() (*sql.DB, sqlmock.Sqlmock)
		args     args
		want     TeamModel
		wantErr  bool
		teardown func(*testing.T, *sql.DB, sqlmock.Sqlmock)
	}{
		{
			name: "happy path",
			setup: func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					log.Fatal(err)
				}
				rows := mock.NewRows([]string{"id", "access_token", "scope", "name", "paid"}).AddRow(
					teamID,
					accessToken,
					scope,
					name,
					paid,
				)
				mock.ExpectQuery("SELECT id, access_token, scope, name, paid FROM teams WHERE id = \\$1").WithArgs(teamID).WillReturnRows(rows)
				return db, mock
			},
			args: args{
				httpContext: context.Background(),
				id:          teamID,
			},
			want: TeamModel{
				ID:          teamID,
				AccessToken: accessToken,
				Scope:       scope,
				Name:        name,
				Paid:        paid,
			},
			teardown: func(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) {
				assert.NoError(t, mock.ExpectationsWereMet())
				db.Close()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := tt.setup()
			r := &teamRepository{
				db: db,
			}
			got, err := r.FindByID(tt.args.httpContext, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("teamRepository.FindByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("teamRepository.FindByID() = %v, want %v", got, tt.want)
			}
			tt.teardown(t, db, mock)
		})
	}
}

func Test_teamRepository_Create(t *testing.T) {

	teamID := "TABC123"
	accessToken := "xoxb-123"
	scope := "chat:write,chat:read"
	name := "myteam"
	paid := true

	type args struct {
		httpContext context.Context
		team        *TeamModel
	}
	tests := []struct {
		name     string
		setup    func() (*sql.DB, sqlmock.Sqlmock)
		args     args
		wantErr  bool
		teardown func(*testing.T, *sql.DB, sqlmock.Sqlmock)
	}{
		{
			name: "happy path",
			setup: func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					log.Fatal(err)
				}
				stmt := "INSERT INTO teams \\(id, access_token, scope, name, paid\\) VALUES \\(\\$1, \\$2, \\$3, \\$4, \\$5\\)"
				mock.ExpectPrepare(stmt)
				mock.ExpectExec(stmt).WithArgs(teamID, accessToken, scope, name, paid).WillReturnResult(sqlmock.NewResult(1, 1))
				return db, mock
			},
			args: args{
				httpContext: context.Background(),
				team: &TeamModel{
					ID:          teamID,
					AccessToken: accessToken,
					Scope:       scope,
					Name:        name,
					Paid:        paid,
				},
			},
			teardown: func(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) {
				assert.NoError(t, mock.ExpectationsWereMet())
				db.Close()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := tt.setup()
			r := &teamRepository{
				db: db,
			}
			err := r.Create(tt.args.httpContext, tt.args.team)
			if (err != nil) != tt.wantErr {
				t.Errorf("teamRepository.Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.teardown(t, db, mock)
		})
	}
}

func Test_teamRepository_Update(t *testing.T) {

	teamID := "TABC123"
	accessToken := "xoxb-123"
	scope := "chat:write,chat:read"
	name := "myteam"
	paid := true

	type args struct {
		httpContext context.Context
		team        *TeamModel
	}
	tests := []struct {
		name     string
		setup    func() (*sql.DB, sqlmock.Sqlmock)
		args     args
		wantErr  bool
		teardown func(*testing.T, *sql.DB, sqlmock.Sqlmock)
	}{
		{
			name: "happy path",
			setup: func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New()
				if err != nil {
					log.Fatal(err)
				}
				stmt := "UPDATE teams SET id = \\$1, access_token = \\$2, scope = \\$3, name = \\$4, paid = \\$5 WHERE id = \\$1"
				mock.ExpectPrepare(stmt)
				mock.ExpectExec(stmt).WithArgs(teamID, accessToken, scope, name, paid).WillReturnResult(sqlmock.NewResult(1, 1))
				return db, mock
			},
			args: args{
				httpContext: context.Background(),
				team: &TeamModel{
					ID:          teamID,
					AccessToken: accessToken,
					Scope:       scope,
					Name:        name,
					Paid:        paid,
				},
			},
			teardown: func(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) {
				assert.NoError(t, mock.ExpectationsWereMet())
				db.Close()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := tt.setup()
			r := &teamRepository{
				db: db,
			}
			err := r.Update(tt.args.httpContext, tt.args.team)
			if (err != nil) != tt.wantErr {
				t.Errorf("teamRepository.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.teardown(t, db, mock)
		})
	}
}
