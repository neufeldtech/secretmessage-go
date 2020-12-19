package secretdb

import (
	"context"
	"database/sql"
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func Test_secretRepository_FindByID(t *testing.T) {
	secretIDHashed := "000c285457fc971f862a79b786476c78812c8897063c6fa9c045f579a3b2d63f"
	secretPayload := "30303030303030303030303029c9922a9be75ba2e6be5afd32d19387baea51fa577c0c51dc9809a54adb9085490f109237d15a3262a585"
	createdAtTime := time.Unix(123, 0)
	expiresAtTime := createdAtTime.Add(time.Hour)

	type args struct {
		httpContext context.Context
		id          string
	}
	tests := []struct {
		name     string
		setup    func() (*sql.DB, sqlmock.Sqlmock)
		args     args
		want     SecretModel
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
				rows := mock.NewRows([]string{"id", "created_at", "expires_at", "value"}).AddRow(
					secretIDHashed,
					createdAtTime,
					expiresAtTime,
					secretPayload,
				)
				mock.ExpectQuery("SELECT id, created_at, expires_at, value FROM secrets WHERE id = \\$1").WithArgs(secretIDHashed).WillReturnRows(rows)
				return db, mock
			},
			args: args{
				httpContext: context.Background(),
				id:          secretIDHashed,
			},
			want: SecretModel{
				ID:        secretIDHashed,
				CreatedAt: createdAtTime,
				ExpiresAt: expiresAtTime,
				Value:     secretPayload,
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
			r := &secretRepository{
				db: db,
			}
			got, err := r.FindByID(tt.args.httpContext, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("secretRepository.FindByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("secretRepository.FindByID() = %v, want %v", got, tt.want)
			}
			tt.teardown(t, db, mock)
		})
	}
}
func Test_secretRepository_Create(t *testing.T) {
	secretIDHashed := "000c285457fc971f862a79b786476c78812c8897063c6fa9c045f579a3b2d63f"
	secretPayload := "30303030303030303030303029c9922a9be75ba2e6be5afd32d19387baea51fa577c0c51dc9809a54adb9085490f109237d15a3262a585"
	createdAtTime := time.Unix(123, 0)
	expiresAtTime := createdAtTime.Add(time.Hour)

	type args struct {
		httpContext context.Context
		secret      *SecretModel
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

				stmt := "INSERT INTO secrets \\(id, created_at, expires_at, value\\) VALUES \\(\\$1, \\$2, \\$3, \\$4\\)"
				mock.ExpectPrepare(stmt)
				mock.ExpectExec(stmt).WithArgs(secretIDHashed, createdAtTime, expiresAtTime, secretPayload).WillReturnResult(sqlmock.NewResult(1, 1))
				return db, mock
			},
			args: args{
				httpContext: context.Background(),
				secret: &SecretModel{
					ID:        secretIDHashed,
					CreatedAt: createdAtTime,
					ExpiresAt: expiresAtTime,
					Value:     secretPayload,
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
			r := &secretRepository{
				db: db,
			}
			err := r.Create(tt.args.httpContext, tt.args.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("secretRepository.Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.teardown(t, db, mock)
		})
	}
}

func Test_secretRepository_Delete(t *testing.T) {
	secretIDHashed := "000c285457fc971f862a79b786476c78812c8897063c6fa9c045f579a3b2d63f"

	type args struct {
		httpContext context.Context
		id          string
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
				stmt := "DELETE FROM secrets WHERE id = \\$1"
				mock.ExpectPrepare(stmt)
				mock.ExpectExec(stmt).WithArgs(secretIDHashed).WillReturnResult(sqlmock.NewResult(1, 1))
				return db, mock
			},
			args: args{
				httpContext: context.Background(),
				id:          secretIDHashed,
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
			r := &secretRepository{
				db: db,
			}
			err := r.Delete(tt.args.httpContext, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("secretRepository.Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.teardown(t, db, mock)
		})
	}
}
