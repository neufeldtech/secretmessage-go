package secretmessage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid"
	"github.com/neufeldtech/secretmessage-go/pkg/secretdb"
	"github.com/neufeldtech/secretmessage-go/pkg/secretredis"
	"github.com/prometheus/common/log"
	"github.com/slack-go/slack"
	"go.elastic.co/apm"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

type PublicController struct {
	db               *sql.DB
	secretRepository secretdb.SecretRepository
}

func NewController(db *sql.DB, secretRepository secretdb.SecretRepository) *PublicController {
	return &PublicController{
		db:               db,
		secretRepository: secretRepository,
	}
}

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

func (ctl *PublicController) HandleOauthBegin(c *gin.Context) {
	state := shortuuid.New()
	url := GetConfig().OauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)

	c.SetCookie("state", state, 0, "", "", false, true)
	c.Redirect(302, url)
}

func (ctl *PublicController) HandleOauthCallback(c *gin.Context) {
	hc := c.Request.Context()
	tx := apm.TransactionFromContext(c.Request.Context())
	r := secretredis.Client().WithContext(c.Request.Context())
	tx.Context.SetLabel("slackOauthVersion", "v2")
	tx.Context.SetLabel("action", "handleOauthCallback")

	stateQuery := c.Query("state")
	conf := GetConfig()
	stateCookie, err := c.Cookie("state")
	if err != nil {
		log.Errorf("error retrieving state cookie from request: %v", err)
		apm.CaptureError(hc, err).Send()
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "state_cookie_parse_error")
		return
	}
	if stateCookie != stateQuery {
		log.Error("error validating state cookie with state query param")
		apm.CaptureError(hc, fmt.Errorf("state cookie was invalid")).Send()
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "state_cookie_invalid")
		return
	}
	token, err := conf.OauthConfig.Exchange(context.Background(), c.Query("code"))
	if err != nil {
		log.Errorf("error retrieving initial oauth token: %v", err)
		apm.CaptureError(hc, err).Send()
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "oauth_token_exchange_error")
		return
	}

	team, ok := token.Extra("team").(map[string]interface{})
	if !ok {
		log.Errorf("error unmarshalling team from token: %v", token)
		apm.CaptureError(hc, fmt.Errorf("could not unmarshal team from token: %v", token)).Send()
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "token_team_unmarshal_error")
		return
	}

	teamID, ok := team["id"].(string)
	if !ok {
		log.Errorf("error unmarshalling teamID from token: %v", token)
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "token_team_unmarshal_error")
		return
	}

	teamName, ok := team["name"].(string)
	if !ok {
		log.Errorf("error unmarshalling teamName from token: %v", token)
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "token_team_unmarshal_error")
		return
	}

	fields := map[string]interface{}{
		"access_token": token.AccessToken,
		"name":         teamName,
		"scope":        token.Extra("scope"),
	}
	err = r.HMSet(teamID, fields).Err()
	if err != nil {
		log.Errorf("error setting token in redis: %v", err)
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "token_team_unmarshal_error")
		return
	}
	c.Redirect(302, "https://secretmessage.xyz/success")
}

func (ctl *PublicController) HandleHealth(c *gin.Context) {
	err := ctl.db.Ping()
	if err != nil {
		log.Error(err)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

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
	tx.Context.SetLabel("teamHash", hash(i.User.TeamID))
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
