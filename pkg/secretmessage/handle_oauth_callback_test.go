package secretmessage_test

import (
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jarcoal/httpmock"
	"github.com/neufeldtech/secretmessage-go/pkg/secretmessage"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var _ = Describe("/auth/slack/callback", func() {
	var gdb *gorm.DB
	var err error
	var ctl *secretmessage.PublicController
	var router *gin.Engine
	var serverResponse *httptest.ResponseRecorder
	var callbackURI string
	var callbackHeaders = make(map[string]string)
	var teamID = "T0000001"
	var teamName = "foobar"
	var accessToken = "xoxb-foobar"
	var scopes = "scope1,scope2,scope3"
	BeforeEach(func() {
		httpmock.Activate()
		gdb, err = gorm.Open(sqlite.Open("file::memory:?cache=shared&dbname=handle_oauth_callback"), &gorm.Config{})
		if err != nil {
			log.Fatal(err)
		}
		gdb.AutoMigrate(secretmessage.Team{})
		gdb.AutoMigrate(secretmessage.Secret{})
		ctl = secretmessage.NewController(
			secretmessage.Config{
				SkipSignatureValidation: true,
				OauthConfig: &oauth2.Config{
					ClientID:     "myclientID",
					ClientSecret: "myClientSecret",
					RedirectURL:  "https://localhost/foo",
					Scopes:       []string{"anyscope", "anotherscope"},
					Endpoint: oauth2.Endpoint{
						AuthURL:  "https://testingslack.com/oauth/v2/authorize",
						TokenURL: "https://testingslack.com/api/oauth.v2.access",
					},
				},
			},
			gdb,
			nil,
		)
		callbackURI = "/auth/slack/callback"
		// error=access_denied&state=7wos7tXr2zj9t7oU3mXGK7
		// code=foo
	})

	JustBeforeEach(func() {
		// creation of objects
		router = ctl.ConfigureRoutes()
		// Call our server route!
		serverResponse = doHttpRequest(router, nil, callbackHeaders, "GET", callbackURI)
	})
	AfterEach(func() {
		httpmock.DeactivateAndReset()
		db, _ := gdb.DB()
		db.Close()
	})
	Context("on happy path", func() {
		BeforeEach(func() {
			callbackHeaders["Cookie"] = "state=abc;"
			callbackURI = "/auth/slack/callback?state=abc&code=123"
			httpmock.RegisterResponder("POST", "https://testingslack.com/api/oauth.v2.access", httpmock.NewJsonResponderOrPanic(200, OauthToken{
				AccessToken: accessToken,
				Scope:       scopes,
				Team: struct {
					Name string `json:"name"`
					ID   string `json:"id"`
				}{
					Name: teamName,
					ID:   teamID,
				},
			}))
		})
		Context("when team does not exist in db", func() {
			It("redirects to success page and has a recently created record", func() {
				var team secretmessage.Team
				threshold := time.Now().Add(-time.Minute)
				tx := gdb.First(&team, "id = ? AND created_at >= ?", teamID, threshold)
				Expect(tx.RowsAffected).To(BeEquivalentTo(1))
				Expect(serverResponse.Code).To(Equal(http.StatusFound))
				Expect(serverResponse.Result().Header.Get("Location")).To(MatchRegexp(`/success`))
			})
		})
		Context("when team already exists in db", func() {
			createTime := time.Time{}
			BeforeEach(func() {
				team := secretmessage.Team{ID: teamID}
				tx := gdb.Create(&team).First(&team)
				createTime = team.CreatedAt
				Expect(tx.RowsAffected).To(BeEquivalentTo(1))
			})
			It("redirects to success page without creating any team", func() {
				var team secretmessage.Team
				tx := gdb.First(&team, "id = ? AND created_at = ?", teamID, createTime)
				Expect(tx.RowsAffected).To(BeEquivalentTo(1))
				Expect(serverResponse.Code).To(Equal(http.StatusFound))
				Expect(serverResponse.Result().Header.Get("Location")).To(MatchRegexp(`/success`))
			})
		})

	})

	Context("when state cookie doesn't match state query-string ", func() {
		BeforeEach(func() {
			callbackHeaders["Cookie"] = "state=nope;"
			callbackURI = "/auth/slack/callback?state=abc"
		})
		It("redirects to error page", func() {
			Expect(serverResponse.Code).To(Equal(http.StatusFound))
			Expect(serverResponse.Result().Header.Get("Location")).To(MatchRegexp(`/error`))
		})
	})

	Context("when state cookie missing", func() {
		BeforeEach(func() {
			callbackHeaders["Cookie"] = "foo=bar;"
			callbackURI = "/auth/slack/callback?state=abc"
		})
		It("redirects to error page", func() {
			Expect(serverResponse.Code).To(Equal(http.StatusFound))
			Expect(serverResponse.Result().Header.Get("Location")).To(MatchRegexp(`/error`))
		})
	})

	Context("when state query-string param missing", func() {
		BeforeEach(func() {
			callbackHeaders["Cookie"] = "state=abc;"
			callbackURI = "/auth/slack/callback"
		})
		It("redirects to error page", func() {
			Expect(serverResponse.Code).To(Equal(http.StatusFound))
			Expect(serverResponse.Result().Header.Get("Location")).To(MatchRegexp(`/error`))
		})
	})

	Context("when token parse error", func() {
		BeforeEach(func() {
			callbackHeaders["Cookie"] = "state=abc;"
			callbackURI = "/auth/slack/callback?state=abc"
			httpmock.RegisterResponder("POST", "https://testingslack.com/api/oauth.v2.access", httpmock.NewJsonResponderOrPanic(200, OauthToken{
				AccessToken: accessToken,
				// Team is missing
			}))
		})
		It("redirects to error page", func() {
			Expect(serverResponse.Code).To(Equal(http.StatusFound))
			Expect(serverResponse.Result().Header.Get("Location")).To(MatchRegexp(`/error`))
		})
	})

	Context("when http error from Slack on Token Exchange", func() {
		BeforeEach(func() {
			callbackHeaders["Cookie"] = "state=abc;"
			callbackURI = "/auth/slack/callback?state=abc"
			httpmock.RegisterResponder("POST", "https://testingslack.com/api/oauth.v2.access", httpmock.NewStringResponder(503, "something bad happened"))
		})
		It("redirects to error page", func() {
			Expect(serverResponse.Code).To(Equal(http.StatusFound))
			Expect(serverResponse.Result().Header.Get("Location")).To(MatchRegexp(`/error`))
		})
	})

})
