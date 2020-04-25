package middleware

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/nlopes/slack"
	"github.com/prometheus/common/log"
)

var (
	signingSecret, _ = os.LookupEnv("SLACK_SIGNING_SECRET")
)

func ValidateSignature() gin.HandlerFunc {
	return func(c *gin.Context) {
		verifier, err := slack.NewSecretsVerifier(c.Request.Header, signingSecret)
		if err != nil {
			log.Errorf("error verifying signature: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "Error verifying secret"})
			return
		}
		c.Request.Body = ioutil.NopCloser(io.TeeReader(c.Request.Body, &verifier))
		_, err = slack.SlashCommandParse(c.Request)
		if err != nil {
			log.Error(err)
			return
		}
		if err = verifier.Ensure(); err != nil {
			log.Error(err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "Signature not valid"})
			return
		}
		c.Next()
	}
}
