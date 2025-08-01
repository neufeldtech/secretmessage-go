package secretmessage

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid"
	"github.com/neufeldtech/secretmessage-go/pkg/secretmessage/actions"
	"github.com/slack-go/slack"
	"go.elastic.co/apm"
	"go.uber.org/zap"
)

// PrepareAndSendSecretEnvelope encrypts the secret, stores in db, and sends the 'envelope' back to slack
func PrepareAndSendSecretEnvelope(ctl *PublicController, c *gin.Context, tx *apm.Transaction, secretText string, TeamID string, UserName string, ResponseUrl string, options ...SecretOption) error {
	hc := c.Request.Context()

	secretID := shortuuid.New()
	tx.Context.SetLabel("secretIDHash", hash(secretID))
	secretEncrypted, encryptErr := encrypt(secretText, secretID)

	if encryptErr != nil {
		tx.Context.SetLabel("errorCode", "encrypt_error")
		ctl.logger.Error("error encrypting secret", zap.Error(encryptErr), zap.String("secretID", secretID))
		return encryptErr
	}

	sec := NewSecret(hash(secretID), secretEncrypted, options...)
	// Store the secret
	storeErr := ctl.db.WithContext(hc).Create(sec).Error

	if storeErr != nil {
		tx.Context.SetLabel("errorCode", "db_store_error")
		ctl.logger.Error("error storing secret in database", zap.Error(storeErr), zap.String("secretID", secretID))
		return storeErr
	}

	footerMsg := fmt.Sprintf("Message expires <!date^%d^{date_pretty}|%s>", sec.ExpiresAt.Unix(), sec.ExpiresAt.Format("2006-01-02 15:04 MST"))

	secretResponse := slack.Message{
		Msg: slack.Msg{
			ResponseType: slack.ResponseTypeInChannel,
			Attachments: []slack.Attachment{{
				Title:      fmt.Sprintf("%v sent a secret message", UserName),
				Fallback:   fmt.Sprintf("%v sent a secret message", UserName),
				CallbackID: fmt.Sprintf("%s:%v", actions.ReadMessage, secretID),
				Color:      "#6D5692",
				Footer:     footerMsg,
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
	sendMessageErr := ctl.slackService.SendResponseUrlMessage(hc, ResponseUrl, secretResponse)
	if sendMessageErr != nil {
		sendSpan.Context.SetLabel("errorCode", "send_message_error")
		ctl.logger.Error("error sending secret to slack", zap.Error(sendMessageErr), zap.String("secretID", secretID))
		return sendMessageErr
	}

	return nil

}

// PromptCreateSecretModal encrypts the secret, stores in db, and sends the 'envelope' back to slack
func PromptCreateSecretModal(ctl *PublicController, c *gin.Context, tx *apm.Transaction, s slack.SlashCommand) error {

	datePicker := slack.NewDatePickerBlockElement("expiry_date_input")
	datePicker.InitialDate = time.Now().AddDate(0, 0, 7).Format("2006-01-02")

	textInput := slack.NewPlainTextInputBlockElement(slack.NewTextBlockObject("plain_text", "Enter your secret...", false, false), "secret_text_input")
	textInput.Multiline = true
	modalRequest := slack.ModalViewRequest{
		Type:            slack.VTModal,
		Title:           slack.NewTextBlockObject("plain_text", "Send a Secret", false, false),
		Close:           slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		Submit:          slack.NewTextBlockObject("plain_text", "Send", false, false),
		PrivateMetadata: s.ResponseURL,
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				slack.NewInputBlock(
					"secret_text_input",
					slack.NewTextBlockObject("plain_text", "Secret Text", false, false),
					slack.NewTextBlockObject("plain_text", "Max 10,000 characters", false, false),
					textInput,
				),
				slack.NewInputBlock(
					"expiry_date_input",
					slack.NewTextBlockObject("plain_text", "Secret Expiry", false, false),
					slack.NewTextBlockObject("plain_text", "Expiry date is limited to a maximum of 30 days from today", false, false),
					datePicker,
				),
			},
		},
	}

	team := Team{}

	getTeamErr := ctl.db.Where(Team{ID: s.TeamID}).First(&team).Error
	if getTeamErr != nil {
		ctl.logger.Error("error getting team for slash command", zap.Error(getTeamErr), zap.String("teamID", s.TeamID))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "Error with the stuffs"})
		tx.Context.SetLabel("errorCode", "team_not_found")
		return getTeamErr
	}

	api := ctl.slackService.GetSlackClient(team.AccessToken)

	_, err := api.OpenView(s.TriggerID, modalRequest)

	if err != nil {
		ctl.logger.Error("error opening modal for slash command", zap.Error(err), zap.String("teamID", s.TeamID), zap.String("triggerID", s.TriggerID))
		return err
	}

	return nil
}

// SlashSecret is the main entrypoint for the slash command /secret
func SlashSecret(ctl *PublicController, c *gin.Context, tx *apm.Transaction, s slack.SlashCommand) {

	tx.Context.SetLabel("userHash", hash(s.UserID))
	tx.Context.SetLabel("teamHash", hash(s.TeamID))
	tx.Context.SetLabel("action", "createSecret")
	tx.Context.SetLabel("slashCommand", "/secret")

	var err error
	switch {
	case strings.TrimSpace(s.Text) == "":
		// If user provided no text, prompt them with modal
		err = PromptCreateSecretModal(ctl, c, tx, s)
	default:
		// If user provided text inline, do the old behaviour
		err = PrepareAndSendSecretEnvelope(ctl, c, tx, s.Text, s.TeamID, s.UserName, s.ResponseURL)
	}
	if err != nil {
		ctl.logger.Error("error processing slash command", zap.Error(err))
		res, code := ctl.slackService.NewSlackErrorResponse(
			":x: Sorry, an error occurred",
			"An error occurred",
			false,
			"create_secret_error")
		c.Data(code, gin.MIMEJSON, res)
		c.Abort()
		return
	}
	// Send empty Ack to Slack if we got here without errors
	c.Data(http.StatusOK, gin.MIMEPlain, nil)

	if AppReinstallNeeded(ctl, c, tx, s) {
		SendReinstallMessage(ctl, c, tx, s)
	}
}

func AppReinstallNeeded(ctl *PublicController, c *gin.Context, tx *apm.Transaction, s slack.SlashCommand) bool {
	var team Team
	hc := c.Request.Context()
	err := ctl.db.WithContext(hc).Where("id = ?", s.TeamID).First(&team).Error
	if err != nil || team.AccessToken == "" {
		ctl.logger.Warn("App reinstall needed", zap.String("teamID", s.TeamID), zap.Error(err))
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
	sendMessageEphemeralErr := ctl.slackService.SendResponseUrlMessage(c.Request.Context(), s.ResponseURL, responseEphemeral)
	if sendMessageEphemeralErr != nil {
		ctl.logger.Error("error sending ephemeral reinstall message", zap.Error(sendMessageEphemeralErr), zap.String("teamID", s.TeamID))
	}
}
