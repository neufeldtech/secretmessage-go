package secretmessage

import (
	"crypto/rand"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

func (ctl *PublicController) HandleOauthBegin(c *gin.Context) {
	state := rand.Text()
	url := ctl.config.OauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)

	c.SetCookie("state", state, 0, "", "", false, true)
	c.Redirect(302, url)
}
