package secretmessage

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (ctl *PublicController) HandleHealth(c *gin.Context) {
	// err := ctl.db.Ping()
	// if err != nil {
	// 	log.Error(err)
	// 	c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN"})
	// 	return
	// }
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}
