package router

import (
	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/api/handler"
)

func InitRouter(engine *gin.Engine) {
	// 版本信息
	engine.GET("/version", handler.GetVersion)
}
