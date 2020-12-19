package secretmessage

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jarcoal/httpmock"
	"github.com/neufeldtech/secretmessage-go/pkg/secretdb"
	"github.com/slack-go/slack"

	"github.com/stretchr/testify/assert"
)

// SQLMock Helpers
type AnyTime struct{}

// Match satisfies sqlmock.Argument interface
func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

type AnySecretID struct{}

// Match satisfies sqlmock.Argument interface
func (a AnySecretID) Match(v driver.Value) bool {
	id, ok := v.(string)
	if ok && len(id) == 64 {
		return true
	}
	return false
}

type AnySecretValue struct{}

// Match satisfies sqlmock.Argument interface
func (a AnySecretValue) Match(v driver.Value) bool {
	val, ok := v.(string)
	if ok && len(val) > 1 {
		return true
	}
	return false
}

// POST helper
func postRequest(r http.Handler, body io.Reader, headers map[string]string, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, body)
	for h, v := range headers {
		req.Header.Add(h, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandleSlashSecret(t *testing.T) {
	responseURL := "https://fake-webhooks.fakeslack.com/response_url_1"
	requestBody := url.Values{
		"command":         []string{"/secret"},
		"team_domain":     []string{"myteam"},
		"enterprise_id":   []string{"E0001"},
		"enterprise_name": []string{"Globular%20Construct%20Inc"},
		"channel_id":      []string{"C1234ABCD"},
		"text":            []string{"this is my secret"},
		"team_id":         []string{"T1234ABCD"},
		"user_id":         []string{"U1234ABCD"},
		"user_name":       []string{"imafish"},
		"response_url":    []string{responseURL},
		"token":           []string{"xoxb-1234"},
		"channel_name":    []string{"fishbowl"},
		"trigger_id":      []string{"0000000000.1111111111.222222222222aaaaaaaaaaaaaa"},
	}
	tests := []struct {
		name   string
		setup  func() (*PublicController, url.Values, sqlmock.Sqlmock, *sql.DB)
		verify func(*testing.T, *httptest.ResponseRecorder, sqlmock.Sqlmock)
		// requestBody url.Values
	}{
		{
			name: "happy path",
			setup: func() (*PublicController, url.Values, sqlmock.Sqlmock, *sql.DB) {
				httpmock.Activate()
				httpmock.RegisterResponder("POST", responseURL, httpmock.NewStringResponder(200, `ok`))
				db, mock, err := sqlmock.New()
				if err != nil {
					log.Fatalf("error initializing sqlmock %v", err)
				}
				stmt := "INSERT INTO secrets \\(id, created_at, expires_at, value\\) VALUES \\(\\$1, \\$2, \\$3, \\$4\\)"
				mock.ExpectPrepare(stmt)
				mock.ExpectExec(stmt).WithArgs(AnySecretID{}, AnyTime{}, AnyTime{}, AnySecretValue{}).WillReturnResult(sqlmock.NewResult(1, 1))

				ctl := NewController(
					db,
					secretdb.NewSecretsRepository(db),
				)
				return ctl, requestBody, mock, db
			},
			verify: func(t *testing.T, r *httptest.ResponseRecorder, mock sqlmock.Sqlmock) {
				assert.Equal(t, http.StatusOK, r.Code)
				b, err := ioutil.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.Len(t, b, 0)
				assert.NoError(t, mock.ExpectationsWereMet())
			},
		},
		{
			name: "db problem",
			setup: func() (*PublicController, url.Values, sqlmock.Sqlmock, *sql.DB) {
				httpmock.Activate()
				httpmock.RegisterResponder("POST", responseURL, httpmock.NewStringResponder(200, `ok`))
				db, mock, err := sqlmock.New()
				if err != nil {
					log.Fatalf("error initializing sqlmock %v", err)
				}
				stmt := "INSERT INTO secrets \\(id, created_at, expires_at, value\\) VALUES \\(\\$1, \\$2, \\$3, \\$4\\)"
				mock.ExpectPrepare(stmt)
				mock.ExpectExec(stmt).WithArgs(AnySecretID{}, AnyTime{}, AnyTime{}, AnySecretValue{}).WillReturnError(fmt.Errorf("the database encountered an error executing insert"))

				ctl := NewController(
					db,
					secretdb.NewSecretsRepository(db),
				)
				return ctl, requestBody, mock, db
			},
			verify: func(t *testing.T, r *httptest.ResponseRecorder, mock sqlmock.Sqlmock) {
				var response slack.Message
				b, _ := ioutil.ReadAll(r.Body)
				json.Unmarshal(b, &response)
				assert.Regexp(t, regexp.MustCompile(`An error occurred attempting to create secret`), response.Attachments[0].Text)
				assert.NoError(t, mock.ExpectationsWereMet())

			},
		},
		{
			name: "send message to slack problem",
			setup: func() (*PublicController, url.Values, sqlmock.Sqlmock, *sql.DB) {
				httpmock.Activate()
				httpmock.RegisterResponder("POST", responseURL, httpmock.NewStringResponder(500, `error`))
				db, mock, err := sqlmock.New()
				if err != nil {
					log.Fatalf("error initializing sqlmock %v", err)
				}
				stmt := "INSERT INTO secrets \\(id, created_at, expires_at, value\\) VALUES \\(\\$1, \\$2, \\$3, \\$4\\)"
				mock.ExpectPrepare(stmt)
				mock.ExpectExec(stmt).WithArgs(AnySecretID{}, AnyTime{}, AnyTime{}, AnySecretValue{}).WillReturnResult(sqlmock.NewResult(1, 1))

				ctl := NewController(
					db,
					secretdb.NewSecretsRepository(db),
				)
				return ctl, requestBody, mock, db
			},
			verify: func(t *testing.T, r *httptest.ResponseRecorder, mock sqlmock.Sqlmock) {
				var response slack.Message
				b, _ := ioutil.ReadAll(r.Body)
				json.Unmarshal(b, &response)
				assert.Regexp(t, regexp.MustCompile(`An error occurred attempting to create secret`), response.Attachments[0].Text)
				assert.NoError(t, mock.ExpectationsWereMet())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctl, requestBody, mock, db := tt.setup()
			defer db.Close()
			defer httpmock.DeactivateAndReset()
			router := ctl.ConfigureRoutes(Config{SkipSignatureValidation: true})
			recordedResponse := postRequest(router, strings.NewReader(requestBody.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "POST", "/slash")
			tt.verify(t, recordedResponse, mock)
		})
	}

}

func TestHandleInteractiveGetSecret(t *testing.T) {
	secretID := "monkey"
	secretIDHashed := "000c285457fc971f862a79b786476c78812c8897063c6fa9c045f579a3b2d63f"
	encryptedPayload := "30303030303030303030303029c9922a9be75ba2e6be5afd32d19387baea51fa577c0c51dc9809a54adb9085490f109237d15a3262a585"
	interactionPayload := slack.InteractionCallback{
		CallbackID: fmt.Sprintf("send_secret:%v", secretID),
	}
	interactionBytes, err := json.Marshal(interactionPayload)
	if err != nil {
		panic(err)
	}
	requestBody := url.Values{
		"payload": []string{string(interactionBytes)},
	}

	tests := []struct {
		name   string
		setup  func() (*PublicController, url.Values, sqlmock.Sqlmock, *sql.DB)
		verify func(*testing.T, *httptest.ResponseRecorder, sqlmock.Sqlmock)
	}{
		{
			name: "happy path",
			setup: func() (*PublicController, url.Values, sqlmock.Sqlmock, *sql.DB) {
				db, mock, err := sqlmock.New()
				if err != nil {
					log.Fatalf("error initializing sqlmock %v", err)
				}

				stmt := "SELECT id, created_at, expires_at, value FROM secrets WHERE id = \\$1"
				rows := sqlmock.NewRows([]string{"id", "created_at", "expires_at", "value"}).AddRow(
					secretIDHashed,
					time.Now(),
					time.Now(),
					encryptedPayload,
				)
				mock.ExpectQuery(stmt).WithArgs(secretIDHashed).WillReturnRows(rows)
				stmt = "DELETE FROM secrets WHERE id = \\$1"
				mock.ExpectPrepare(stmt)
				mock.ExpectExec(stmt).WithArgs(secretIDHashed).WillReturnResult(sqlmock.NewResult(1, 1))
				ctl := NewController(
					db,
					secretdb.NewSecretsRepository(db),
				)
				return ctl, requestBody, mock, db
			},
			verify: func(t *testing.T, r *httptest.ResponseRecorder, mock sqlmock.Sqlmock) {
				var response slack.Message
				b, _ := ioutil.ReadAll(r.Body)
				json.Unmarshal(b, &response)
				assert.Regexp(t, regexp.MustCompile(`the password is baseball123`), response.Attachments[0].Text)
				assert.NoError(t, mock.ExpectationsWereMet())
			},
		},
		{
			name: "secret not found",
			setup: func() (*PublicController, url.Values, sqlmock.Sqlmock, *sql.DB) {
				db, mock, err := sqlmock.New()
				if err != nil {
					log.Fatalf("error initializing sqlmock %v", err)
				}

				stmt := "SELECT id, created_at, expires_at, value FROM secrets WHERE id = \\$1"
				rows := sqlmock.NewRows([]string{"id", "created_at", "expires_at", "value"})
				mock.ExpectQuery(stmt).WithArgs(secretIDHashed).WillReturnRows(rows)
				ctl := NewController(
					db,
					secretdb.NewSecretsRepository(db),
				)
				return ctl, requestBody, mock, db
			},
			verify: func(t *testing.T, r *httptest.ResponseRecorder, mock sqlmock.Sqlmock) {
				var response slack.Message
				b, _ := ioutil.ReadAll(r.Body)
				json.Unmarshal(b, &response)
				assert.Regexp(t, regexp.MustCompile(`An error occurred attempting to retrieve secret`), response.Attachments[0].Text)
				assert.NoError(t, mock.ExpectationsWereMet())
			},
		},
		{
			name: "db error",
			setup: func() (*PublicController, url.Values, sqlmock.Sqlmock, *sql.DB) {
				db, mock, err := sqlmock.New()
				if err != nil {
					log.Fatalf("error initializing sqlmock %v", err)
				}

				stmt := "SELECT id, created_at, expires_at, value FROM secrets WHERE id = \\$1"
				mock.ExpectQuery(stmt).WithArgs(secretIDHashed).WillReturnError(fmt.Errorf("the db exploded"))
				ctl := NewController(
					db,
					secretdb.NewSecretsRepository(db),
				)
				return ctl, requestBody, mock, db
			},
			verify: func(t *testing.T, r *httptest.ResponseRecorder, mock sqlmock.Sqlmock) {
				var response slack.Message
				b, _ := ioutil.ReadAll(r.Body)
				json.Unmarshal(b, &response)
				assert.Regexp(t, regexp.MustCompile(`An error occurred attempting to retrieve secret`), response.Attachments[0].Text)
				assert.NoError(t, mock.ExpectationsWereMet())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctl, requestBody, mock, db := tt.setup()

			defer db.Close()
			defer httpmock.DeactivateAndReset()
			router := ctl.ConfigureRoutes(Config{SkipSignatureValidation: true})
			recordedResponse := postRequest(router, strings.NewReader(requestBody.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "POST", "/interactive")
			tt.verify(t, recordedResponse, mock)
		})
	}
}
func TestHandleInteractiveDeleteSecret(t *testing.T) {
	secretID := "monkey"
	interactionPayload := slack.InteractionCallback{
		CallbackID: fmt.Sprintf("delete_secret:%v", secretID),
	}
	interactionBytes, err := json.Marshal(interactionPayload)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	requestBody := url.Values{
		"payload": []string{string(interactionBytes)},
	}

	tests := []struct {
		name   string
		setup  func() (*PublicController, url.Values, sqlmock.Sqlmock, *sql.DB)
		verify func(*testing.T, *httptest.ResponseRecorder, sqlmock.Sqlmock)
	}{
		{
			name: "happy path",
			setup: func() (*PublicController, url.Values, sqlmock.Sqlmock, *sql.DB) {
				db, mock, err := sqlmock.New()
				if err != nil {
					log.Fatalf("error initializing sqlmock %v", err)
				}

				ctl := NewController(
					db,
					secretdb.NewSecretsRepository(db),
				)
				return ctl, requestBody, mock, db
			},
			verify: func(t *testing.T, r *httptest.ResponseRecorder, mock sqlmock.Sqlmock) {
				var response slack.Message
				b, _ := ioutil.ReadAll(r.Body)
				json.Unmarshal(b, &response)
				assert.True(t, response.DeleteOriginal)
				assert.NoError(t, mock.ExpectationsWereMet())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctl, requestBody, mock, db := tt.setup()

			defer db.Close()
			defer httpmock.DeactivateAndReset()
			router := ctl.ConfigureRoutes(Config{SkipSignatureValidation: true})
			recordedResponse := postRequest(router, strings.NewReader(requestBody.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "POST", "/interactive")
			tt.verify(t, recordedResponse, mock)
		})
	}
}
