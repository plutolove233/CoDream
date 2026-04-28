package router

import (
	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/api/handler"
)

func InitRouter(engine *gin.Engine) {
	// 版本信息
	{
		v1 := engine.Group("/api/v1")
		v1.GET("/version", handler.GetVersion)
	}
}
