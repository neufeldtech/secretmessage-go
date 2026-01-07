package secretmessage

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/neufeldtech/secretmessage-go/pkg/secretmessage/actions"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

func (ctl *PublicController) HandleInteractive(c *gin.Context) {

	var err error
	var i slack.InteractionCallback
	payload := c.PostForm("payload")
	err = json.Unmarshal([]byte(payload), &i)
	if err != nil {
		ctl.logger.Error("error parsing interaction payload", zap.Error(err), zap.String("payload", payload))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "Error with the stuffs"})

		return
	}

	switch i.Type {
	case slack.InteractionTypeViewSubmission:
		CallbackViewSubmission(ctl, c, i)
	default:
		callbackType := strings.Split(i.CallbackID, ":")[0]
		switch callbackType {
		case actions.ReadMessage:
			CallbackReadSecret(ctl, c, i)
		case actions.DeleteMessage:
			CallbackDeleteSecret(ctl, c, i)
		default:
			ctl.logger.Error("unknown interaction type", zap.String("type", string(i.Type)), zap.String("callbackID", i.CallbackID))
			c.Data(http.StatusInternalServerError, gin.MIMEPlain, nil)
		}
	}
}
