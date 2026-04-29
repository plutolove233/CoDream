package router

import (
	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/api/handler"
	v1 "github.com/plutolove233/co-dream/internal/api/router/v1"
)

func InitRouter(engine *gin.Engine) {
	// 版本信息
	{
		base := engine.Group("/api/v1")
		base.GET("/version", handler.GetVersion)
		v1.RegisterUserRoutes(base)
	}
}
