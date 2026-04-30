package message

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/agent"
	"github.com/plutolove233/co-dream/internal/globals"
	openai "github.com/sashabaranov/go-openai"
)

type MessageAPI struct {
}

func NewMessageAPI() *MessageAPI {
	return &MessageAPI{}
}

type ChatCompletionParser struct {
	ChatSessionID string `json:"chat_session_id" required:"true"`
	Prompt        string `json:"prompt" required:"true"`
}

type ChatCompletionResponse struct {
}

func (m *MessageAPI) ChatHandler(c *gin.Context) {
	// 设置响应头为text/event-stream
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	var parser ChatCompletionParser
	if err := c.ShouldBindJSON(&parser); err != nil {
		globals.JsonParameterIllegal(c, "请求体不符合要求", err)
		return
	}
	llm := agent.NewClient()
	req := openai.ChatCompletionRequest{
		Model: os.Getenv("MODEL_EPIP"),
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: parser.Prompt,
			},
		},
	}
	stream, err := llm.CreateChatCompletionStream(c.Request.Context(), req)
	if err != nil {
		globals.JsonInternalError(c, "创建聊天完成流失败", err)
		return
	}
	defer stream.Close()
	for {
		response, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			globals.JsonInternalError(c, "接收聊天完成流失败", err)
			return
		}
		// 这里可以根据需要处理 response，例如发送给前端
		c.SSEvent("message", response.Choices[0].Delta.Content)
		c.Writer.Flush()
	}
}
