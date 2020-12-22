package secretmessage

import (
	"github.com/gin-gonic/gin"
	"go.elastic.co/apm/module/apmgin"
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

func (ctl *PublicController) ConfigureRoutes() *gin.Engine {

	r := gin.Default()
	r.Use(apmgin.Middleware(r))

	r.GET("/health", ctl.HandleHealth)

	r.GET("/auth/slack", ctl.HandleOauthBegin)
	r.GET("/auth/slack/callback", ctl.HandleOauthCallback)

	// Signature validation required
	r.POST("/slash", ValidateSignature(ctl.config), ctl.HandleSlash)
	r.POST("/interactive", ValidateSignature(ctl.config), ctl.HandleInteractive)

	return r
}
