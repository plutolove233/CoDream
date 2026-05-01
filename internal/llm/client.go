package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/plutolove233/co-dream/pkg/interfaces"
	"github.com/plutolove233/co-dream/pkg/types"
	openai "github.com/sashabaranov/go-openai"
)

type Client struct {
	client   *openai.Client
	model    string
	registry interfaces.ToolRegistry
}

func NewClient(apiKey string, baseURL string, model string, registry interfaces.ToolRegistry) *Client {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	return &Client{
		client:   openai.NewClientWithConfig(cfg),
		model:    model,
		registry: registry,
	}
}

func (c *Client) Model() string {
	return c.model
}

func (c *Client) Client() *openai.Client {
	return c.client
}

func (c *Client) Complete(ctx context.Context, messages []types.Message, system string) (<-chan types.CompleteEvent, error) {
	req := openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: c.buildMessages(messages, system),
		Stream:   true,
		Tools:    c.buildToolDefs(c.registry),
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, err
	}

	events := make(chan types.CompleteEvent, 16)
	go c.receiveCompletionStream(ctx, stream, events)
	return events, nil
}

func (c *Client) receiveCompletionStream(ctx context.Context, stream *openai.ChatCompletionStream, events chan<- types.CompleteEvent) {
	defer close(events)
	defer stream.Close()

	var fullContent strings.Builder

	type partialToolCall struct {
		id        string
		toolType  string
		name      string
		arguments strings.Builder
	}
	var toolCallOrder []int
	toolCallsByIdx := map[int]*partialToolCall{}
	var finishReason string
	usage := &types.TokenUsage{}

	for {
		resp, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			sendCompleteEvent(ctx, events, types.CompleteEvent{Err: err})
			return
		}

		if len(resp.Choices) == 0 {
			continue
		}

		delta := resp.Choices[0].Delta

		if fr := string(resp.Choices[0].FinishReason); fr != "" {
			finishReason = fr
		}

		if resp.Usage.PromptTokens > 0 || resp.Usage.CompletionTokens > 0 || resp.Usage.TotalTokens > 0 {
			usage.PromptTokens = int(resp.Usage.PromptTokens)
			usage.CompletionTokens = int(resp.Usage.CompletionTokens)
			usage.TotalTokens = int(resp.Usage.TotalTokens)
		}

		if delta.Content != "" {
			if !sendCompleteEvent(ctx, events, types.CompleteEvent{Delta: delta.Content}) {
				return
			}
			fullContent.WriteString(delta.Content)
		}

		for _, tc := range delta.ToolCalls {
			if tc.Index == nil {
				continue
			}
			idx := *tc.Index
			if _, exists := toolCallsByIdx[idx]; !exists {
				toolCallsByIdx[idx] = &partialToolCall{}
				toolCallOrder = append(toolCallOrder, idx)
			}
			p := toolCallsByIdx[idx]
			if tc.ID != "" {
				p.id = tc.ID
			}
			if tc.Type != "" {
				p.toolType = string(tc.Type)
			}
			if tc.Function.Name != "" {
				p.name = tc.Function.Name
			}
			p.arguments.WriteString(tc.Function.Arguments)
		}
	}

	toolCalls := make([]openai.ToolCall, 0, len(toolCallOrder))
	for _, idx := range toolCallOrder {
		p := toolCallsByIdx[idx]
		toolCalls = append(toolCalls, openai.ToolCall{
			ID:   p.id,
			Type: openai.ToolType(p.toolType),
			Function: openai.FunctionCall{
				Name:      p.name,
				Arguments: p.arguments.String(),
			},
		})
	}

	var usageResult *types.TokenUsage
	if usage.TotalTokens > 0 || usage.PromptTokens > 0 || usage.CompletionTokens > 0 {
		usageResult = usage
	}

	sendCompleteEvent(ctx, events, types.CompleteEvent{Result: &types.CompleteResult{
		Content:      fullContent.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usageResult,
	}})
}

func sendCompleteEvent(ctx context.Context, events chan<- types.CompleteEvent, event types.CompleteEvent) bool {
	select {
	case events <- event:
		return true
	case <-ctx.Done():
		return false
	}
}

func (c *Client) buildMessages(messages []types.Message, system string) []openai.ChatCompletionMessage {
	openaiMsgs := make([]openai.ChatCompletionMessage, 0, len(messages)+1)
	openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: system,
	})

	for _, m := range messages {
		switch m.Role {
		case openai.ChatMessageRoleUser:
			openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: m.Content,
			})
		case openai.ChatMessageRoleTool:
			for _, r := range m.ToolResults {
				openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    r.Content,
					ToolCallID: r.ToolCallID,
				})
			}
		case openai.ChatMessageRoleAssistant:
			openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
				Role:      openai.ChatMessageRoleAssistant,
				Content:   m.Content,
				ToolCalls: m.ToolCalls,
			})
		}
	}
	return openaiMsgs
}

func (c *Client) buildToolDefs(registry interfaces.ToolRegistry) []openai.Tool {
	if registry == nil {
		return nil
	}

	toolDefs := registry.EnabledTools()
	if len(toolDefs) == 0 {
		return nil
	}

	result := make([]openai.Tool, len(toolDefs))
	for i, t := range toolDefs {
		result[i] = openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		}
	}
	return result
}

func (c *Client) ExecuteTools(ctx context.Context, toolCalls []openai.ToolCall) ([]types.ToolCallResult, error) {
	var results []types.ToolCallResult
	if c.registry == nil {
		return results, nil
	}

	enabledTools := c.registry.EnabledTools()

	for _, tc := range toolCalls {
		if err := ctx.Err(); err != nil {
			return results, err
		}

		fn := tc.Function
		if fn.Name == "" {
			continue
		}

		input := []byte(fn.Arguments)
		var output string
		var toolFound bool

		for _, t := range enabledTools {
			if t.Name() == fn.Name {
				toolFound = true
				out, execErr := t.Execute(ctx, input)
				if execErr != nil {
					output = "Error: " + execErr.Error()
				} else {
					output = out
				}
				break
			}
		}
		if !toolFound {
			output = fmt.Sprintf("Error: tool %q not found or not enabled", fn.Name)
		}

		results = append(results, types.ToolCallResult{
			Name:       fn.Name,
			ToolCallID: tc.ID,
			Content:    output,
		})
	}
	return results, nil
}
