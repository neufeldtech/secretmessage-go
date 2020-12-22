package secretmessage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid"
	"github.com/prometheus/common/log"
	"github.com/slack-go/slack"
	"go.elastic.co/apm"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type PublicController struct {
	db     *gorm.DB
	config Config
}

func NewController(config Config, db *gorm.DB) *PublicController {
	return &PublicController{
		db:     db,
		config: config,
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
	url := ctl.config.OauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)

	c.SetCookie("state", state, 0, "", "", false, true)
	c.Redirect(302, url)
}

func (ctl *PublicController) HandleOauthCallback(c *gin.Context) {
	hc := c.Request.Context()
	tx := apm.TransactionFromContext(c.Request.Context())

	tx.Context.SetLabel("slackOauthVersion", "v2")
	tx.Context.SetLabel("action", "handleOauthCallback")

	stateQuery := c.Query("state")
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
	token, err := ctl.config.OauthConfig.Exchange(context.Background(), c.Query("code"))
	if err != nil {
		log.Errorf("error retrieving initial oauth token: %v", err)
		apm.CaptureError(hc, err).Send()
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "oauth_token_exchange_error")
		return
	}

	teamMap, ok := token.Extra("team").(map[string]interface{})
	if !ok {
		log.Errorf("error unmarshalling team from token: %v", token)
		apm.CaptureError(hc, fmt.Errorf("could not unmarshal team from token: %v", token)).Send()
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "token_team_unmarshal_error")
		return
	}

	teamID, ok := teamMap["id"].(string)
	if !ok {
		log.Errorf("error unmarshalling teamID from token: %v", token)
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "token_team_unmarshal_error")
		return
	}

	teamName, ok := teamMap["name"].(string)
	if !ok {
		log.Errorf("error unmarshalling teamName from token: %v", token)
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "token_team_unmarshal_error")
		return
	}

	scope, ok := token.Extra("scope").(string)
	if !ok {
		log.Errorf("error unmarshalling scope from token: %v", token)
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "token_team_unmarshal_error")
		return
	}

	var team Team
	updateTeamErr := ctl.db.
		Where(&team, Team{ID: teamID}).
		// Attrs() is for setting fields on new records
		Attrs(Team{Paid: sql.NullBool{Bool: false, Valid: true}}).
		// Assign() is for updating fields on all records
		Assign(Team{AccessToken: token.AccessToken, Scope: scope, Name: teamName}).
		FirstOrCreate(&team).Error

	if updateTeamErr != nil {
		log.Errorf("error updating team in db: %v", err)
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "team_update_error")
		return
	}

	c.Redirect(302, "https://secretmessage.xyz/success")
}

func (ctl *PublicController) HandleHealth(c *gin.Context) {
	// err := ctl.db.Ping()
	// if err != nil {
	// 	log.Error(err)
	// 	c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN"})
	// 	return
	// }
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
