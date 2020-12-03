package secretmessage

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid"
	"github.com/prometheus/common/log"
	"github.com/slack-go/slack"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmgoredis"
)

func SlashSecret(c *gin.Context, tx *apm.Transaction, s slack.SlashCommand) {

	r := apmgoredis.Wrap(GetRedisClient()).WithContext(c.Request.Context())
	tx.Context.SetLabel("userHash", hash(s.UserID))
	tx.Context.SetLabel("teamHash", hash(s.TeamID))
	tx.Context.SetLabel("action", "createSecret")
	tx.Context.SetLabel("slashCommand", "/secret")

	// Handle if no input was given
	if s.Text == "" {
		res, code := NewSlackErrorResponse(
			"Error: secret text is empty",
			"It looks like you tried to send a secret but forgot to provide the secret's text. You can send a secret like this: `/secret I am scared of heights`",
			"secret_text_empty")
		tx.Context.SetLabel("errorCode", "text_empty")
		c.Data(code, gin.MIMEJSON, res)
		return
	}

	// Create and Encrypt the secret
	secretID := shortuuid.New()
	tx.Context.SetLabel("secretIDHash", hash(secretID))
	secretEncrypted, encryptErr := encrypt(s.Text, secretID)
	if encryptErr != nil {
		tx.Context.SetLabel("errorCode", "encrypt_error")
		log.Errorf("error storing secretID %v in redis: %v", secretID, encryptErr)
		res, code := NewSlackErrorResponse(
			":x: Sorry, an error occurred",
			"An error occurred attempting to create secret",
			"encrypt_error")
		c.Data(code, gin.MIMEJSON, res)
		return
	}

	// Store the secret in Redis
	storeErr := r.Set(hash(secretID), secretEncrypted, 0).Err()
	if storeErr != nil {
		tx.Context.SetLabel("errorCode", "redis_set_error")
		log.Errorf("error storing secretID %v in redis: %v", secretID, storeErr)
		res, code := NewSlackErrorResponse(
			":x: Sorry, an error occurred",
			"An error occurred attempting to create secret",
			"store_error")
		c.Data(code, gin.MIMEJSON, res)
		return
	}

	// Send the envelope to Slack
	sendSpan := tx.StartSpan("send_message", "client_request", nil)
	defer sendSpan.End()
	response := slack.Message{
		Msg: slack.Msg{
			ResponseType:   slack.ResponseTypeInChannel,
			DeleteOriginal: true,
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
	sendMessageErr := SendMessage(c.Request.Context(), s.ResponseURL, response)
	if sendMessageErr != nil {
		sendSpan.Context.SetLabel("errorCode", "send_message_error")
		log.Errorf("error sending secret to slack: %v", sendMessageErr)
		res, code := NewSlackErrorResponse(
			":x: Sorry, an error occurred",
			"An error occurred attempting to create secret",
			"send_message_error")
		c.Data(code, gin.MIMEJSON, res)
		return
	}

	responseEphemeral := slack.Message{
		Msg: slack.Msg{
			ResponseType:   slack.ResponseTypeEphemeral,
			DeleteOriginal: true,
			Text:           ":wave: Hey, we're working hard updating Secret Message. In order to keep using the app, <https://slacksecret.herokuapp.com/auth/slack|please click here to reinstall>",
		},
	}
	// Send the empty Ack to Slack if everything is gucci
	c.Data(http.StatusOK, gin.MIMEPlain, nil)

	teamAccessToken, err := r.HGet(s.TeamID, "access_token").Result()
	if err != nil || teamAccessToken == "" {
		//User needs to reinstall the app, send them a message about that now
		sendMessageEphemeralErr := SendMessage(c.Request.Context(), s.ResponseURL, responseEphemeral)
		if sendMessageErr != nil {
			sendSpan.Context.SetLabel("errorCode", "send_ephemeral_message_error")
			log.Errorf("error sending ephemeral message to slack: %v", sendMessageEphemeralErr)
		}
	}

	return
}
