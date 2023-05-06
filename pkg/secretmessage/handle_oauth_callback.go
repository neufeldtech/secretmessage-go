package secretmessage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"go.elastic.co/apm"
	"golang.org/x/net/context"
)

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
	r := token.Extra("raw")
	b, _ := json.Marshal(r)
	fmt.Printf("%+v", string(b))

	teamMap, ok := token.Extra("team").(map[string]interface{})
	if !ok {
		log.Errorf("error unmarshalling team from token: %v", token)
		apm.CaptureError(hc, fmt.Errorf("could not unmarshal team from token: %v", token)).Send()
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "token_team_unmarshal_error")
		return
	}

	teamID, ok := teamMap["id"].(string)
	if !ok || teamID == "" {
		log.Errorf("error unmarshalling teamID from token: %v", token)
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "token_team_unmarshal_error")
		return
	}

	teamName, ok := teamMap["name"].(string)
	if !ok || teamName == "" {
		log.Errorf("error unmarshalling teamName from token: %v", token)
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "token_team_unmarshal_error")
		return
	}

	scope, ok := token.Extra("scope").(string)
	if !ok || scope == "" {
		log.Errorf("error unmarshalling scope from token: %v", token)
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "token_team_unmarshal_error")
		return
	}

	var team Team
	updateTeamErr := ctl.db.
		WithContext(hc).
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
