package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/neufeldtech/smsg-go/pkg/redis"
	"github.com/nlopes/slack"
	"github.com/prometheus/common/log"
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
		if err != nil {
			log.Error(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// r.Set("last", s.Text, 0)
		res, err := r.Get("last").Result()
		if err != nil {
			log.Error(err)
			c.String(http.StatusOK, "Error retrieving secret")
		}
		c.String(http.StatusOK, res)
	default:
		return
	}
}
