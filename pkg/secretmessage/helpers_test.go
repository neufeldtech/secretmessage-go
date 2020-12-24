package secretmessage_test

import (
	"database/sql/driver"
	"io"
	"net/http"
	"net/http/httptest"
	"time"
)

func doHttpRequest(r http.Handler, body io.Reader, headers map[string]string, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, body)
	for h, v := range headers {
		req.Header.Add(h, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

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

type OauthToken struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	Team        struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"team"`
}
