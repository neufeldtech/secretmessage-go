package secretmessage

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/log"
	"github.com/slack-go/slack"
	"go.elastic.co/apm"
)

func (ctl *PublicController) HandleInteractive(c *gin.Context) {
	tx := apm.TransactionFromContext(c.Request.Context())
	var err error
	var i slack.InteractionCallback
	payload := c.PostForm("payload")
	err = json.Unmarshal([]byte(payload), &i)
	if err != nil {
		log.Error(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "Error with the stuffs"})
		tx.Context.SetLabel("errorCode", "interaction_payload_parse_error")
		return
	}
	tx.Context.SetLabel("userHash", hash(i.User.ID))
	tx.Context.SetLabel("teamHash", hash(i.Team.ID))
	callbackType := strings.Split(i.CallbackID, ":")[0]
	switch callbackType {
	case "send_secret":
		CallbackSendSecret(ctl, tx, c, i)
	case "delete_secret":
		CallbackDeleteSecret(ctl, tx, c, i)
	default:
		log.Error("Hit the default case. bad things happened")
		c.Data(http.StatusInternalServerError, gin.MIMEPlain, nil)
	}
}
