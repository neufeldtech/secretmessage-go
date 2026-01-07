package secretmessage

import (
	"database/sql"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

func (ctl *PublicController) HandleOauthCallback(c *gin.Context) {
	hc := c.Request.Context()

	stateQuery := c.Query("state")
	stateCookie, err := c.Cookie("state")
	if err != nil {
		ctl.logger.Error("error retrieving state cookie from request", zap.Error(err), zap.String("stateQuery", stateQuery))
		c.Redirect(302, "https://secretmessage.xyz/error")
		return
	}
	if stateCookie != stateQuery {
		ctl.logger.Error("error validating state cookie with state query param", zap.String("stateCookie", stateCookie), zap.String("stateQuery", stateQuery))
		c.Redirect(302, "https://secretmessage.xyz/error")
		return
	}
	token, err := ctl.config.OauthConfig.Exchange(context.Background(), c.Query("code"))
	if err != nil {
		ctl.logger.Error("error retrieving initial oauth token", zap.Error(err))
		c.Redirect(302, "https://secretmessage.xyz/error")
		return
	}
	r := token.Extra("raw")
	b, _ := json.Marshal(r)
	ctl.logger.Sugar().Infof("%+v", string(b))

	teamMap, ok := token.Extra("team").(map[string]interface{})
	if !ok {
		ctl.logger.Error("error unmarshalling team from token", zap.Any("token", token))
		c.Redirect(302, "https://secretmessage.xyz/error")
		return
	}

	teamID, ok := teamMap["id"].(string)
	if !ok || teamID == "" {
		ctl.logger.Error("error unmarshalling teamID from token", zap.Any("token", token))
		c.Redirect(302, "https://secretmessage.xyz/error")
		return
	}

	teamName, ok := teamMap["name"].(string)
	if !ok || teamName == "" {
		ctl.logger.Error("error unmarshalling teamName from token", zap.Any("token", token))
		c.Redirect(302, "https://secretmessage.xyz/error")
		return
	}

	scope, ok := token.Extra("scope").(string)
	if !ok || scope == "" {
		ctl.logger.Error("error unmarshalling scope from token", zap.Any("token", token))
		c.Redirect(302, "https://secretmessage.xyz/error")
		return
	}

	var team Team
	updateTeamErr := ctl.db.
		WithContext(hc).
		Where(&team, Team{ID: teamID}).
		Attrs(Team{Paid: sql.NullBool{Bool: false, Valid: true}}).
		Assign(Team{AccessToken: token.AccessToken, Scope: scope, Name: teamName}).
		FirstOrCreate(&team).Error

	if updateTeamErr != nil {
		ctl.logger.Error("error updating team in db", zap.Error(updateTeamErr))
		c.Redirect(302, "https://secretmessage.xyz/error")
		return
	}

	c.Redirect(302, "https://secretmessage.xyz/success")
}
