package secretmessage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/neufeldtech/secretmessage-go/pkg/secretredis"
	"github.com/neufeldtech/secretmessage-go/pkg/secretslack"
	"github.com/prometheus/common/log"
	"github.com/slack-go/slack"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmgoredis"
)

func CallbackSendSecret(tx *apm.Transaction, c *gin.Context, i slack.InteractionCallback) {
	r := apmgoredis.Wrap(secretredis.Client()).WithContext(c.Request.Context())
	tx.Context.SetLabel("callbackID", "send_secret")
	tx.Context.SetLabel("action", "sendSecret")

	secretID := strings.ReplaceAll(i.CallbackID, "send_secret:", "")
	tx.Context.SetLabel("secretIDHash", hash(secretID))

	// Fetch secret from redis
	secretEncrypted, getSecretErr := r.Get(hash(secretID)).Result()
	if getSecretErr != nil {
		tx.Context.SetLabel("errorCode", "redis_get_error")
		log.Errorf("error retrieving secret from redis: %v", getSecretErr)
		res, code := secretslack.NewSlackErrorResponse(
			":x: Sorry, an error occurred",
			"An error occurred attempting to retrieve secret",
			"redis_get_error")
		c.Data(code, gin.MIMEJSON, res)
		return
	}

	// Decrypt the secret
	var secretDecrypted string
	var decryptionErr error
	if strings.Contains(secretEncrypted, ":") {
		secretDecrypted, decryptionErr = decryptIV(secretEncrypted, config.LegacyCryptoKey)
	} else {
		secretDecrypted, decryptionErr = decrypt(secretEncrypted, secretID)
	}
	if decryptionErr != nil {
		log.Errorf("error retrieving secretID %v from redis: %v", secretID, decryptionErr)
		tx.Context.SetLabel("errorCode", "decrypt_error")
		res, code := secretslack.NewSlackErrorResponse(
			":x: Sorry, an error occurred",
			"An error occurred attempting to retrieve secret",
			"decrypt_error")
		c.Data(code, gin.MIMEJSON, res)
		return
	}

	response := slack.Message{
		Msg: slack.Msg{
			DeleteOriginal: true,
			ResponseType:   slack.ResponseTypeEphemeral,
			Attachments: []slack.Attachment{{
				Title:      "Secret message",
				Fallback:   "Secret message",
				Text:       secretDecrypted,
				CallbackID: fmt.Sprintf("delete_secret:%v", secretID),
				Color:      "#6D5692",
				Footer:     "The above message is only visible to you and will disappear when your Slack client reloads. To remove it immediately, click the button below:",
				Actions: []slack.AttachmentAction{{
					Name:  "removeMessage",
					Text:  ":x: Delete message",
					Type:  "button",
					Style: "danger",
					Value: "removeMessage",
				}},
			}},
		},
	}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Errorf("error marshalling response: %v", err)
		res, code := secretslack.NewSlackErrorResponse(
			":x: Sorry, an error occurred",
			"An error occurred attempting to retrieve secret",
			"json_marshal_error")
		c.Data(code, gin.MIMEJSON, res)
		return
	}
	c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
	r.Del(hash(secretID))
	return
}

func CallbackDeleteSecret(tx *apm.Transaction, c *gin.Context, i slack.InteractionCallback) {
	secretID := strings.ReplaceAll(i.CallbackID, "delete_secret:", "")
	tx.Context.SetLabel("secretIDHash", hash(secretID))
	tx.Context.SetLabel("callbackID", "delete_secret")
	tx.Context.SetLabel("action", "deleteMessage")
	response := slack.Message{
		Msg: slack.Msg{
			DeleteOriginal: true,
		},
	}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Errorf("error marshalling response: %v", err)
		res, code := secretslack.NewSlackErrorResponse(
			":x: Sorry, an error occurred",
			"An error occurred attempting to delete secret",
			"json_marshal_error")
		c.Data(code, gin.MIMEJSON, res)
		return
	}
	c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
}
