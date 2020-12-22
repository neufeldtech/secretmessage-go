package secretmessage_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jarcoal/httpmock"
	"github.com/neufeldtech/secretmessage-go/pkg/secretmessage"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/slack-go/slack"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var _ = Describe("/secret", func() {
	responseURL := "https://fake-webhooks.fakeslack.com/response_url_1"
	teamID := "T1234ABCD"
	requestBody := url.Values{
		"command":         []string{"/secret"},
		"team_domain":     []string{"myteam"},
		"enterprise_id":   []string{"E0001"},
		"enterprise_name": []string{"Globular%20Construct%20Inc"},
		"channel_id":      []string{"C1234ABCD"},
		"text":            []string{"this is my secret"},
		"team_id":         []string{teamID},
		"user_id":         []string{"U1234ABCD"},
		"user_name":       []string{"imafish"},
		"response_url":    []string{responseURL},
		"token":           []string{"xoxb-1234"},
		"channel_name":    []string{"fishbowl"},
		"trigger_id":      []string{"0000000000.1111111111.222222222222aaaaaaaaaaaaaa"},
	}
	// secretID := "monkey"
	secretIDHashed := "000c285457fc971f862a79b786476c78812c8897063c6fa9c045f579a3b2d63f"
	encryptedPayload := "30303030303030303030303029c9922a9be75ba2e6be5afd32d19387baea51fa577c0c51dc9809a54adb9085490f109237d15a3262a585"

	var mock sqlmock.Sqlmock
	var db *sql.DB
	var gdb *gorm.DB
	var err error
	var ctl *secretmessage.PublicController
	var router *gin.Engine
	var serverResponse *httptest.ResponseRecorder

	BeforeEach(func() {
		// Configuration
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
			secretmessage.Config{SkipSignatureValidation: true},
			gdb,
		)
	})
	JustBeforeEach(func() {
		// creation of objects
		router = ctl.ConfigureRoutes()
		serverResponse = postRequest(router, strings.NewReader(requestBody.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "POST", "/slash")
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(BeNil())
		httpmock.DeactivateAndReset()
		db.Close()
	})

	Context("on happy path with team present in DB", func() {
		BeforeEach(func() {
			httpmock.RegisterResponder("POST", responseURL, httpmock.NewStringResponder(200, `ok`))
			stmt := `INSERT INTO "secrets" \("id","created_at","updated_at","deleted_at","expires_at","value"\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6\)`
			mock.ExpectExec(stmt).WithArgs(AnySecretID{}, AnyTime{}, AnyTime{}, nil, AnyTime{}, AnySecretValue{}).WillReturnResult(sqlmock.NewResult(1, 1))
			stmt = `SELECT \* FROM "teams" WHERE id \= \$1 AND "teams"\."deleted_at" IS NULL ORDER BY "teams"\."id" LIMIT 1`
			mock.ExpectQuery(stmt).WithArgs(teamID).WillReturnError(fmt.Errorf("no rows fam"))
		})
		It("should have a zero byte body", func() {
			b, _ := ioutil.ReadAll(serverResponse.Body)
			Expect(len(b)).To(BeZero())
		})
		It("should respond with 200", func() {
			Expect(serverResponse.Code).To(Equal(http.StatusOK))
		})
	})

	Context("on happy path with team not in DB", func() {
		BeforeEach(func() {
			httpmock.RegisterResponder("POST", responseURL, httpmock.NewStringResponder(200, `ok`))
			stmt := `INSERT INTO "secrets" \("id","created_at","updated_at","deleted_at","expires_at","value"\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6\)`
			mock.ExpectExec(stmt).WithArgs(AnySecretID{}, AnyTime{}, AnyTime{}, nil, AnyTime{}, AnySecretValue{}).WillReturnResult(sqlmock.NewResult(1, 1))

			rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "expires_at", "value"}).AddRow(
				secretIDHashed,
				time.Now(),
				time.Now(),
				nil,
				time.Now().Add(time.Hour),
				encryptedPayload,
			)
			stmt = `SELECT \* FROM "teams" WHERE id \= \$1 AND "teams"\."deleted_at" IS NULL ORDER BY "teams"\."id" LIMIT 1`
			mock.ExpectQuery(stmt).WithArgs(teamID).WillReturnRows(rows)
		})
		It("should have a zero byte body", func() {
			b, _ := ioutil.ReadAll(serverResponse.Body)
			Expect(len(b)).To(BeZero())
		})
		It("should respond with 200", func() {
			Expect(serverResponse.Code).To(Equal(http.StatusOK))
		})
	})

	Context("on db error storing secret", func() {
		BeforeEach(func() {
			httpmock.RegisterResponder("POST", responseURL, httpmock.NewStringResponder(200, `ok`))
			stmt := `INSERT INTO "secrets" \("id","created_at","updated_at","deleted_at","expires_at","value"\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6\)`
			mock.ExpectExec(stmt).WithArgs(AnySecretID{}, AnyTime{}, AnyTime{}, nil, AnyTime{}, AnySecretValue{}).WillReturnError(fmt.Errorf("this is a db error"))
		})
		It("should return a useful error message", func() {
			var msg slack.Message
			b, _ := ioutil.ReadAll(serverResponse.Body)
			json.Unmarshal(b, &msg)
			Expect(msg.Attachments[0].Text).To(MatchRegexp(`An error occurred attempting to create secret`))
		})
		It("should respond with 200", func() {
			Expect(serverResponse.Code).To(Equal(http.StatusOK))
		})
	})
	Context("on error sending responseURL POST msg to slack", func() {
		BeforeEach(func() {
			httpmock.RegisterResponder("POST", responseURL, httpmock.NewStringResponder(503, `ok`))
			stmt := `INSERT INTO "secrets" \("id","created_at","updated_at","deleted_at","expires_at","value"\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6\)`
			mock.ExpectExec(stmt).WithArgs(AnySecretID{}, AnyTime{}, AnyTime{}, nil, AnyTime{}, AnySecretValue{}).WillReturnError(fmt.Errorf("this is a db error"))
		})
		It("should return a useful error message", func() {
			var msg slack.Message
			b, _ := ioutil.ReadAll(serverResponse.Body)
			json.Unmarshal(b, &msg)
			Expect(msg.Attachments[0].Text).To(MatchRegexp(`An error occurred attempting to create secret`))
		})
		It("should respond with 200", func() {
			Expect(serverResponse.Code).To(Equal(http.StatusOK))
		})
	})
})
