package secretslack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/slack-go/slack"
)

var (
	apiClients = make(map[string]*slack.Client)
	mux        sync.Mutex
	httpClient = http.DefaultClient
)

func SetHTTPClient(hc *http.Client) {
	httpClient = hc
}

func GetSlackClient(token string) *slack.Client {
	mux.Lock()
	defer mux.Unlock()

	if client, exists := apiClients[token]; exists {
		return client
	}

	client := slack.New(token, slack.OptionDebug(true))
	apiClients[token] = client
	return client
}

// SendResponseUrlMessage sends a slack message via a response_url - It does not require a token
func SendResponseUrlMessage(ctx context.Context, uri string, msg slack.Message) error {

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, bytes.NewBuffer(msgBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		e := fmt.Sprintf("error: received status code from slack %v", resp.StatusCode)
		return errors.New(e)
	}
	return err
}

// NewSlackErrorResponse Constructs a json response for an ephemeral message back to a user
func NewSlackErrorResponse(title string, text string, deleteOriginal bool, callbackID string) ([]byte, int) {
	responseCode := http.StatusOK
	response := slack.Message{
		Msg: slack.Msg{
			DeleteOriginal: deleteOriginal,
			ResponseType:   slack.ResponseTypeEphemeral,
			Attachments: []slack.Attachment{{
				Title:      title,
				Fallback:   title,
				Text:       text,
				CallbackID: callbackID,
				Color:      "#FF0000",
			}},
		},
	}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("error marshalling json: %v", err)
		responseCode = http.StatusInternalServerError
	}
	return responseBytes, responseCode
}
