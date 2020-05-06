package main

import (
	"fmt"
	"strconv"

	"github.com/neufeldtech/smsg-go/pkg/secretmessage"
	"golang.org/x/oauth2"

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
	secretmessage.SetConfig(secretmessage.Config{
		Port:          resolvePort(),
		RedisAddress:  os.Getenv("REDIS_ADDR"),
		SlackToken:    "",
		RedisPassword: os.Getenv("REDIS_PASS"),
		SigningSecret: os.Getenv("SLACK_SIGNING_SECRET"),
		OauthConfig: &oauth2.Config{
			ClientID:     os.Getenv("SLACK_CLIENT_ID"),
			ClientSecret: os.Getenv("SLACK_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("SLACK_CALLBACK_URL"),
			Scopes:       []string{"commands", "chat:write:bot"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://slack.com/oauth/authorize",
				TokenURL: "https://slack.com/api/oauth.access",
			},
		},
	})

	r := secretmessage.SetupRouter(secretmessage.GetConfig())

	r.Run(fmt.Sprintf("0.0.0.0:%v", secretmessage.GetConfig().Port)) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
