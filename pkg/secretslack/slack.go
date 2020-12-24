package secretslack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/common/log"
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

// Client returns a team-specific Slack API client for a given teamID. If one does not yet exist, it attempts to build one if we have an access_token stored for said team.
// func Client(teamID string) (*slack.Client, error) {
// 	if teamID == "" {
// 		return nil, errors.New("Invalid Team ID")
// 	}

// 	var apiClient *slack.Client
// 	apiClient = apiClients[teamID]

// 	if apiClient == nil {
// 		r := secretredis.Client()

// 		token, err := r.HGet(teamID, "access_token").Result()
// 		if err != nil {
// 			return nil, fmt.Errorf("error getting token from store for team %v: %v", teamID, err)
// 		}

// 		apiClient = slack.New(token, slack.OptionDebug(false), slack.OptionHTTPClient(httpClient))
// 		mux.Lock()
// 		defer mux.Unlock()
// 		apiClients[teamID] = apiClient
// 		return apiClient, nil
// 	}

// 	return apiClient, nil
// }

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
		log.Errorf("error marshalling json: %v", err)
		responseCode = http.StatusInternalServerError
	}
	return responseBytes, responseCode
}
