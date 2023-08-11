package secretmessage_test

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jarcoal/httpmock"
	"github.com/neufeldtech/secretmessage-go/pkg/secretmessage"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/slack-go/slack"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var _ = Describe("/secret", func() {
	responseURL := "https://fake-webhooks.fakeslack.com/response_url_1"
	teamID := "T1234ABCD"
	accessToken := "xoxb-1234"
	var requestBody = url.Values{}
	var gdb *gorm.DB
	var err error
	var ctl *secretmessage.PublicController
	var router *gin.Engine
	var serverResponse *httptest.ResponseRecorder

	BeforeEach(func() {
		requestBody = url.Values{
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
			"token":           []string{accessToken},
			"channel_name":    []string{"fishbowl"},
			"trigger_id":      []string{"0000000000.1111111111.222222222222aaaaaaaaaaaaaa"},
		}

		httpmock.Activate()
		gdb, err = gorm.Open(sqlite.Open("file::memory:?cache=shared&dbname=handle_slash"), &gorm.Config{})
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
		serverResponse = doHttpRequest(router, strings.NewReader(requestBody.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "POST", "/slash")
	})

	AfterEach(func() {
		httpmock.DeactivateAndReset()
		db, _ := gdb.DB()
		db.Close()
	})

	Context("on happy path with team and access token present in DB", func() {
		BeforeEach(func() {
			httpmock.RegisterResponder("POST", responseURL, httpmock.NewStringResponder(200, `ok`))
			tx := gdb.Create(&secretmessage.Team{ID: teamID, AccessToken: accessToken})
			Expect(tx.Error).To(BeNil())
			var s secretmessage.Secret
			tx = gdb.Take(&s)
			Expect(tx.RowsAffected).To(BeEquivalentTo(0))
		})
		It("should have a zero byte body", func() {
			b, _ := ioutil.ReadAll(serverResponse.Body)
			Expect(len(b)).To(BeZero())
		})
		It("should respond with 200", func() {
			Expect(serverResponse.Code).To(Equal(http.StatusOK))
		})
		It("should POST to Slack at responseURL exactly once", func() {
			Expect(httpmock.GetTotalCallCount()).To(Equal(1))
		})
		It("should store the message", func() {
			var s secretmessage.Secret
			gdb.Take(&s)
			Expect(s.ID).To(MatchRegexp(`^[a-f0-9]{64}$`))
			Expect(s.Value).To(MatchRegexp(`^[a-f0-9]{1,}$`))
		})
	})

	Context("on happy path with team not in DB", func() {
		BeforeEach(func() {
			httpmock.RegisterResponder("POST", responseURL, httpmock.NewStringResponder(200, `ok`))
			tx := gdb.First(&secretmessage.Team{ID: teamID})
			Expect(tx.RowsAffected).To(BeEquivalentTo(0))
		})
		It("should have a zero byte body", func() {
			b, _ := ioutil.ReadAll(serverResponse.Body)
			Expect(len(b)).To(BeZero())
		})
		It("should respond with 200", func() {
			Expect(serverResponse.Code).To(Equal(http.StatusOK))
		})
		It("should POST to Slack at responseURL exactly twice", func() {
			Expect(httpmock.GetTotalCallCount()).To(BeEquivalentTo(2))
		})
	})

	Context("on db error storing secret", func() {
		BeforeEach(func() {
			// Close the DB early to force an error
			db, _ := gdb.DB()
			db.Close()
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

	Context("on empty text given", func() {
		BeforeEach(func() {
			requestBody = url.Values{
				"command":         []string{"/secret"},
				"team_domain":     []string{"myteam"},
				"enterprise_id":   []string{"E0001"},
				"enterprise_name": []string{"Globular%20Construct%20Inc"},
				"channel_id":      []string{"C1234ABCD"},
				"text":            []string{""},
				"team_id":         []string{teamID},
				"user_id":         []string{"U1234ABCD"},
				"user_name":       []string{"imafish"},
				"response_url":    []string{responseURL},
				"token":           []string{"xoxb-1234"},
				"channel_name":    []string{"fishbowl"},
				"trigger_id":      []string{"0000000000.1111111111.222222222222aaaaaaaaaaaaaa"},
			}
		})
		It("should return a useful error message", func() {
			var msg slack.Message
			b, _ := ioutil.ReadAll(serverResponse.Body)
			json.Unmarshal(b, &msg)
			Expect(msg.Attachments[0].Text).To(MatchRegexp(`It looks like you tried to send a secret but forgot to provide the secret's text`))
		})
		It("should respond with 200", func() {
			Expect(serverResponse.Code).To(Equal(http.StatusOK))
		})
	})
	Context("on error sending responseURL POST msg to slack", func() {
		BeforeEach(func() {
			httpmock.RegisterResponder("POST", responseURL, httpmock.NewStringResponder(503, `ok`))
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
