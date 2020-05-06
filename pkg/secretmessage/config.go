package secretmessage

import "golang.org/x/oauth2"

var config Config

type Config struct {
	SkipSignatureValidation bool
	Port                    int64
	RedisAddress            string
	RedisPassword           string
	SlackToken              string
	SigningSecret           string
	OauthConfig             *oauth2.Config
}

func SetConfig(c Config) {
	config = c
}
func GetConfig() Config {
	return config
}
