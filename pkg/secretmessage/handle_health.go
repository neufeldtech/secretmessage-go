package secretmessage

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func (ctl *PublicController) HandleHealth(c *gin.Context) {
	version, ok := os.LookupEnv("NF_DEPLOYMENT_SHA")
	if !ok {
		version = "dev"
	}
	db, err := ctl.db.DB()
	if err != nil {
		log.Error(err)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN", "sha": version})
		return
	}

	err = db.PingContext(c.Request.Context())
	if err != nil {
		log.Error(err)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN", "sha": version})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "UP", "sha": version})
}
