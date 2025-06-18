package secretmessage

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

func (ctl *PublicController) ValidateSignature() gin.HandlerFunc {
	return func(c *gin.Context) {
		if ctl.config.SkipSignatureValidation {
			ctl.logger.Warn("Signature validation is disabled. This is not recommended.")
			return
		}
		verifier, err := slack.NewSecretsVerifier(c.Request.Header, ctl.config.SigningSecret)
		if err != nil {
			ctl.logger.Error("error verifying signature", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "Error verifying secret"})
			return
		}

		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			ctl.logger.Error("error reading request body during validateSignature", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "Error verifying secret"})
			return
		}

		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		_, err = verifier.Write(body)
		if err != nil {
			ctl.logger.Error("error writing to verifier", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "Error verifying secret"})
			return
		}

		if err = verifier.Ensure(); err != nil {
			ctl.logger.Error("signature validation failed", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "Signature not valid"})
			return
		}
		c.Next()
	}
}
