package secretmessage

import (
	"github.com/gin-gonic/gin"
)

func SetupRouter(config Config) *gin.Engine {
	InitRedis(config)
	InitSlackClient(config)
	r := gin.Default()

	r.GET("/auth/slack", HandleOauthBegin)
	r.GET("/auth/slack/callback", HandleOauthCallback)

	// Signature validation required
	r.POST("/slash", ValidateSignature(config), HandleSlash)
	r.POST("/interactive", ValidateSignature(config), HandleInteractive)

	return r
}
