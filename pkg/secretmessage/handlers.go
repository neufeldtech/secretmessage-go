package secretmessage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid"
	"github.com/neufeldtech/smsg-go/pkg/redis"
	"github.com/prometheus/common/log"
	"github.com/slack-go/slack"
)

func HandleSlash(c *gin.Context) {
	r := redis.GetClient()
	s, err := slack.SlashCommandParse(c.Request)
	if err != nil {
		log.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": "Bad Request"})
		return
	}

	switch s.Command {
	case "/secret":

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
			c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
			return
		}

		secretID := shortuuid.New()

		// err = r.Set(hash(secretID), encrypt(s.Text, "jordan"), 0).Err()
		err = r.Set(hash(secretID), s.Text, 0).Err()
		log.Infof("secret id: %v", secretID)

		if err != nil {
			log.Errorf("error storing secretID %v in redis: %v", secretID, err)
			response = slack.Message{
				Msg: slack.Msg{
					ResponseType: slack.ResponseTypeEphemeral,
					Text:         ":x: Sorry, an error occurred attempting to create secret",
				},
			}
			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Errorf("error marshalling response: %v", err)
			}
			c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
			return
		}
		response = slack.Message{
			Msg: slack.Msg{
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
		responseBytes, err := json.Marshal(response)
		if err != nil {
			log.Errorf("error marshalling response: %v", err)
		}
		c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
		return
	default:
		c.Data(http.StatusOK, gin.MIMEPlain, nil)
		return
	}
}

func HandleInteractive(c *gin.Context) {
	r := redis.GetClient()

	var err error
	if err != nil {
		log.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": "Bad Request"})
		return
	}

	var i slack.InteractionCallback
	payload := c.PostForm("payload")
	err = json.Unmarshal([]byte(payload), &i)
	if err != nil {
		log.Error(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "Error with the stuffs"})
		return
	}
	callbackType := strings.Split(i.CallbackID, ":")[0]
	switch callbackType {
	case "send_secret":
		secretID := strings.ReplaceAll(i.CallbackID, "send_secret:", "")
		log.Infof("secret id: %v", secretID)
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
			responseBytes, err := json.Marshal(response)
			log.Error(err)
			c.Data(http.StatusOK, gin.MIMEJSON, responseBytes)
			return
		}

		// secret := decrypt(secretEncrypted, "jordan")
		secret := secretEncrypted

		response := slack.Message{
			Msg: slack.Msg{
				DeleteOriginal: true,
				ResponseType:   slack.ResponseTypeEphemeral,
				Attachments: []slack.Attachment{{
					Title:      "Secret message",
					Fallback:   "Secret message",
					Text:       secret,
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
		c.Data(http.StatusOK, gin.MIMEPlain, nil)
	}
}
