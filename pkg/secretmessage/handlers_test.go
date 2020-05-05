package secretmessage

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/slack-go/slack"

	"github.com/stretchr/testify/assert"
)

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
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

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
		"response_url":    []string{"https://hooks.slack.com/commands/XXXXXXXX/00000000000/YYYYYYYYYYYYYY"},
		"token":           []string{"xoxb-1234"},
		"channel_name":    []string{"fishbowl"},
		"trigger_id":      []string{"0000000000.1111111111.222222222222aaaaaaaaaaaaaa"},
	}
	// Grab our router
	config := Config{
		RedisAddress:             s.Addr(),
		SigningSecret:            "secret",
		SkipSkignatureValidation: true,
	}
	router := SetupRouter(config)
	// Perform a GET request with that handler.

	w := postRequest(router, strings.NewReader(requestBody.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "POST", "/slash")

	assert.Equal(t, http.StatusOK, w.Code)

	var response slack.Message
	b, err := ioutil.ReadAll(w.Body)
	err = json.Unmarshal(b, &response)

	assert.Nil(t, err)

	if len(response.Attachments) < 1 {
		assert.FailNow(t, "Expected at least 1 response.Attachments")
	}
	assert.Equal(t, "imafish sent a secret message", response.Attachments[0].Title)
	if len(response.Attachments[0].Actions) < 1 {
		assert.FailNow(t, "Expected at least 1 response.Attachments[0].Actions")
	}
	assert.Equal(t, ":envelope: Read message", response.Attachments[0].Actions[0].Text)
}
func TestHandleInteractive(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

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
		"response_url":    []string{"https://hooks.slack.com/commands/XXXXXXXX/00000000000/YYYYYYYYYYYYYY"},
		"token":           []string{"xoxb-1234"},
		"channel_name":    []string{"fishbowl"},
		"trigger_id":      []string{"0000000000.1111111111.222222222222aaaaaaaaaaaaaa"},
	}
	// Grab our router
	config := Config{
		RedisAddress:             s.Addr(),
		SigningSecret:            "secret",
		SkipSkignatureValidation: true,
	}
	router := SetupRouter(config)
	// Perform a GET request with that handler.

	w := postRequest(router, strings.NewReader(requestBody.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "POST", "/slash")

	assert.Equal(t, http.StatusOK, w.Code)

	var response slack.Message
	b, err := ioutil.ReadAll(w.Body)
	err = json.Unmarshal(b, &response)

	assert.Nil(t, err)

	if len(response.Attachments) < 1 {
		assert.FailNow(t, "Expected at least 1 response.Attachments")
	}
	assert.Equal(t, "imafish sent a secret message", response.Attachments[0].Title)
	if len(response.Attachments[0].Actions) < 1 {
		assert.FailNow(t, "Expected at least 1 response.Attachments[0].Actions")
	}
	assert.Equal(t, ":envelope: Read message", response.Attachments[0].Actions[0].Text)
}
