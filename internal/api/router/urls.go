package router

import (
	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/api/handler"
	"github.com/plutolove233/co-dream/internal/api/handler/ws"
	v1 "github.com/plutolove233/co-dream/internal/api/router/v1"
)

func InitRouter(engine *gin.Engine) {
	// http基本路由
	{
		base := engine.Group("/api/v1")
		base.GET("/version", handler.GetVersion)
		v1.RegisterUserRouters(base)
	}
	// ws基本路由
	{
		base := engine.Group("/ws/v1")
		base.GET("/ping", ws.PingWSHandler)
	}
}
