package secretmessage

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/log"
	"go.elastic.co/apm/module/apmgin"
)

func callHealth(url string) error {
	resp, err := http.Get(url + "/health")
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
		time.Sleep(5 * time.Minute)
		err := callHealth(config.AppURL)
		if err != nil {
			log.Error(err)
		}
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
