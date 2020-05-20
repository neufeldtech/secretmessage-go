package secretmessage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/slack-go/slack"
)

var (
	api *slack.Client
)

func InitSlackClient(config Config) {
	api = slack.New(config.SlackToken, slack.OptionDebug(true))
}
func SlackClient() *slack.Client {
	return api
}

func SendMessage(uri string, msg slack.Message) error {
	client := &http.Client{}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, uri, bytes.NewBuffer(msgBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		e := fmt.Sprintf("error: received status code from slack %v", resp.StatusCode)
		return errors.New(e)
	}
	return nil
}
