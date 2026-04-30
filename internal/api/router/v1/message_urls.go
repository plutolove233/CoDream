package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/api/handler/message"
)

func RegisterMessageRouters(engine *gin.RouterGroup) {
	m := message.NewMessageAPI()
	messageGroup := engine.Group("/message")
	{
		messageGroup.POST("/chat", m.ChatHandler)
	}
}
