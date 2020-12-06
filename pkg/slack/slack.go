package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neufeldtech/secretmessage-go/pkg/redis"
	"github.com/prometheus/common/log"
	sl "github.com/slack-go/slack"
	"go.elastic.co/apm/module/apmgoredis"
	"go.elastic.co/apm/module/apmhttp"
)

var (
	apiClients = make(map[string]*sl.Client)
	mux        sync.Mutex
)

func GetClient(teamID string) (*sl.Client, error) {
	if teamID == "" {
		return nil, errors.New("Invalid Team ID")
	}

	var apiClient *sl.Client
	apiClient = apiClients[teamID]

	if apiClient == nil {
		r := apmgoredis.Wrap(redis.GetRedisClient())
		token, err := r.HGet(teamID, "access_token").Result()
		if err != nil {
			return nil, fmt.Errorf("error getting token from redis for team %v: %v", teamID, err)
		}

		apiClient = sl.New(token, sl.OptionDebug(false))
		mux.Lock()
		defer mux.Unlock()
		apiClients[teamID] = apiClient
		return apiClient, nil
	}

	return apiClient, nil
}

func SendMessage(ctx context.Context, uri string, msg sl.Message) error {
	htc := &http.Client{
		Timeout: time.Second * 5,
	}
	client := apmhttp.WrapClient(htc)

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, bytes.NewBuffer(msgBytes))
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
	return err
}

// NewSlackErrorResponse Constructs a json response for an ephemeral message back to a user
func NewSlackErrorResponse(title, text, callbackID string) ([]byte, int) {
	responseCode := http.StatusOK
	response := sl.Message{
		Msg: sl.Msg{
			ResponseType: sl.ResponseTypeEphemeral,
			Attachments: []sl.Attachment{{
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
