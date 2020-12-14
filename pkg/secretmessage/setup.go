package secretmessage

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/log"
	"go.elastic.co/apm/module/apmgin"
)

func callHealth() error {
	resp, err := http.Get(config.AppURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func StayAwake(config Config) {
	for {
		err := callHealth()
		if err != nil {
			log.Error(err)
		}
		time.Sleep(5 * time.Minute)
	}
}

func (ctl *PublicController) ConfigureRoutes(config Config) *gin.Engine {

	r := gin.Default()
	r.Use(apmgin.Middleware(r))

	r.GET("/health", ctl.HandleHealth)

	r.GET("/auth/slack", ctl.HandleOauthBegin)
	r.GET("/auth/slack/callback", ctl.HandleOauthCallback)

	// Signature validation required
	r.POST("/slash", ValidateSignature(config), ctl.HandleSlash)
	r.POST("/interactive", ValidateSignature(config), ctl.HandleInteractive)

	return r
}
