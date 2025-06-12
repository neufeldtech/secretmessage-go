package secretmessage_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jarcoal/httpmock"
	"github.com/neufeldtech/secretmessage-go/pkg/secretmessage"
	"github.com/neufeldtech/secretmessage-go/pkg/secretmessage/actions"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/slack-go/slack"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var _ = Describe("/interactive", func() {
	var gdb *gorm.DB
	var ctl *secretmessage.PublicController
	var router *gin.Engine
	var serverResponse *httptest.ResponseRecorder
	secretID := "monkey"
	secretIDHashed := "000c285457fc971f862a79b786476c78812c8897063c6fa9c045f579a3b2d63f"
	encryptedPayload := "30303030303030303030303029c9922a9be75ba2e6be5afd32d19387baea51fa577c0c51dc9809a54adb9085490f109237d15a3262a585"

	Describe("Get Secret", func() {
		interactionPayload := slack.InteractionCallback{
			CallbackID: fmt.Sprintf("%s:%v", actions.ReadMessage, secretID),
		}
		interactionBytes, err := json.Marshal(interactionPayload)
		if err != nil {
			panic(err)
		}
		requestBody := url.Values{
			"payload": []string{string(interactionBytes)},
		}

		BeforeEach(func() {
			httpmock.Activate()
			gdb, err = gorm.Open(sqlite.Open("file::memory:?cache=shared&dbname=handle_interactive_get"), &gorm.Config{})
			if err != nil {
				log.Fatal(err)
			}
			gdb.AutoMigrate(secretmessage.Team{})
			gdb.AutoMigrate(secretmessage.Secret{})
			ctl = secretmessage.NewController(
				secretmessage.Config{SkipSignatureValidation: true},
				gdb,
			)
		})
		JustBeforeEach(func() {
			router = ctl.ConfigureRoutes()
			serverResponse = doHttpRequest(router, strings.NewReader(requestBody.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "POST", "/interactive")
		})
		AfterEach(func() {
			httpmock.DeactivateAndReset()
			db, _ := gdb.DB()
			db.Close()
		})

		Context("on happy path", func() {
			BeforeEach(func() {
				tx := gdb.Create(&secretmessage.Secret{ID: secretIDHashed, Value: encryptedPayload, ExpiresAt: time.Now().Add(time.Hour)})
				Expect(tx.RowsAffected).To(BeEquivalentTo(1))
			})
			It("should return decrypted secret", func() {
				var msg slack.Message
				b, _ := ioutil.ReadAll(serverResponse.Body)
				json.Unmarshal(b, &msg)
				Expect(serverResponse.Code).To(Equal(http.StatusOK))
				Expect(msg.Attachments[0].Text).To(MatchRegexp(`the password is baseball123`))
			})
			It("should delete secret from DB", func() {
				var s secretmessage.Secret
				tx := gdb.Take(&s)
				Expect(tx.RowsAffected).To(BeEquivalentTo(0))
			})
		})
		Context("on secret not found in DB", func() {
			BeforeEach(func() {
				var s secretmessage.Secret
				tx := gdb.Take(&s)
				Expect(tx.RowsAffected).To(BeEquivalentTo(0))
			})
			It("should return error message", func() {
				var msg slack.Message
				b, _ := ioutil.ReadAll(serverResponse.Body)
				json.Unmarshal(b, &msg)
				Expect(serverResponse.Code).To(Equal(http.StatusOK))
				Expect(msg.Attachments[0].Text).To(MatchRegexp(`This Secret has already been retrieved or has expired`))
				Expect(msg.DeleteOriginal).To(BeTrue())
			})
		})
		Context("on secret expired", func() {
			BeforeEach(func() {
				// Insert the secret with an expired timestamp
				tx := gdb.Create(&secretmessage.Secret{
					ID:        secretIDHashed,
					Value:     encryptedPayload,
					ExpiresAt: time.Now().Add(-time.Hour), // expired 1 hour ago
				})
				Expect(tx.RowsAffected).To(BeEquivalentTo(1))
			})
			It("should return error message for expired secret", func() {
				var msg slack.Message
				b, _ := ioutil.ReadAll(serverResponse.Body)
				json.Unmarshal(b, &msg)
				Expect(serverResponse.Code).To(Equal(http.StatusOK))
				Expect(msg.Attachments[0].Text).To(MatchRegexp(`This Secret has expired`))
				Expect(msg.DeleteOriginal).To(BeTrue())
			})
		})
		Context("on db error", func() {
			BeforeEach(func() {
				// force an error by closing DB
				db, _ := gdb.DB()
				db.Close()
			})
			It("should return error message", func() {
				var msg slack.Message
				b, _ := ioutil.ReadAll(serverResponse.Body)
				json.Unmarshal(b, &msg)
				Expect(serverResponse.Code).To(Equal(http.StatusOK))
				Expect(msg.Attachments[0].Text).To(MatchRegexp(`An error occurred attempting to retrieve secret`))
				Expect(msg.DeleteOriginal).To(BeFalse())
			})
		})
	})

	Describe("Delete Secret", func() {
		interactionPayload := slack.InteractionCallback{
			CallbackID: fmt.Sprintf("%s:%v", actions.DeleteMessage, secretID),
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
			gdb, err = gorm.Open(sqlite.Open("file::memory:?cache=shared&dbname=handle_interactive_delete"), &gorm.Config{})
			if err != nil {
				log.Fatal(err)
			}
			gdb.AutoMigrate(secretmessage.Team{})
			gdb.AutoMigrate(secretmessage.Secret{})
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
			db, _ := gdb.DB()
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
