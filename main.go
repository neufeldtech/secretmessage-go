package main

import (
	"fmt"
	"net/http"
	"time"

	"strconv"

	"os"

	"github.com/neufeldtech/secretmessage-go/pkg/secretmessage"
	"github.com/neufeldtech/secretmessage-go/pkg/secretslack"
	"github.com/prometheus/common/log"
	_ "go.elastic.co/apm/module/apmgormv2"
	postgres "go.elastic.co/apm/module/apmgormv2/driver/postgres"

	"go.elastic.co/apm/module/apmhttp"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

var (
	defaultPort                 int64 = 8080
	slackSigningSecretConfigKey       = "slackSigningSecret"
	slackClientIDConfigKey            = "slackClientID"
	slackClientSecretConfigKey        = "slackClientSecret"
	slackCallbackURLConfigKey         = "slackCallbackURL"
	legacyCryptoKeyConfigKey          = "legacyCryptoKey"
	appURLConfigKey                   = "appURL"
	databaseURL                       = "databaseURL"

	configMap = map[string]string{
		slackSigningSecretConfigKey: os.Getenv("SLACK_SIGNING_SECRET"),
		slackClientIDConfigKey:      os.Getenv("SLACK_CLIENT_ID"),
		slackClientSecretConfigKey:  os.Getenv("SLACK_CLIENT_SECRET"),
		slackCallbackURLConfigKey:   os.Getenv("SLACK_CALLBACK_URL"),
		legacyCryptoKeyConfigKey:    os.Getenv("CRYPTO_KEY"),
		appURLConfigKey:             os.Getenv("APP_URL"),
		databaseURL:                 os.Getenv("DATABASE_URL"),
	}
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
	for k, v := range configMap {
		if v == "" {
			log.Fatalf("error initializaing config. key %v was not set", k)
		}
	}

	conf := secretmessage.Config{
		Port:            resolvePort(),
		SlackToken:      "",
		SigningSecret:   configMap[slackSigningSecretConfigKey],
		AppURL:          configMap[appURLConfigKey],
		LegacyCryptoKey: configMap[legacyCryptoKeyConfigKey],
		DatabaseURL:     configMap[databaseURL],
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

	db, err := gorm.Open(postgres.Open(conf.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	d, _ := db.DB()
	d.SetMaxIdleConns(10)
	d.SetMaxOpenConns(10)

	db.AutoMigrate(secretmessage.Secret{})
	db.AutoMigrate(secretmessage.Team{})

	controller := secretmessage.NewController(
		conf,
		db,
	)

	go secretmessage.StayAwake(conf)
	r := controller.ConfigureRoutes()
	log.Infof("Booted and listening on port %v", conf.Port)
	r.Run(fmt.Sprintf("0.0.0.0:%v", conf.Port))
}
