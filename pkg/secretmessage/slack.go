package secretmessage

import "github.com/slack-go/slack"

var (
	api *slack.Client
)

func InitSlackClient(config Config) {
	api = slack.New(config.SlackToken, slack.OptionDebug(true))
}
func SlackClient() *slack.Client {
	return api
}
