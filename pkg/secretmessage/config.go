package secretmessage

type Config struct {
	SkipSignatureValidation bool
	Port                    int64
	RedisAddress            string
	RedisPassword           string
	SlackToken              string
	SigningSecret           string
}
