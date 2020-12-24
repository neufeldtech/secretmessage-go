package secretmessage_test

import (
	"database/sql"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jarcoal/httpmock"
	"github.com/neufeldtech/secretmessage-go/pkg/secretmessage"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var _ = Describe("/auth/slack/callback", func() {
	var mock sqlmock.Sqlmock
	var db *sql.DB
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
		db, mock, err = sqlmock.New()
		if err != nil {
			log.Fatalf("error initializing sqlmock %v", err)
		}
		gdb, err = gorm.Open(postgres.New(postgres.Config{
			Conn: db,
		}), &gorm.Config{})
		if err != nil {
			log.Fatal(err)
		}
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
		Expect(mock.ExpectationsWereMet()).To(BeNil())
		httpmock.DeactivateAndReset()
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
			BeforeEach(func() {
				stmt := `SELECT \* FROM "teams" WHERE "teams"\."id" \= \$1 AND "teams"\."deleted_at" IS NULL ORDER BY "teams"\."id" LIMIT 1`
				rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "access_token", "scope", "name", "paid"})
				mock.ExpectQuery(stmt).WithArgs(teamID).WillReturnRows(rows)

				stmt = `INSERT INTO "teams" \("id","created_at","updated_at","deleted_at","access_token","scope","name","paid"\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6,\$7,\$8\)`
				mock.ExpectExec(stmt).WithArgs(teamID, AnyTime{}, AnyTime{}, nil, accessToken, scopes, teamName, false).WillReturnResult(sqlmock.NewResult(1, 1))
			})
			It("redirects to success page", func() {
				Expect(serverResponse.Code).To(Equal(http.StatusFound))
				Expect(serverResponse.Result().Header.Get("Location")).To(MatchRegexp(`/success`))
			})
		})
		Context("when team already exists in db", func() {
			BeforeEach(func() {
				stmt := `SELECT \* FROM "teams" WHERE "teams"\."id" \= \$1 AND "teams"\."deleted_at" IS NULL ORDER BY "teams"\."id" LIMIT 1`
				rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "access_token", "scope", "name", "paid"}).
					AddRow(teamID, time.Now(), time.Now(), nil, "old-access-token", "oldscope", "oldname", true)
				mock.ExpectQuery(stmt).WithArgs(teamID).WillReturnRows(rows)

				stmt = `UPDATE "teams" SET "access_token"\=\$1,"name"\=\$2,"scope"\=\$3,"updated_at"\=\$4 WHERE "teams"\."id" \= \$5 AND "teams"\."deleted_at" IS NULL AND "id" \= \$6`
				mock.ExpectExec(stmt).WithArgs(accessToken, teamName, scopes, AnyTime{}, teamID, teamID).WillReturnResult(sqlmock.NewResult(1, 1))
			})
			It("redirects to success page", func() {
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
