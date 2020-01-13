package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"

	"os"
)

func main() {
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	port, err := strconv.ParseInt(os.Getenv("PORT"), 10, 64)
	if err != nil {
		port = 8080
	}

	r.Run(fmt.Sprintf("0.0.0.0:%v", port)) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
