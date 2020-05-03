package main

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/neufeldtech/smsg-go/pkg/secretmessage"

	"os"
)

var (
	defaultPort int64 = 8080
)

func resolvePort() int64 {

	portString := os.Getenv("PORT")
	if portString == "" {
		return defaultPort
	}
	port64, err := strconv.ParseInt(portString, 10, 64)
	if err != nil {
		return defaultPort
	}
	return port64
}

func setupRouter(config secretmessage.Config) *gin.Engine {
	secretmessage.InitRedis(config)
	secretmessage.InitSlackClient(config)
	r := gin.Default()
	r.Use(gin.Logger())
	r.Use(secretmessage.ValidateSignature(config))

	r.POST("/slash", secretmessage.HandleSlash)
	r.POST("/interactive", secretmessage.HandleInteractive)
	return r
}

func main() {
	config := secretmessage.Config{
		Port:          resolvePort(),
		RedisAddress:  os.Getenv("REDIS_ADDR"),
		SlackToken:    "",
		RedisPassword: os.Getenv("REDIS_PASS"),
		SigningSecret: os.Getenv("SLACK_SIGNING_SECRET"),
	}
	r := setupRouter(config)

	r.Run(fmt.Sprintf("0.0.0.0:%v", config.Port)) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
