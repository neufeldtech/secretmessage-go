package secretmessage

type Config struct {
	Port          int64
	RedisAddress  string
	RedisPassword string
	SlackToken    string
	SigningSecret string
}
