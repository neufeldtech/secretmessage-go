package secretslack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

type SlackService struct {
	apiClients map[string]*slack.Client
	mux        sync.Mutex
	httpClient *http.Client
	logger     *zap.Logger
}

func NewSlackService() *SlackService {
	return &SlackService{
		apiClients: make(map[string]*slack.Client),
		mux:        sync.Mutex{},
		logger:     zap.Must(zap.NewProduction()),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (srv *SlackService) WithHTTPClient(hc *http.Client) *SlackService {
	srv.httpClient = hc
	return srv
}

func (srv *SlackService) WithLogger(logger *zap.Logger) *SlackService {
	if logger == nil {
		logger = zap.Must(zap.NewProduction())
		srv.logger.Warn("Logger is nil, using default production logger")
	}
	srv.logger = logger
	return srv
}

func (srv *SlackService) GetSlackClient(token string) *slack.Client {
	srv.mux.Lock()
	defer srv.mux.Unlock()

	if client, exists := srv.apiClients[token]; exists {
		return client
	}

	client := slack.New(token, slack.OptionDebug(true))
	srv.apiClients[token] = client
	return client
}

// SendResponseUrlMessage sends a slack message via a response_url - It does not require a token
func (srv *SlackService) SendResponseUrlMessage(ctx context.Context, uri string, msg slack.Message) error {

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, bytes.NewBuffer(msgBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := srv.httpClient.Do(req)
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
func (srv *SlackService) NewSlackErrorResponse(title string, text string, deleteOriginal bool, callbackID string) ([]byte, int) {
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
		srv.logger.Error("error marshalling json for slack error response", zap.Error(err), zap.String("callbackID", callbackID))
		responseCode = http.StatusInternalServerError
	}
	return responseBytes, responseCode
}
