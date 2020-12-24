package secretmessage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/neufeldtech/secretmessage-go/pkg/secretslack"
	"github.com/prometheus/common/log"
	"github.com/slack-go/slack"
	"go.elastic.co/apm"
	"gorm.io/gorm"
)

func CallbackSendSecret(ctl *PublicController, tx *apm.Transaction, c *gin.Context, i slack.InteractionCallback) {
	hc := c.Request.Context()
	tx.Context.SetLabel("callbackID", "send_secret")
	tx.Context.SetLabel("action", "sendSecret")

	secretID := strings.ReplaceAll(i.CallbackID, "send_secret:", "")
	tx.Context.SetLabel("secretIDHash", hash(secretID))

	// Fetch secret

	var secret Secret
	getSecretErr := ctl.db.WithContext(hc).Where("id = ?", hash(secretID)).First(&secret).Error
	var errTitle string
	var errMsg string
	var errCallback string
	var deleteOriginal bool
	switch {
	case getSecretErr == gorm.ErrRecordNotFound:
		tx.Context.SetLabel("errorCode", "secret_not_found")
		errTitle = ":question: Secret not found"
		errMsg = "This Secret has already been retrieved or has expired"
		errCallback = "secret_not_found"
		deleteOriginal = true
	case getSecretErr != nil:
		tx.Context.SetLabel("errorCode", "secret_get_error")
		errTitle = ":x: Sorry, an error occurred"
		errMsg = "An error occurred attempting to retrieve secret"
		errCallback = "secret_get_error"
		deleteOriginal = false
	}
	if getSecretErr != nil {
		log.Errorf("error retrieving secret from store: %v", getSecretErr)
		res, code := secretslack.NewSlackErrorResponse(
			errTitle,
			errMsg,
			deleteOriginal,
			errCallback)
		c.Data(code, gin.MIMEJSON, res)
		return
	}

	// Decrypt the secret
	var secretDecrypted string
	var decryptionErr error
	if strings.Contains(secret.Value, ":") {
		secretDecrypted, decryptionErr = decryptIV(secret.Value, config.LegacyCryptoKey)
	} else {
		secretDecrypted, decryptionErr = decrypt(secret.Value, secretID)
	}
	if decryptionErr != nil {
		log.Errorf("error decrypting secretID %v: %v", secretID, decryptionErr)
		tx.Context.SetLabel("errorCode", "decrypt_error")
		res, code := secretslack.NewSlackErrorResponse(
			":x: Sorry, an error occurred",
			"An error occurred attempting to retrieve secret",
			false,
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
				Footer:     "The above message is only visible to you and will disappear when your Slack client reloads. To remove it immediately, press the delete button",
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
			false,
			"json_marshal_error")
		c.Data(code, gin.MIMEJSON, res)
		return
	}
	c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)

	if delSecretErr := ctl.db.WithContext(hc).Unscoped().Where("id = ?", hash(secretID)).Delete(Secret{}).Error; delSecretErr != nil {
		log.Error(delSecretErr)
	}
	return
}

func CallbackDeleteSecret(ctl *PublicController, tx *apm.Transaction, c *gin.Context, i slack.InteractionCallback) {
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
			false,
			"json_marshal_error")
		c.Data(code, gin.MIMEJSON, res)
		return
	}
	c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
}
