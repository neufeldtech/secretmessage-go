package secretmessage

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid"
	"github.com/neufeldtech/secretmessage-go/pkg/redis"
	"github.com/neufeldtech/secretmessage-go/pkg/slack"
	"github.com/prometheus/common/log"
	sl "github.com/slack-go/slack"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmgoredis"
)

func PrepareAndSendSecret(c *gin.Context, tx *apm.Transaction, s sl.SlashCommand) error {
	r := apmgoredis.Wrap(redis.GetRedisClient()).WithContext(c.Request.Context())

	secretID := shortuuid.New()
	tx.Context.SetLabel("secretIDHash", hash(secretID))
	secretEncrypted, encryptErr := encrypt(s.Text, secretID)

	if encryptErr != nil {
		tx.Context.SetLabel("errorCode", "encrypt_error")
		log.Errorf("error storing secretID %v in redis: %v", secretID, encryptErr)
		return encryptErr
	}

	// Store the secret in Redis
	storeErr := r.Set(hash(secretID), secretEncrypted, 0).Err()
	if storeErr != nil {
		tx.Context.SetLabel("errorCode", "redis_set_error")
		log.Errorf("error storing secretID %v in redis: %v", secretID, storeErr)
		return storeErr
	}

	secretResponse := sl.Message{
		Msg: sl.Msg{
			ResponseType:   sl.ResponseTypeInChannel,
			DeleteOriginal: true,
			Attachments: []sl.Attachment{{
				Title:      fmt.Sprintf("%v sent a secret message", s.UserName),
				Fallback:   fmt.Sprintf("%v sent a secret message", s.UserName),
				CallbackID: fmt.Sprintf("send_secret:%v", secretID),
				Color:      "#6D5692",
				Actions: []sl.AttachmentAction{{
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
	sendMessageErr := slack.SendMessage(c.Request.Context(), s.ResponseURL, secretResponse)
	if sendMessageErr != nil {
		sendSpan.Context.SetLabel("errorCode", "send_message_error")
		log.Errorf("error sending secret to slack: %v", sendMessageErr)
		return sendMessageErr
	}

	return nil

}

func SlashSecret(c *gin.Context, tx *apm.Transaction, s sl.SlashCommand) {

	tx.Context.SetLabel("userHash", hash(s.UserID))
	tx.Context.SetLabel("teamHash", hash(s.TeamID))
	tx.Context.SetLabel("action", "createSecret")
	tx.Context.SetLabel("slashCommand", "/secret")

	// Handle if no input was given
	if s.Text == "" {
		res, code := slack.NewSlackErrorResponse(
			"Error: secret text is empty",
			"It looks like you tried to send a secret but forgot to provide the secret's text. You can send a secret like this: `/secret I am scared of heights`",
			"secret_text_empty")
		tx.Context.SetLabel("errorCode", "text_empty")
		c.Data(code, gin.MIMEJSON, res)
		return
	}

	// Prepare and send message to channel using response_url link
	err := PrepareAndSendSecret(c, tx, s)
	if err != nil {
		res, code := slack.NewSlackErrorResponse(
			":x: Sorry, an error occurred",
			"An error occurred attempting to create secret",
			"encrypt_error")
		tx.Context.SetLabel("errorCode", "send_secret_payload_error")
		c.Data(code, gin.MIMEJSON, res)
		return
	}

	// Send empty Ack to Slack if we got here without errors
	c.Data(http.StatusOK, gin.MIMEPlain, nil)

	r := apmgoredis.Wrap(redis.GetRedisClient()).WithContext(c.Request.Context())
	responseEphemeral := sl.Message{
		Msg: sl.Msg{
			ResponseType: sl.ResponseTypeEphemeral,
			Text:         fmt.Sprintf(":wave: Hey, we're working hard updating Secret Message. In order to keep using the app, <%v/auth/slack|please click here to reinstall>", config.AppURL),
		},
	}
	_, err = r.HGet(s.TeamID, "access_token").Result()
	if err != nil {
		log.Warnf("Did not find access_token for team %v in redis... sending the please reinstall message now.", s.TeamID)
		//User needs to reinstall the app, send them a message about that now
		sendMessageEphemeralErr := slack.SendMessage(c.Request.Context(), s.ResponseURL, responseEphemeral)
		if sendMessageEphemeralErr != nil {
			log.Errorf("error sending ephemeral message to slack: %v", sendMessageEphemeralErr)
		}
	}

	return
}
