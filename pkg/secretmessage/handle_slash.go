package secretmessage

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

func (ctl *PublicController) HandleSlash(c *gin.Context) {
	s, err := slack.SlashCommandParse(c.Request)
	if err != nil {
		ctl.logger.Error("error parsing slash command", zap.Error(err), zap.String("command", c.Request.URL.Path))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": "Bad Request"})
		return
	}
	switch s.Command {
	case "/secret":
		SlashSecret(ctl, c, s)
	default:
		c.Data(http.StatusOK, gin.MIMEPlain, nil)
	}
	return
}
