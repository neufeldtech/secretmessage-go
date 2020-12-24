package secretmessage

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/log"
	"github.com/slack-go/slack"
	"go.elastic.co/apm"
)

func (ctl *PublicController) HandleSlash(c *gin.Context) {
	hc := c.Request.Context()
	tx := apm.TransactionFromContext(hc)
	s, err := slack.SlashCommandParse(c.Request)
	if err != nil {
		log.Errorf("error parsing slash command: %v", err)
		apm.CaptureError(hc, err).Send()
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": "Bad Request"})
		tx.Context.SetLabel("errorCode", "slash_payload_parse_error")
		return
	}
	switch s.Command {
	case "/secret":
		SlashSecret(ctl, c, tx, s)
	default:
		c.Data(http.StatusOK, gin.MIMEPlain, nil)
	}
	return
}
