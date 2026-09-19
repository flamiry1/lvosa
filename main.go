package main

import (
	"log"
	"github.com/gin-gonic/gin"
	"net/http"
)

func main(){
	r := gin.Default()

	r.GET("health/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	log.Println("server started")
	r.Run(":8080")

}
