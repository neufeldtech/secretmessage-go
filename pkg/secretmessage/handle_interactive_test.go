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

var _ = Describe("/interactive", func() {
	var mock sqlmock.Sqlmock
	var db *sql.DB
	var gdb *gorm.DB
	// var err error
	var ctl *secretmessage.PublicController
	var router *gin.Engine
	var serverResponse *httptest.ResponseRecorder
	secretID := "monkey"
	secretIDHashed := "000c285457fc971f862a79b786476c78812c8897063c6fa9c045f579a3b2d63f"
	encryptedPayload := "30303030303030303030303029c9922a9be75ba2e6be5afd32d19387baea51fa577c0c51dc9809a54adb9085490f109237d15a3262a585"

	Describe("Get Secret", func() {
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
			serverResponse = doHttpRequest(router, strings.NewReader(requestBody.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "POST", "/interactive")
		})
		AfterEach(func() {
			Expect(mock.ExpectationsWereMet()).To(BeNil())
			httpmock.DeactivateAndReset()
			db.Close()
		})

		Context("on happy path", func() {
			BeforeEach(func() {
				stmt := `SELECT \* FROM "secrets" WHERE id \= \$1 AND "secrets"\."deleted_at" IS NULL ORDER BY "secrets"\."id" LIMIT 1`
				rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "expires_at", "value"}).AddRow(
					secretIDHashed,
					time.Now(),
					time.Now(),
					nil,
					time.Now(),
					encryptedPayload,
				)
				mock.ExpectQuery(stmt).WithArgs(secretIDHashed).WillReturnRows(rows)
				stmt = `DELETE FROM "secrets" WHERE id \= \$1`
				mock.ExpectExec(stmt).WithArgs(secretIDHashed).WillReturnResult(sqlmock.NewResult(1, 1))
			})
			It("should return decrypted secret", func() {
				var msg slack.Message
				b, _ := ioutil.ReadAll(serverResponse.Body)
				json.Unmarshal(b, &msg)
				Expect(serverResponse.Code).To(Equal(http.StatusOK))
				Expect(msg.Attachments[0].Text).To(MatchRegexp(`the password is baseball123`))
			})
		})
		Context("on secret not found in DB", func() {
			BeforeEach(func() {
				stmt := `SELECT \* FROM "secrets" WHERE id \= \$1 AND "secrets"\."deleted_at" IS NULL ORDER BY "secrets"\."id" LIMIT 1`
				rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "expires_at", "value"})
				mock.ExpectQuery(stmt).WithArgs(secretIDHashed).WillReturnRows(rows)
			})
			It("should return error message", func() {
				var msg slack.Message
				b, _ := ioutil.ReadAll(serverResponse.Body)
				json.Unmarshal(b, &msg)
				Expect(serverResponse.Code).To(Equal(http.StatusOK))
				Expect(msg.Attachments[0].Text).To(MatchRegexp(`An error occurred attempting to retrieve secret`))
			})
		})
		Context("on db error", func() {
			BeforeEach(func() {
				stmt := `SELECT \* FROM "secrets" WHERE id \= \$1 AND "secrets"\."deleted_at" IS NULL ORDER BY "secrets"\."id" LIMIT 1`
				mock.ExpectQuery(stmt).WithArgs(secretIDHashed).WillReturnError(fmt.Errorf("something bad happened"))
			})
			It("should return error message", func() {
				var msg slack.Message
				b, _ := ioutil.ReadAll(serverResponse.Body)
				json.Unmarshal(b, &msg)
				Expect(serverResponse.Code).To(Equal(http.StatusOK))
				Expect(msg.Attachments[0].Text).To(MatchRegexp(`An error occurred attempting to retrieve secret`))
			})
		})
	})

	Describe("Delete Secret", func() {
		interactionPayload := slack.InteractionCallback{
			CallbackID: fmt.Sprintf("delete_secret:%v", secretID),
		}
		interactionBytes, err := json.Marshal(interactionPayload)
		if err != nil {
			log.Fatal(err)
		}
		requestBody := url.Values{
			"payload": []string{string(interactionBytes)},
		}

		BeforeEach(func() {
			// Configuration
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
			serverResponse = doHttpRequest(router, strings.NewReader(requestBody.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "POST", "/interactive")
		})
		AfterEach(func() {
			Expect(mock.ExpectationsWereMet()).To(BeNil())
			db.Close()
		})

		Context("on happy path", func() {
			It("should return deleteOriginal", func() {
				var msg slack.Message
				b, _ := ioutil.ReadAll(serverResponse.Body)
				json.Unmarshal(b, &msg)
				Expect(serverResponse.Code).To(Equal(http.StatusOK))
				Expect(msg.DeleteOriginal).To(BeTrue())
			})
		})
	})
})
