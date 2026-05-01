package interfaces

import (
	"context"

	"github.com/plutolove233/co-dream/pkg/types"

	openai "github.com/sashabaranov/go-openai"
)

// LLMProvider abstracts the LLM provider for use by tools.
type LLMProvider interface {
	// Complete performs a chat completion with tool support.
	Complete(ctx context.Context, messages []types.Message, system string) (<-chan types.CompleteEvent, error)
	// Model returns the model name being used.
	Model() string
	// ExecuteTool allows tools to call other tools via the LLM.
	ExecuteTools(ctx context.Context, toolCalls []openai.ToolCall) ([]types.ToolCallResult, error)
}
