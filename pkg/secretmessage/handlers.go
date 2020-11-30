package secretmessage

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid"
	"github.com/prometheus/common/log"
	"github.com/slack-go/slack"
	"go.elastic.co/apm"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

func HandleSlash(c *gin.Context) {
	tx := apm.TransactionFromContext(c.Request.Context())
	s, err := slack.SlashCommandParse(c.Request)
	if err != nil {
		log.Errorf("error parsing slash command: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": "Bad Request"})
		tx.Context.SetLabel("errorCode", "slash_payload_parse_error")
		return
	}
	switch s.Command {
	case "/secret":
		SlashSecret(c, tx, s)
	default:
		c.Data(http.StatusOK, gin.MIMEPlain, nil)
	}
	return
}

func HandleOauthBegin(c *gin.Context) {
	state := shortuuid.New()
	url := GetConfig().OauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)

	c.SetCookie("state", state, 0, "", "", false, true)
	c.Redirect(302, url)
}

func HandleOauthCallback(c *gin.Context) {
	tx := apm.TransactionFromContext(c.Request.Context())

	stateQuery := c.Query("state")
	conf := GetConfig()
	stateCookie, err := c.Cookie("state")
	if err != nil {
		log.Errorf("error retrieving state cookie from request: %v", err)
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "state_cookie_parse_error")
		return
	}
	if stateCookie != stateQuery {
		log.Error("error validating state cookie with state query param")
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "state_cookie_invalid")
		return
	}

	_, err = conf.OauthConfig.Exchange(context.Background(), c.Query("code"))
	if err != nil {
		log.Errorf("error retrieving initial oauth token: %v", err)
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "oauth_token_exchange_error")
		return
	}

	c.Redirect(302, "https://secretmessage.xyz/success")
}

func HandleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

func HandleInteractive(c *gin.Context) {
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
	tx.Context.SetLabel("teamHash", hash(i.User.TeamID))
	callbackType := strings.Split(i.CallbackID, ":")[0]
	switch callbackType {
	case "send_secret":
		CallbackSendSecret(tx, c, i)
	case "delete_secret":
		CallbackDeleteSecret(tx, c, i)
	default:
		log.Error("Hit the default case. bad things happened")
		c.Data(http.StatusInternalServerError, gin.MIMEPlain, nil)
	}
}
