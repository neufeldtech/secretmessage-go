package secretmessage

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func (ctl *PublicController) HandleHealth(c *gin.Context) {
	version, ok := os.LookupEnv("NF_DEPLOYMENT_SHA")
	if !ok {
		version = "dev"
	}

	// Neon removed their free tier. It now costs me money to keep the database up, so let it sleep to save costs.

	// db, err := ctl.db.DB()
	// if err != nil {
	// 	log.Error(err)
	// 	c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN", "sha": version})
	// 	return
	// }

	// err = db.PingContext(c.Request.Context())
	// if err != nil {
	// 	log.Error(err)
	// 	c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN", "sha": version})
	// 	return
	// }
	c.JSON(http.StatusOK, gin.H{"status": "UP", "sha": version})
}
