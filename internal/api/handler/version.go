package handler

import "github.com/gin-gonic/gin"

func GetVersion(c *gin.Context) {
	c.JSON(200, gin.H{
		"version": "1.0.0",
	})
}
