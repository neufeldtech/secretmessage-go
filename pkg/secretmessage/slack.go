package secretmessage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/slack-go/slack"
	"go.elastic.co/apm/module/apmhttp"
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

func SendMessage(ctx context.Context, uri string, msg slack.Message) (int, error) {
	client := apmhttp.WrapClient(http.DefaultClient)

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, bytes.NewBuffer(msgBytes))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return resp.StatusCode, err
	}
	if resp.StatusCode != http.StatusOK {
		e := fmt.Sprintf("error: received status code from slack %v", resp.StatusCode)
		return resp.StatusCode, errors.New(e)
	}
	return resp.StatusCode, nil
}
