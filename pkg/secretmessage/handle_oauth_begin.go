package secretmessage

import (
	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid"
	"golang.org/x/oauth2"
)

func (ctl *PublicController) HandleOauthBegin(c *gin.Context) {
	state := shortuuid.New()
	url := ctl.config.OauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)

	c.SetCookie("state", state, 0, "", "", false, true)
	c.Redirect(302, url)
}
