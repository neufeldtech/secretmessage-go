package secretmessage

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid"
	"github.com/neufeldtech/secretmessage-go/pkg/secretslack"
	"github.com/prometheus/common/log"
	"github.com/slack-go/slack"
	"go.elastic.co/apm"
)

// PrepareAndSendSecretEnvelope encrypts the secret, stores in redis, and sends the 'envelope' back to slack
func PrepareAndSendSecretEnvelope(ctl *PublicController, c *gin.Context, tx *apm.Transaction, s slack.SlashCommand) error {
	hc := c.Request.Context()

	secretID := shortuuid.New()
	tx.Context.SetLabel("secretIDHash", hash(secretID))
	secretEncrypted, encryptErr := encrypt(s.Text, secretID)

	if encryptErr != nil {
		tx.Context.SetLabel("errorCode", "encrypt_error")
		log.Errorf("error storing secretID %v: %v", secretID, encryptErr)
		return encryptErr
	}

	// Store the secret
	secretStoreTime := time.Now()
	storeErr := ctl.db.WithContext(hc).Create(
		&Secret{
			ID:        hash(secretID),
			ExpiresAt: secretStoreTime.Add(time.Hour * 24 * 7),
			Value:     secretEncrypted,
		},
	).Error

	if storeErr != nil {
		tx.Context.SetLabel("errorCode", "redis_set_error")
		log.Errorf("error storing secretID %v: %v", secretID, storeErr)
		return storeErr
	}

	secretResponse := slack.Message{
		Msg: slack.Msg{
			ResponseType: slack.ResponseTypeInChannel,
			Attachments: []slack.Attachment{{
				Title:      fmt.Sprintf("%v sent a secret message", s.UserName),
				Fallback:   fmt.Sprintf("%v sent a secret message", s.UserName),
				CallbackID: fmt.Sprintf("send_secret:%v", secretID),
				Color:      "#6D5692",
				Actions: []slack.AttachmentAction{{
					Name:  "readMessage",
					Text:  ":envelope: Read message",
					Type:  "button",
					Value: "readMessage",
				}},
			}},
		},
	}

	sendSpan := tx.StartSpan("send_message", "client_request", nil)
	defer sendSpan.End()
	sendMessageErr := secretslack.SendResponseUrlMessage(hc, s.ResponseURL, secretResponse)
	if sendMessageErr != nil {
		sendSpan.Context.SetLabel("errorCode", "send_message_error")
		log.Errorf("error sending secret to slack: %v", sendMessageErr)
		return sendMessageErr
	}

	return nil

}

// SlashSecret is the main entrypoint for the slash command /secret
func SlashSecret(ctl *PublicController, c *gin.Context, tx *apm.Transaction, s slack.SlashCommand) {

	tx.Context.SetLabel("userHash", hash(s.UserID))
	tx.Context.SetLabel("teamHash", hash(s.TeamID))
	tx.Context.SetLabel("action", "createSecret")
	tx.Context.SetLabel("slashCommand", "/secret")

	// Handle if no input was given
	if s.Text == "" {
		res, code := secretslack.NewSlackErrorResponse(
			"Error: secret text is empty",
			"It looks like you tried to send a secret but forgot to provide the secret's text. You can send a secret like this: `/secret I am scared of heights`",
			false,
			"secret_text_empty")
		tx.Context.SetLabel("errorCode", "text_empty")
		c.Data(code, gin.MIMEJSON, res)
		return
	}

	// Prepare and send message to channel using response_url link
	err := PrepareAndSendSecretEnvelope(ctl, c, tx, s)
	if err != nil {
		log.Error(err)
		res, code := secretslack.NewSlackErrorResponse(
			":x: Sorry, an error occurred",
			"An error occurred attempting to create secret",
			false,
			"prepare_and_send_error")
		tx.Context.SetLabel("errorCode", "send_secret_payload_error")
		c.Data(code, gin.MIMEJSON, res)
		c.Abort()
		return
	}

	// Send empty Ack to Slack if we got here without errors
	c.Data(http.StatusOK, gin.MIMEPlain, nil)

	if AppReinstallNeeded(ctl, c, tx, s) {
		SendReinstallMessage(ctl, c, tx, s)
	}

	return
}

func AppReinstallNeeded(ctl *PublicController, c *gin.Context, tx *apm.Transaction, s slack.SlashCommand) bool {
	var team Team
	err := ctl.db.WithContext(c).Where("id = ?", s.TeamID).First(&team).Error
	if err != nil || team.AccessToken == "" {
		log.Warnf("%v: could not find access_token for team %v in store", err, s.TeamID)
		return true
	}
	return false
}

func SendReinstallMessage(ctl *PublicController, c *gin.Context, tx *apm.Transaction, s slack.SlashCommand) {
	responseEphemeral := slack.Message{
		Msg: slack.Msg{
			ResponseType: slack.ResponseTypeEphemeral,
			Text:         fmt.Sprintf(":wave: Hey, we're working hard updating Secret Message. In order to keep using the app, <%v/auth/slack|please click here to reinstall>", ctl.config.AppURL),
		},
	}
	sendMessageEphemeralErr := secretslack.SendResponseUrlMessage(c.Request.Context(), s.ResponseURL, responseEphemeral)
	if sendMessageEphemeralErr != nil {
		log.Errorf("error sending ephemeral reinstall message: %v", sendMessageEphemeralErr)
	}
}
