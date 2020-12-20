package main

import (
	"fmt"
	"net/http"
	"time"

	"strconv"

	"os"

	"github.com/go-redis/redis"
	"github.com/neufeldtech/secretmessage-go/pkg/secretdb"
	"github.com/neufeldtech/secretmessage-go/pkg/secretmessage"
	"github.com/neufeldtech/secretmessage-go/pkg/secretslack"
	"github.com/prometheus/common/log"
	"go.elastic.co/apm/module/apmhttp"
	"golang.org/x/oauth2"
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
	// Setup custom HTTP Client for calling Slack
	secretslack.SetHTTPClient(apmhttp.WrapClient(
		&http.Client{
			Timeout: time.Second * 5,
		},
	))
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

	db, err := secretdb.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	log.Info("running db migrations")
	err = secretdb.RunMigrations(db)
	if err != nil {
		log.Fatal(err)
	}
	conf := secretmessage.Config{
		Port:            resolvePort(),
		RedisOptions:    redisOptions,
		SlackToken:      "",
		SigningSecret:   configMap[slackSigningSecretConfigKey],
		AppURL:          configMap[appURLConfigKey],
		LegacyCryptoKey: configMap[legacyCryptoKeyConfigKey],
		OauthConfig: &oauth2.Config{
			ClientID:     configMap[slackClientIDConfigKey],
			ClientSecret: configMap[slackClientSecretConfigKey],
			RedirectURL:  configMap[slackCallbackURLConfigKey],
			Scopes:       []string{"chat:write", "commands", "workflow.steps:execute"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://slack.com/oauth/v2/authorize",
				TokenURL: "https://slack.com/api/oauth.v2.access",
			},
		},
	}
	controller := secretmessage.NewController(
		db,
		secretdb.NewSecretsRepository(db),
		secretdb.NewTeamsRepository(db),
		conf,
	)

	// secretmessage.SetConfig(secretmessage.Config{})

	go secretmessage.StayAwake(conf)
	r := controller.ConfigureRoutes()
	log.Infof("Booted and listening on port %v", conf.Port)
	r.Run(fmt.Sprintf("0.0.0.0:%v", conf.Port)) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
