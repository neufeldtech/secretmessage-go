package secretmessage

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/neufeldtech/secretmessage-go/pkg/secretredis"
	"github.com/prometheus/common/log"
	"go.elastic.co/apm/module/apmgin"
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
	secretredis.Connect(config.RedisOptions)

	r := gin.Default()
	r.Use(apmgin.Middleware(r))

	r.GET("/health", HandleHealth)

	r.GET("/auth/slack", HandleOauthBegin)
	r.GET("/auth/slack/callback", HandleOauthCallback)

	// Signature validation required
	r.POST("/slash", ValidateSignature(config), HandleSlash)
	r.POST("/interactive", ValidateSignature(config), HandleInteractive)

	return r
}
