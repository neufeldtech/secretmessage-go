package secretmessage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid"
	"github.com/prometheus/common/log"
	"github.com/slack-go/slack"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmgoredis"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

func HandleSlash(c *gin.Context) {
	tx := apm.TransactionFromContext(c.Request.Context())
	r := apmgoredis.Wrap(GetRedisClient()).WithContext(c.Request.Context())

	s, err := slack.SlashCommandParse(c.Request)
	if err != nil {
		log.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": "Bad Request"})
		tx.Context.SetLabel("errorCode", "slash_payload_parse_error")
		return
	}
	tx.Context.SetLabel("userHash", hash(s.UserID))
	tx.Context.SetLabel("teamHash", hash(s.TeamID))
	tx.Context.SetLabel("action", "createSecret")
	switch s.Command {
	case "/secret":
		tx.Context.SetLabel("slashCommand", "/secret")
		var response slack.Message
		if s.Text == "" {
			response = slack.Message{
				Msg: slack.Msg{
					ResponseType: slack.ResponseTypeEphemeral,
					Attachments: []slack.Attachment{{
						Title:      "Error: secret text is empty",
						Fallback:   "Error: secret text is empty",
						Text:       "It looks like you tried to send a secret but forgot to provide the secret's text. You can send a secret like this: `/secret I am scared of heights`",
						CallbackID: "secret_text_empty:",
						Color:      "#FF0000",
					}},
				},
			}
			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Errorf("error marshalling response: %v", err)
			}
			tx.Context.SetLabel("errorCode", "text_empty")
			c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
			return
		}

		secretID := shortuuid.New()
		tx.Context.SetLabel("secretIDHash", hash(secretID))
		secretEncrypted, err := encrypt(s.Text, secretID)
		if err != nil {
			log.Errorf("error storing secretID %v in redis: %v", secretID, err)
			response = slack.Message{
				Msg: slack.Msg{
					ResponseType: slack.ResponseTypeEphemeral,
					Text:         ":x: Sorry, an error occurred attempting to create secret",
				},
			}
			tx.Context.SetLabel("errorCode", "encrypt_error")
			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Errorf("error marshalling response: %v", err)
			}
			c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
			return
		}

		err = r.Set(hash(secretID), secretEncrypted, 0).Err()

		if err != nil {
			log.Errorf("error storing secretID %v in redis: %v", secretID, err)
			response = slack.Message{
				Msg: slack.Msg{
					ResponseType: slack.ResponseTypeEphemeral,
					Text:         ":x: Sorry, an error occurred attempting to create secret",
				},
			}
			tx.Context.SetLabel("errorCode", "redis_set_error")
			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Errorf("error marshalling response: %v", err)
			}
			c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
			return
		}
		// Send the empty Ack to Slack
		c.Data(http.StatusOK, gin.MIMEPlain, nil)
		// tx.End()

		// tx2 := apm.DefaultTracer.StartTransaction("POST SlackResponseURL", "client_request")
		// defer tx2.End()
		response = slack.Message{
			Msg: slack.Msg{
				ResponseType:   slack.ResponseTypeInChannel,
				DeleteOriginal: true,
				Attachments: []slack.Attachment{{
					Title:      fmt.Sprintf("%v sent a secret message", s.UserName),
					Fallback:   fmt.Sprintf("%v sent a secret message", s.UserName),
					CallbackID: fmt.Sprintf("send_secret:%v", secretID),
					Color:      "#6D5692",
					Actions: []slack.AttachmentAction{{
						Name:  "readMessage",
						Text:  ":envelope: Read message",
						Type:  "button",
						Value: "readMessage",
					}},
				}},
			},
		}

		err = SendMessage(s.ResponseURL, response)

		if err != nil {
			log.Error(err)
			response = slack.Message{
				Msg: slack.Msg{
					ResponseType: slack.ResponseTypeEphemeral,
					Text:         ":x: Sorry, an error occurred attempting to create secret",
				},
			}
			// tx2.Context.SetLabel("errorCode", "send_secret_error")
			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Errorf("error marshalling response: %v", err)
			}
			c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
		}

		return
	default:
		c.Data(http.StatusOK, gin.MIMEPlain, nil)
		return
	}
}

func HandleOauthBegin(c *gin.Context) {
	state := shortuuid.New()
	url := GetConfig().OauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)

	c.SetCookie("state", state, 0, "", "", false, true)
	c.Redirect(302, url)
}

func HandleOauthCallback(c *gin.Context) {
	tx := apm.TransactionFromContext(c.Request.Context())

	stateQuery := c.Query("state")
	conf := GetConfig()
	stateCookie, err := c.Cookie("state")
	if err != nil {
		log.Errorf("error retrieving state cookie from request: %v", err)
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "state_cookie_parse_error")
		return
	}
	if stateCookie != stateQuery {
		log.Error("error validating state cookie with state query param")
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "state_cookie_invalid")
		return
	}

	_, err = conf.OauthConfig.Exchange(context.Background(), c.Query("code"))
	if err != nil {
		log.Errorf("error retrieving initial oauth token: %v", err)
		c.Redirect(302, "https://secretmessage.xyz/error")
		tx.Context.SetLabel("errorCode", "oauth_token_exchange_error")
		return
	}

	c.Redirect(302, "https://secretmessage.xyz/success")
}

func HandleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

func HandleInteractive(c *gin.Context) {
	tx := apm.TransactionFromContext(c.Request.Context())
	r := apmgoredis.Wrap(GetRedisClient()).WithContext(c.Request.Context())

	var err error

	var i slack.InteractionCallback
	payload := c.PostForm("payload")
	err = json.Unmarshal([]byte(payload), &i)
	if err != nil {
		log.Error(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "Error with the stuffs"})
		tx.Context.SetLabel("errorCode", "interaction_payload_parse_error")
		return
	}
	tx.Context.SetLabel("userHash", hash(i.User.ID))
	tx.Context.SetLabel("teamHash", hash(i.User.TeamID))
	callbackType := strings.Split(i.CallbackID, ":")[0]
	switch callbackType {
	case "send_secret":
		tx.Context.SetLabel("callbackID", "send_secret")
		tx.Context.SetLabel("action", "sendSecret")
		secretID := strings.ReplaceAll(i.CallbackID, "send_secret:", "")
		tx.Context.SetLabel("secretIDHash", hash(secretID))
		secretEncrypted, err := r.Get(hash(secretID)).Result()
		if err != nil {
			log.Error(err)
			response := slack.Message{
				Msg: slack.Msg{
					ResponseType:   slack.ResponseTypeEphemeral,
					DeleteOriginal: true,
					Text:           ":x: Sorry, an error occurred attempting to retrieve secret",
				},
			}
			tx.Context.SetLabel("errorCode", "redis_get_error")
			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Errorf("error marshalling response: %v", err)
			}
			c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
			return
		}
		var secretDecrypted string
		if strings.Contains(secretEncrypted, ":") {
			secretDecrypted, err = decryptIV(secretEncrypted, config.LegacyCryptoKey)
		} else {
			secretDecrypted, err = decrypt(secretEncrypted, secretID)
		}
		if err != nil {
			log.Errorf("error retrieving secretID %v from redis: %v", secretID, err)
			response := slack.Message{
				Msg: slack.Msg{
					ResponseType: slack.ResponseTypeEphemeral,
					Text:         ":x: Sorry, an error occurred attempting to retrieve secret",
				},
			}
			tx.Context.SetLabel("errorCode", "decrypt_error")
			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Errorf("error marshalling response: %v", err)
			}
			c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
			return
		}

		response := slack.Message{
			Msg: slack.Msg{
				DeleteOriginal: true,
				ResponseType:   slack.ResponseTypeEphemeral,
				Attachments: []slack.Attachment{{
					Title:      "Secret message",
					Fallback:   "Secret message",
					Text:       secretDecrypted,
					CallbackID: fmt.Sprintf("delete_secret:%v", secretID),
					Color:      "#6D5692",
					Footer:     "The above message is only visible to you and will disappear when your Slack client reloads. To remove it immediately, click the button below:",
					Actions: []slack.AttachmentAction{{
						Name:  "removeMessage",
						Text:  ":x: Delete message",
						Type:  "button",
						Style: "danger",
						Value: "removeMessage",
					}},
				}},
			},
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			log.Errorf("error marshalling response: %v", err)
		}
		r.Del(hash(secretID))
		c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
		return
	case "delete_secret":
		secretID := strings.ReplaceAll(i.CallbackID, "delete_secret:", "")
		tx.Context.SetLabel("secretIDHash", hash(secretID))
		tx.Context.SetLabel("callbackID", "delete_secret")
		tx.Context.SetLabel("action", "deleteMessage")
		response := slack.Message{
			Msg: slack.Msg{
				DeleteOriginal: true,
			},
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			log.Errorf("error marshalling response: %v", err)
		}
		c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
	default:
		log.Error("Hit the default case. bad things happened")
		c.Data(http.StatusInternalServerError, gin.MIMEPlain, nil)
	}
}
