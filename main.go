package main

import (
	"net/http"
	"os"

	"cure-link/controller"

	"github.com/gin-gonic/gin"
)

// var collection *mongo.Collection
// var ctx = context.TODO()

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{"message": "Hello World!"})
	})
	r.GET("/:code", controller.Redirect)
	r.GET("/generateUser", controller.GenerateUser)
	r.POST("/shorten", controller.Shorten)
	r.POST("/custom", controller.Custom)
	r.POST("/getAPIKey", controller.GetUserAPIKey)
	r.GET("/stats/:code", controller.GetStats)

	port := os.Getenv("PORT")
	r.Run(":" + port)

	// r.Run((":8080"))

}
