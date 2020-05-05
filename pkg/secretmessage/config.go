package secretmessage

type Config struct {
	SkipSkignatureValidation bool
	Port                     int64
	RedisAddress             string
	RedisPassword            string
	SlackToken               string
	SigningSecret            string
}
