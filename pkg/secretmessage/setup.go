package secretmessage

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/log"
)

func StayAwake(config Config) {
	for {
		resp, err := http.Get(config.AppURL + "/health")
		if err != nil {
			log.Error(err)
		}
		defer resp.Body.Close()
		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error(err)
		}
		time.Sleep(5 * time.Minute)
	}
}

func SetupRouter(config Config) *gin.Engine {
	InitRedis(config)
	InitSlackClient(config)
	r := gin.Default()

	r.GET("/health", HandleHealth)

	r.GET("/auth/slack", HandleOauthBegin)
	r.GET("/auth/slack/callback", HandleOauthCallback)

	// Signature validation required
	r.POST("/slash", ValidateSignature(config), HandleSlash)
	r.POST("/interactive", ValidateSignature(config), HandleInteractive)

	return r
}
