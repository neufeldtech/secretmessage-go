package secretmessage

import "github.com/gin-gonic/gin"

func SetupRouter(config Config) *gin.Engine {
	InitRedis(config)
	InitSlackClient(config)
	r := gin.Default()
	r.Use(gin.Logger())
	r.Use(ValidateSignature(config))

	r.POST("/slash", HandleSlash)
	r.POST("/interactive", HandleInteractive)
	return r
}
