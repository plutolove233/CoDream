package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/plutolove233/co-dream/internal/llm/agents"
	"github.com/plutolove233/co-dream/pkg/interfaces"
	"github.com/plutolove233/co-dream/pkg/types"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type AgentToolContext struct {
	Pipeline    agents.PipelineDefinition
	Stage       agents.PipelineStage
	BaseContext map[string]any
}

func RegisterAgentCollaborationTools(registry interfaces.ToolRegistry, runtime *agents.CollaborationRuntime, scope AgentToolContext) error {
	if registry == nil {
		return fmt.Errorf("tool registry is required")
	}
	tools := []interfaces.Tool{
		NewRunAgentTool(runtime, scope),
		NewSendAgentMessageTool(runtime),
		NewReadAgentMessagesTool(runtime),
	}
	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("register %s: %w", tool.Name(), err)
		}
	}
	return nil
}

type RunAgentInput struct {
	AgentName     string         `json:"agent_name" validate:"required"`
	TaskID        string         `json:"task_id,omitempty"`
	Instructions  string         `json:"instructions" validate:"required"`
	ChannelID     string         `json:"channel_id,omitempty"`
	Context       map[string]any `json:"context,omitempty"`
	MaxToolRounds int            `json:"max_tool_rounds,omitempty" validate:"omitempty,min=1,max=64"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type RunAgentTool struct {
	BaseTool[RunAgentInput]
	runtime *agents.CollaborationRuntime
	scope   AgentToolContext
}

func NewRunAgentTool(runtime *agents.CollaborationRuntime, scope AgentToolContext) *RunAgentTool {
	t := &RunAgentTool{runtime: runtime, scope: scope}
	t.BaseTool = BaseTool[RunAgentInput]{
		name:        "run_agent",
		description: "Run an isolated ReAct sub-agent by name. Use this when the main agent needs a specialist agent to work independently or in parallel.",
		parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"agent_name": {
					Type:        jsonschema.String,
					Description: "Registered sub-agent name, such as code_generator or code_reviewer.",
				},
				"task_id": {
					Type:        jsonschema.String,
					Description: "Stable task identifier for traceability.",
				},
				"instructions": {
					Type:        jsonschema.String,
					Description: "Concrete task instructions for the sub-agent.",
				},
				"channel_id": {
					Type:        jsonschema.String,
					Description: "Optional collaboration channel shared with related sub-agents.",
				},
				"context": {
					Type:        jsonschema.Object,
					Description: "Sub-agent specific context. This is copied before the sub-agent receives it.",
				},
				"max_tool_rounds": {
					Type:        jsonschema.Number,
					Description: "Optional per-run ReAct tool round limit.",
				},
				"metadata": {
					Type:        jsonschema.Object,
					Description: "Optional trace metadata for frontend or audit display.",
				},
			},
			Required: []string{"agent_name", "instructions"},
		},
		metadata: types.ToolMetadata{Category: types.CategoryExternal},
		ctxFn:    t.execute,
	}
	return t
}

func (t *RunAgentTool) execute(ctx context.Context, p RunAgentInput) (string, error) {
	contextData := cloneToolMap(t.scope.BaseContext)
	mergeToolMap(contextData, p.Context)
	result, err := t.runtime.RunSubAgent(ctx, agents.SubAgentRunRequest{
		AgentName:     p.AgentName,
		TaskID:        p.TaskID,
		Instructions:  p.Instructions,
		ChannelID:     p.ChannelID,
		Pipeline:      t.scope.Pipeline,
		Stage:         t.scope.Stage,
		Context:       contextData,
		MaxToolRounds: p.MaxToolRounds,
		Metadata:      cloneToolMap(p.Metadata),
	})
	if err != nil {
		return "", err
	}
	return marshalToolOutput(result)
}

type SendAgentMessageInput struct {
	ChannelID string         `json:"channel_id" validate:"required"`
	FromAgent string         `json:"from_agent" validate:"required"`
	ToAgent   string         `json:"to_agent,omitempty"`
	Content   string         `json:"content" validate:"required"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type SendAgentMessageTool struct {
	BaseTool[SendAgentMessageInput]
	runtime *agents.CollaborationRuntime
}

func NewSendAgentMessageTool(runtime *agents.CollaborationRuntime) *SendAgentMessageTool {
	t := &SendAgentMessageTool{runtime: runtime}
	t.BaseTool = BaseTool[SendAgentMessageInput]{
		name:        "send_agent_message",
		description: "Send a message to a collaboration channel so sub-agents can coordinate through explicit shared state.",
		parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"channel_id": {Type: jsonschema.String, Description: "Collaboration channel identifier."},
				"from_agent": {Type: jsonschema.String, Description: "Sender agent name."},
				"to_agent":   {Type: jsonschema.String, Description: "Optional intended recipient agent name."},
				"content":    {Type: jsonschema.String, Description: "Message body."},
				"metadata":   {Type: jsonschema.Object, Description: "Optional structured metadata."},
			},
			Required: []string{"channel_id", "from_agent", "content"},
		},
		metadata: types.ToolMetadata{Category: types.CategoryExternal},
		fn:       t.execute,
	}
	return t
}

func (t *SendAgentMessageTool) execute(p SendAgentMessageInput) (string, error) {
	msg, err := t.runtime.SendMessage(p.ChannelID, p.FromAgent, p.ToAgent, p.Content, p.Metadata)
	if err != nil {
		return "", err
	}
	return marshalToolOutput(msg)
}

type ReadAgentMessagesInput struct {
	ChannelID string `json:"channel_id" validate:"required"`
	AfterID   string `json:"after_id,omitempty"`
	ToAgent   string `json:"to_agent,omitempty"`
	Limit     int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}

type ReadAgentMessagesTool struct {
	BaseTool[ReadAgentMessagesInput]
	runtime *agents.CollaborationRuntime
}

func NewReadAgentMessagesTool(runtime *agents.CollaborationRuntime) *ReadAgentMessagesTool {
	t := &ReadAgentMessagesTool{runtime: runtime}
	t.BaseTool = BaseTool[ReadAgentMessagesInput]{
		name:        "read_agent_messages",
		description: "Read messages from a collaboration channel, optionally after a message ID or for a specific recipient.",
		parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"channel_id": {Type: jsonschema.String, Description: "Collaboration channel identifier."},
				"after_id":   {Type: jsonschema.String, Description: "Only return messages after this message ID."},
				"to_agent":   {Type: jsonschema.String, Description: "Only return broadcast messages and messages addressed to this agent."},
				"limit":      {Type: jsonschema.Number, Description: "Maximum number of messages to return."},
			},
			Required: []string{"channel_id"},
		},
		metadata: types.ToolMetadata{Category: types.CategoryExternal},
		fn:       t.execute,
	}
	return t
}

func (t *ReadAgentMessagesTool) execute(p ReadAgentMessagesInput) (string, error) {
	messages, err := t.runtime.ReadMessages(p.ChannelID, p.AfterID, p.ToAgent, p.Limit)
	if err != nil {
		return "", err
	}
	return marshalToolOutput(map[string]any{"messages": messages})
}

func marshalToolOutput(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal tool output: %w", err)
	}
	return string(data), nil
}

func cloneToolMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func mergeToolMap(dst map[string]any, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}
