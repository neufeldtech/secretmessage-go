package secretmessage

import (
	"database/sql/driver"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jarcoal/httpmock"
	"github.com/neufeldtech/secretmessage-go/pkg/secretdb"
	"github.com/slack-go/slack"

	"github.com/stretchr/testify/assert"
)

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

func postRequest(r http.Handler, body io.Reader, headers map[string]string, method, path string) *httptest.ResponseRecorder {

	req, _ := http.NewRequest(method, path, body)
	for h, v := range headers {
		req.Header.Add(h, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandleSlash(t *testing.T) {
	slackResponseURL := "https://fake-webhooks.fakeslack.com/response_url_1"
	// needed for mocking
	// secretslack.OverrideHTTPClient(http.DefaultClient)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	// // Exact URL match
	httpmock.RegisterResponder("POST", slackResponseURL, httpmock.NewStringResponder(200, `ok`))

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
		"response_url":    []string{slackResponseURL},
		"token":           []string{"xoxb-1234"},
		"channel_name":    []string{"fishbowl"},
		"trigger_id":      []string{"0000000000.1111111111.222222222222aaaaaaaaaaaaaa"},
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("error initializing sqlmock %v", err)
	}
	defer db.Close()
	stmt := "INSERT INTO secrets \\(id, created_at, expires_at, value\\) VALUES \\(\\$1, \\$2, \\$3, \\$4\\)"
	mock.ExpectPrepare(stmt)
	mock.ExpectExec(stmt).WithArgs(AnySecretID{}, AnyTime{}, AnyTime{}, AnySecretValue{}).WillReturnResult(sqlmock.NewResult(1, 1))
	ctl := NewController(
		db,
		secretdb.NewSecretsRepository(db),
	)

	router := ctl.ConfigureRoutes(Config{SkipSignatureValidation: true})
	// Perform a POST to /slash endpoint as Slack.
	w := postRequest(router, strings.NewReader(requestBody.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "POST", "/slash")
	assert.Equal(t, http.StatusOK, w.Code)
	var response slack.Message
	b, err := ioutil.ReadAll(w.Body)
	assert.Len(t, b, 0)
	// The body will be 0 bytes on the happy path
	if len(b) > 0 {
		err = json.Unmarshal(b, &response)
		assert.Nil(t, err)
		assert.Len(t, response.Attachments, 0)
		assert.Len(t, response.Text, 0)
	}
	assert.Nil(t, mock.ExpectationsWereMet())
}

// func TestHandleInteractiveGetSecret(t *testing.T) {
// 	s, err := miniredis.Run()
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer s.Close()
// 	// Grab our router
// 	config := Config{
// 		RedisOptions: &redis.Options{
// 			Addr: s.Addr(),
// 		},
// 		SigningSecret:           "secret",
// 		SkipSignatureValidation: true,
// 	}
// 	router := SetupRouter(config)

// 	secretID := shortuuid.New()
// 	redisClient := secretredis.Client()
// 	secretEncrypted, err := encrypt("this is my secret", secretID)
// 	if err != nil {
// 		assert.Fail(t, err.Error())
// 	}
// 	redisClient.Set(hash(secretID), secretEncrypted, 0)

// 	interactionPayload := slack.InteractionCallback{
// 		CallbackID: fmt.Sprintf("send_secret:%v", secretID),
// 	}

// 	interactionBytes, err := json.Marshal(interactionPayload)
// 	if err != nil {
// 		assert.Fail(t, err.Error())
// 	}

// 	requestBody := url.Values{
// 		"payload": []string{string(interactionBytes)},
// 	}
// 	w := postRequest(router, strings.NewReader(requestBody.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "POST", "/interactive")
// 	assert.Equal(t, http.StatusOK, w.Code)

// 	var response slack.Message
// 	b, err := ioutil.ReadAll(w.Body)
// 	err = json.Unmarshal(b, &response)

// 	assert.Nil(t, err)

// 	if len(response.Attachments) < 1 {
// 		assert.FailNow(t, "Expected at least 1 response.Attachments")
// 	}
// 	assert.Equal(t, "Secret message", response.Attachments[0].Title)
// 	if len(response.Attachments[0].Actions) < 1 {
// 		assert.FailNow(t, "Expected at least 1 response.Attachments[0].Actions")
// 	}
// 	assert.Equal(t, "this is my secret", response.Attachments[0].Text)
// 	assert.Equal(t, ":x: Delete message", response.Attachments[0].Actions[0].Text)

// }
// func TestHandleInteractiveDeleteSecret(t *testing.T) {
// 	s, err := miniredis.Run()
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer s.Close()
// 	// Grab our router
// 	config := Config{
// 		RedisOptions: &redis.Options{
// 			Addr: s.Addr(),
// 		},
// 		SigningSecret:           "secret",
// 		SkipSignatureValidation: true,
// 	}
// 	router := SetupRouter(config)

// 	secretID := shortuuid.New()
// 	redisClient := secretredis.Client()
// 	secretEncrypted, err := encrypt("this is my secret", secretID)
// 	if err != nil {
// 		assert.Fail(t, err.Error())
// 	}
// 	redisClient.Set(hash(secretID), secretEncrypted, 0)

// 	interactionPayload := slack.InteractionCallback{
// 		CallbackID: fmt.Sprintf("delete_secret:%v", secretID),
// 	}

// 	interactionBytes, err := json.Marshal(interactionPayload)
// 	if err != nil {
// 		assert.Fail(t, err.Error())
// 	}

// 	requestBody := url.Values{
// 		"payload": []string{string(interactionBytes)},
// 	}
// 	w := postRequest(router, strings.NewReader(requestBody.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "POST", "/interactive")
// 	assert.Equal(t, http.StatusOK, w.Code)

// 	var response slack.Message
// 	b, err := ioutil.ReadAll(w.Body)
// 	err = json.Unmarshal(b, &response)
// 	assert.Nil(t, err)
// 	assert.Equal(t, true, response.DeleteOriginal)
// }
