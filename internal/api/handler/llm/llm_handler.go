package llm

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/api/service"
	"github.com/plutolove233/co-dream/internal/globals"
	"github.com/plutolove233/co-dream/pkg/types"
)

type API struct {
	streamer llmStreamer
}

type llmStreamer interface {
	Stream(ctx context.Context, req service.LLMStreamRequest) (<-chan types.CompleteEvent, error)
}

func NewAPI() *API {
	return &API{streamer: service.NewLLMStreamService(nil)}
}

func NewAPIWithStreamer(streamer llmStreamer) *API {
	return &API{streamer: streamer}
}

func (a *API) ChatStream(c *gin.Context) {
	if a.streamer == nil {
		a.streamer = service.NewLLMStreamService(nil)
	}

	var req service.LLMStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		globals.JsonParameterIllegal(c, "请求参数不符合要求", err)
		return
	}

	events, err := a.streamer.Stream(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrLLMAPIKeyRequired) {
			globals.JsonParameterIllegal(c, "LLM API Key 未配置", err)
			return
		}
		globals.JsonInternalError(c, "创建模型流失败", err)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	c.Stream(func(w io.Writer) bool {
		event, ok := <-events
		if !ok {
			c.SSEvent("done", gin.H{})
			return false
		}

		switch {
		case event.Err != nil:
			c.SSEvent("error", gin.H{"error": event.Err.Error()})
			return false
		case event.Delta != "":
			c.SSEvent("delta", gin.H{"delta": event.Delta})
		case event.Result != nil:
			c.SSEvent("result", gin.H{
				"content":       event.Result.Content,
				"finish_reason": event.Result.FinishReason,
				"tool_calls":    event.Result.ToolCalls,
				"usage":         event.Result.Usage,
			})
		}
		return true
	})
}
