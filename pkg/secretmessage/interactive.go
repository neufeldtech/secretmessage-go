package secretmessage

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/neufeldtech/secretmessage-go/pkg/secretmessage/actions"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func CallbackReadSecret(ctl *PublicController, c *gin.Context, i slack.InteractionCallback) {
	hc := c.Request.Context()
	secretID := strings.ReplaceAll(i.CallbackID, fmt.Sprintf("%s:", actions.ReadMessage), "")
	// Fetch secret
	var secret Secret
	getSecretErr := ctl.db.WithContext(hc).Where("id = ?", hash(secretID)).First(&secret).Error
	var errTitle string
	var errMsg string
	var errCallback string
	var deleteOriginal bool
	switch {
	case !secret.ExpiresAt.IsZero() && secret.ExpiresAt.Before(time.Now()):
		getSecretErr = errors.New("Secret expired")
		errTitle = ":hourglass: Secret expired"
		errMsg = "This Secret has expired"
		errCallback = "secret_expired"
		deleteOriginal = true
		ctl.db.WithContext(hc).Unscoped().Where("id = ?", hash(secretID)).Delete(Secret{})
	case getSecretErr == gorm.ErrRecordNotFound:
		errTitle = ":question: Secret not found"
		errMsg = "This Secret has already been retrieved or has expired"
		errCallback = "secret_not_found"
		deleteOriginal = true
	case getSecretErr != nil:
		errTitle = ":x: Sorry, an error occurred"
		errMsg = "An error occurred attempting to retrieve secret"
		errCallback = "secret_get_error"
		deleteOriginal = false
	}
	if getSecretErr != nil {
		ctl.logger.Error("error retrieving secret from store", zap.Error(getSecretErr), zap.String("secretID", secretID))
		res, code := ctl.slackService.NewSlackErrorResponse(
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
	secretDecrypted, decryptionErr = decrypt(secret.Value, secretID)
	if decryptionErr != nil {
		ctl.logger.Error("error decrypting secret", zap.Error(decryptionErr), zap.String("secretID", secretID))
		res, code := ctl.slackService.NewSlackErrorResponse(
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
				CallbackID: fmt.Sprintf("%s:%v", actions.DeleteMessage, secretID),
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
		ctl.logger.Error("error marshalling response", zap.Error(err), zap.String("secretID", secretID))
		res, code := ctl.slackService.NewSlackErrorResponse(
			":x: Sorry, an error occurred",
			"An error occurred attempting to retrieve secret",
			false,
			"json_marshal_error")
		c.Data(code, gin.MIMEJSON, res)
		return
	}
	c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)

	if delSecretErr := ctl.db.WithContext(hc).Unscoped().Where("id = ?", hash(secretID)).Delete(Secret{}).Error; delSecretErr != nil {
		ctl.logger.Error("error deleting secret after retrieval", zap.Error(delSecretErr), zap.String("secretID", secretID))
	}
}

func CallbackDeleteSecret(ctl *PublicController, c *gin.Context, i slack.InteractionCallback) {
	secretID := strings.ReplaceAll(i.CallbackID, fmt.Sprintf("%s:", actions.DeleteMessage), "")

	response := slack.Message{
		Msg: slack.Msg{
			DeleteOriginal: true,
		},
	}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		ctl.logger.Error("error marshalling response for delete secret", zap.Error(err), zap.String("secretID", secretID))
		res, code := ctl.slackService.NewSlackErrorResponse(
			":x: Sorry, an error occurred",
			"An error occurred attempting to delete secret",
			false,
			"json_marshal_error")
		c.Data(code, gin.MIMEJSON, res)
		return
	}
	c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
}

func CallbackViewSubmission(ctl *PublicController, c *gin.Context, i slack.InteractionCallback) {

	secretTextVal := i.View.State.Values["secret_text_input"]["secret_text_input"].Value
	datePickerVal := i.View.State.Values["expiry_date_input"]["expiry_date_input"].SelectedDate

	dateParsed, err := time.Parse("2006-01-02", datePickerVal)
	if err != nil {
		ctl.logger.Error("error parsing date from view submission", zap.Error(err), zap.String("datePickerVal", datePickerVal))
	}

	err = PrepareAndSendSecretEnvelope(ctl, c, secretTextVal, i.Team.ID, i.User.Name, i.View.PrivateMetadata, WithExpiryDate(dateParsed))
	if err != nil {
		ctl.logger.Error("error preparing and sending secret envelope", zap.Error(err), zap.String("secretTextVal", secretTextVal), zap.String("teamID", i.Team.ID), zap.String("userName", i.User.Name), zap.String("privateMetadata", i.View.PrivateMetadata))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "Error with the stuffs"})

		return
	}
	c.Data(http.StatusOK, gin.MIMEPlain, nil)
}
