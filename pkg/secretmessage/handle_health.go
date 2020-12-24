package secretmessage

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/log"
)

func (ctl *PublicController) HandleHealth(c *gin.Context) {
	db, err := ctl.db.DB()
	if err != nil {
		log.Error(err)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN"})
		return
	}

	err = db.PingContext(c.Request.Context())
	if err != nil {
		log.Error(err)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}
