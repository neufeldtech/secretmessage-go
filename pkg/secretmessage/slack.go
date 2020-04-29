package secretmessage

import "github.com/slack-go/slack"

var (
	api *slack.Client
)

func InitSlackClient() {
	api = slack.New("anonymous", slack.OptionDebug(true))
}
func SlackClient() *slack.Client {
	return api
}
