package secretmessage

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/log"
	"github.com/slack-go/slack"
)

func ValidateSignature(config Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.SkipSignatureValidation {
			log.Warn("SIGNATURE VALIDATION IS DISABLED. THIS IS NOT RECOMMENDED")
			return
		}
		log.Info(c.Request.Header)
		verifier, err := slack.NewSecretsVerifier(c.Request.Header, config.SigningSecret)
		if err != nil {
			log.Errorf("error verifying signature: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "Error verifying secret"})
			return
		}

		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			log.Errorf("error verifying signature: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "Error verifying secret"})
			return
		}

		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		_, err = verifier.Write(body)
		if err != nil {
			log.Errorf("error verifying signature: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "Error verifying secret"})
			return
		}

		if err = verifier.Ensure(); err != nil {
			log.Error(err)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "Signature not valid"})
			return
		}
		c.Next()
	}
}
