package main

import (
	"context"
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

	var logger *zap.Logger
	switch {
	case os.Getenv("APP_ENV") == "development":
		logger = zap.Must(zap.NewDevelopment())
		gin.SetMode(gin.DebugMode)
	default:
		gin.SetMode(gin.ReleaseMode)
		logger = zap.Must(zap.NewProduction())
	}

	tp, err := secretmessage.InitTracer(secretmessage.ServiceName)
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer func() {
		logger.Sync()
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error("error shutting down trace provider", zap.Error(err))
		}
	}()

	for k, v := range configMap {
		if v == "" {
			logger.Fatal("error initializing config", zap.String("key", k), zap.String("value", v))
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
