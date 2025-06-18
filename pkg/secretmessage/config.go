package secretmessage

import (
	"golang.org/x/oauth2"
)

type Config struct {
	SkipSignatureValidation bool
	Port                    int64
	SlackToken              string
	SigningSecret           string
	LegacyCryptoKey         string
	AppURL                  string
	OauthConfig             *oauth2.Config
	DatabaseURL             string
}
