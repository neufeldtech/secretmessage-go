package main

import (
	"fmt"
	"strconv"

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

func main() {
	config := secretmessage.Config{
		Port:          resolvePort(),
		RedisAddress:  os.Getenv("REDIS_ADDR"),
		SlackToken:    "",
		RedisPassword: os.Getenv("REDIS_PASS"),
		SigningSecret: os.Getenv("SLACK_SIGNING_SECRET"),
	}
	r := secretmessage.SetupRouter(config)

	r.Run(fmt.Sprintf("0.0.0.0:%v", config.Port)) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
