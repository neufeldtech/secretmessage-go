package main

import (
	"fmt"

	"strconv"

	"os"

	"github.com/gin-gonic/gin"
	"github.com/neufeldtech/secretmessage-go/pkg/secretmessage"

	"go.uber.org/zap"

	"golang.org/x/oauth2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	defaultPort                  int64 = 8080
	defaultExpirationTimeSeconds int64 = 86400
	slackSigningSecretConfigKey        = "slackSigningSecret"
	slackClientIDConfigKey             = "slackClientID"
	slackClientSecretConfigKey         = "slackClientSecret"
	slackCallbackURLConfigKey          = "slackCallbackURL"
	appURLConfigKey                    = "appURL"
	databaseURL                        = "databaseURL"

	configMap = map[string]string{
		slackSigningSecretConfigKey: os.Getenv("SLACK_SECRET"),
		slackClientIDConfigKey:      os.Getenv("SLACK_CLIENT_ID"),
		slackClientSecretConfigKey:  os.Getenv("SLACK_CLIENT_SECRET"),
		slackCallbackURLConfigKey:   os.Getenv("SLACK_CALLBACK_URL"),
		appURLConfigKey:             os.Getenv("SLACK_APP_URL"),
	}
)

func resolveExpirationTime() int64 {

	expirationTimeString := os.Getenv("EXPIRATION_TIME")
	if expirationTimeString == "" {
		return defaultExpirationTimeSeconds
	}
	expirationTime64, err := strconv.ParseInt(expirationTimeString, 10, 64)
	if err != nil {
		return defaultExpirationTimeSeconds
	}
	return expirationTime64
}

func resolvePort() int64 {

	portString := os.Getenv("SERVER_PORT")
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

	var logger *zap.Logger
	gin.SetMode(gin.ReleaseMode)
	logger = zap.Must(zap.NewProduction())

	defer func() {
		logger.Sync()
	}()

	for k, v := range configMap {
		if v == "" {
			logger.Fatal("error initializing config", zap.String("key", k), zap.String("value", v))
		}
	}

	configMap[databaseURL] = fmt.Sprintf(
		"postgres://%s:%s@%s/%s", os.Getenv("DATABASE_USERNAME"), os.Getenv("DATABASE_PASSWORD"), os.Getenv("DATABASE_HOST"), os.Getenv("DATABASE_NAME"),
	)

	conf := secretmessage.Config{
		Port:           resolvePort(),
		SlackToken:     "",
		SigningSecret:  configMap[slackSigningSecretConfigKey],
		AppURL:         configMap[appURLConfigKey],
		DatabaseURL:    configMap[databaseURL],
		ExpirationTime: resolveExpirationTime(),
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
		logger.Fatal("error connecting to database", zap.Error(err))
	}
	d, _ := db.DB()
	d.SetMaxIdleConns(10)
	d.SetMaxOpenConns(10)

	db.AutoMigrate(secretmessage.Secret{})
	db.AutoMigrate(secretmessage.Team{})

	controller := secretmessage.NewController(
		conf,
		db,
		logger,
	)

	go controller.StayAwake()
	r := controller.ConfigureRoutes()
	logger.Sugar().Infof("Booted and listening on port %v", conf.Port)

	r.Run(fmt.Sprintf("0.0.0.0:%v", conf.Port))
}
