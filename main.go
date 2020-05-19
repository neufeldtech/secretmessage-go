package main

import (
	"fmt"
	"strconv"

	"github.com/go-redis/redis"
	"github.com/neufeldtech/smsg-go/pkg/secretmessage"
	"github.com/prometheus/common/log"
	"golang.org/x/oauth2"

	"os"
)

var (
	defaultPort                 int64 = 8080
	slackSigningSecretConfigKey       = "slackSigningSecret"
	slackClientIDConfigKey            = "slackClientID"
	slackClientSecretConfigKey        = "slackClientSecret"
	slackCallbackURLConfigKey         = "slackCallbackURL"
	legacyCryptoKeyConfigKey          = "legacyCryptoKey"
	appURLConfigKey                   = "appURL"
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

	configMap := map[string]string{
		slackSigningSecretConfigKey: os.Getenv("SLACK_SIGNING_SECRET"),
		slackClientIDConfigKey:      os.Getenv("SLACK_CLIENT_ID"),
		slackClientSecretConfigKey:  os.Getenv("SLACK_CLIENT_SECRET"),
		slackCallbackURLConfigKey:   os.Getenv("SLACK_CALLBACK_URL"),
		legacyCryptoKeyConfigKey:    os.Getenv("CRYPTO_KEY"),
		appURLConfigKey:             os.Getenv("APP_URL"),
	}
	for k, v := range configMap {
		if v == "" {
			log.Fatalf("error initializaing config. key %v was not set", k)
		}
	}
	redisOptions, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatalf("error parsing REDIS_URL: %v", err)
	}

	secretmessage.SetConfig(secretmessage.Config{
		Port:          resolvePort(),
		RedisOptions:  redisOptions,
		SlackToken:    "",
		SigningSecret: configMap[slackSigningSecretConfigKey],
		AppURL:        configMap[appURLConfigKey],
		OauthConfig: &oauth2.Config{
			ClientID:     configMap[slackClientIDConfigKey],
			ClientSecret: configMap[slackClientSecretConfigKey],
			RedirectURL:  os.Getenv(slackCallbackURLConfigKey),
			Scopes:       []string{"commands", "chat:write:bot"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://slack.com/oauth/authorize",
				TokenURL: "https://slack.com/api/oauth.access",
			},
		},
	})
	go secretmessage.StayAwake(secretmessage.GetConfig())
	r := secretmessage.SetupRouter(secretmessage.GetConfig())
	log.Infof("Booted and listening on port %v", secretmessage.GetConfig().Port)
	r.Run(fmt.Sprintf("0.0.0.0:%v", secretmessage.GetConfig().Port)) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
