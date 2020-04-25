package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nlopes/slack"
	"github.com/prometheus/common/log"

	"os"
)

var (
	signingSecret, _ = os.LookupEnv("SLACK_SIGNING_SECRET")
)

type SlashBody struct {
	Token    string `form:"token" json:"token" binding:"required"`
	SSLCheck bool   `form:"ssl_check" json:"ssl_check"`
}

// func SSLCheckInterceptor() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		var body SlashBody
// 		err := c.BindJSON(&body)
// 		if err != nil {
// 			// c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": "Bad request. expected JSON body"})
// 			// return
// 			log.Error(err)
// 			c.Next()
// 			return
// 		}
// 		if body.SSLCheck {
// 			c.AbortWithStatusJSON(http.StatusOK, gin.H{"status": "OK"})
// 			c.Next()
// 			return
// 		}
// 		c.Next()
// 	}
// }

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

func main() {
	r := gin.Default()
	r.Use(gin.Logger())
	// r.Use(SSLCheckInterceptor())
	r.Use(ValidateSignature())

	r.POST("/slash", func(c *gin.Context) {
		s, err := slack.SlashCommandParse(c.Request)
		if err != nil {
			log.Error(err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": "Bad Request"})
			return
		}
		switch s.Command {
		case "/secret":
			if err != nil {
				log.Error(err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			c.String(http.StatusOK, s.Text)
		default:
			return
		}

	})
	port, err := strconv.ParseInt(os.Getenv("PORT"), 10, 64)
	if err != nil {
		port = 8080
	}

	r.Run(fmt.Sprintf("0.0.0.0:%v", port)) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
