package v1

import (
	"github.com/gin-gonic/gin"
	llmhandler "github.com/plutolove233/co-dream/internal/api/handler/llm"
)

func RegisterLLMRouters(engine *gin.RouterGroup) {
	api := llmhandler.NewAPI()
	llmGroup := engine.Group("/llm")
	// llmGroup.Use(middleware.TokenRequired())
	{
		llmGroup.POST("/chat/stream", api.ChatStream)
	}
}
