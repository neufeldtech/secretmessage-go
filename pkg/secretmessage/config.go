package secretmessage

import (
	"github.com/go-redis/redis"
	"golang.org/x/oauth2"
)

var (
	config Config
)

type Config struct {
	SkipSignatureValidation bool
	Port                    int64
	RedisOptions            *redis.Options
	SlackToken              string
	SigningSecret           string
	LegacyCryptoKey         string
	AppURL                  string
	OauthConfig             *oauth2.Config
}
