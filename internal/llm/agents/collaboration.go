package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/plutolove233/co-dream/pkg/types"
)

type CollaborationEventType string

const (
	CollaborationEventAgentStarted   CollaborationEventType = "agent_started"
	CollaborationEventAgentCompleted CollaborationEventType = "agent_completed"
	CollaborationEventAgentFailed    CollaborationEventType = "agent_failed"
	CollaborationEventChannelMessage CollaborationEventType = "channel_message"
)

type CollaborationEvent struct {
	ID        string                 `json:"id"`
	Type      CollaborationEventType `json:"type"`
	AgentName string                 `json:"agent_name,omitempty"`
	ChannelID string                 `json:"channel_id,omitempty"`
	MessageID string                 `json:"message_id,omitempty"`
	Payload   map[string]any         `json:"payload,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type ChannelMessage struct {
	ID        string         `json:"id"`
	ChannelID string         `json:"channel_id"`
	FromAgent string         `json:"from_agent"`
	ToAgent   string         `json:"to_agent,omitempty"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type SubAgentRunRequest struct {
	AgentName     string             `json:"agent_name"`
	TaskID        string             `json:"task_id,omitempty"`
	Instructions  string             `json:"instructions,omitempty"`
	ChannelID     string             `json:"channel_id,omitempty"`
	Pipeline      PipelineDefinition `json:"pipeline,omitempty"`
	Stage         PipelineStage      `json:"stage,omitempty"`
	PlanItem      StagePlanItem      `json:"plan_item,omitempty"`
	Context       map[string]any     `json:"context,omitempty"`
	MaxToolRounds int                `json:"max_tool_rounds,omitempty"`
	Metadata      map[string]any     `json:"metadata,omitempty"`
}

type SubAgentRunResult struct {
	AgentName  string                 `json:"agent_name"`
	TaskID     string                 `json:"task_id,omitempty"`
	Content    string                 `json:"content"`
	ToolCalls  []types.ToolCallResult `json:"tool_calls,omitempty"`
	Trace      []ReActStep            `json:"trace,omitempty"`
	TokenUsage *types.TokenUsage      `json:"token_usage,omitempty"`
	Metadata   map[string]any         `json:"metadata,omitempty"`
}

type AgentRunner interface {
	Run(ctx context.Context, req AgentRunRequest) (AgentRunResult, error)
}

type RunnerFactory func(agent AgentDefinition) (AgentRunner, error)

type CollaborationRuntime struct {
	mu            sync.RWMutex
	registry      *Registry
	runnerFactory RunnerFactory
	channels      map[string][]ChannelMessage
	events        []CollaborationEvent
	nextID        int64
}

func NewCollaborationRuntime(registry *Registry, runnerFactory RunnerFactory) *CollaborationRuntime {
	return &CollaborationRuntime{
		registry:      registry,
		runnerFactory: runnerFactory,
		channels:      make(map[string][]ChannelMessage),
		events:        make([]CollaborationEvent, 0),
	}
}

func (r *CollaborationRuntime) RunSubAgent(ctx context.Context, req SubAgentRunRequest) (SubAgentRunResult, error) {
	if r == nil {
		return SubAgentRunResult{}, fmt.Errorf("collaboration runtime is required")
	}
	if r.registry == nil {
		return SubAgentRunResult{}, fmt.Errorf("agent registry is required")
	}
	if r.runnerFactory == nil {
		return SubAgentRunResult{}, fmt.Errorf("runner factory is required")
	}
	if req.AgentName == "" {
		return SubAgentRunResult{}, fmt.Errorf("agent_name is required")
	}

	agent, ok := r.registry.Get(req.AgentName)
	if !ok {
		return SubAgentRunResult{}, fmt.Errorf("agent %q not registered", req.AgentName)
	}
	r.publish(CollaborationEvent{
		Type:      CollaborationEventAgentStarted,
		AgentName: agent.Name,
		ChannelID: req.ChannelID,
		Payload: map[string]any{
			"task_id":      req.TaskID,
			"instructions": req.Instructions,
		},
	})

	runner, err := r.runnerFactory(agent)
	if err != nil {
		r.publishAgentFailure(agent.Name, req.ChannelID, req.TaskID, err)
		return SubAgentRunResult{}, fmt.Errorf("create runner for agent %s: %w", agent.Name, err)
	}

	planItem := req.PlanItem
	if planItem.TaskID == "" {
		planItem.TaskID = req.TaskID
	}
	if planItem.AgentType == "" {
		planItem.AgentType = agent.Name
	}
	if planItem.Description == "" {
		planItem.Description = req.Instructions
	}
	if planItem.Input == nil {
		planItem.Input = cloneMap(req.Metadata)
	}

	runResult, err := runner.Run(ctx, AgentRunRequest{
		Agent:         agent,
		Pipeline:      req.Pipeline,
		Stage:         req.Stage,
		PlanItem:      planItem,
		Context:       cloneMap(req.Context),
		MaxToolRounds: req.MaxToolRounds,
	})
	if err != nil {
		r.publishAgentFailure(agent.Name, req.ChannelID, req.TaskID, err)
		return SubAgentRunResult{}, err
	}

	result := SubAgentRunResult{
		AgentName:  agent.Name,
		TaskID:     planItem.TaskID,
		Content:    runResult.Content,
		ToolCalls:  runResult.ToolCalls,
		Trace:      runResult.Trace,
		TokenUsage: runResult.TokenUsage,
		Metadata:   cloneMap(req.Metadata),
	}
	r.publish(CollaborationEvent{
		Type:      CollaborationEventAgentCompleted,
		AgentName: agent.Name,
		ChannelID: req.ChannelID,
		Payload: map[string]any{
			"task_id": planItem.TaskID,
			"content": runResult.Content,
		},
	})
	return result, nil
}

func (r *CollaborationRuntime) SendMessage(channelID, fromAgent, toAgent, content string, metadata map[string]any) (ChannelMessage, error) {
	if r == nil {
		return ChannelMessage{}, fmt.Errorf("collaboration runtime is required")
	}
	if channelID == "" {
		return ChannelMessage{}, fmt.Errorf("channel_id is required")
	}
	if fromAgent == "" {
		return ChannelMessage{}, fmt.Errorf("from_agent is required")
	}
	if content == "" {
		return ChannelMessage{}, fmt.Errorf("content is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	msg := ChannelMessage{
		ID:        r.nextLocked("msg"),
		ChannelID: channelID,
		FromAgent: fromAgent,
		ToAgent:   toAgent,
		Content:   content,
		Metadata:  cloneMap(metadata),
		CreatedAt: time.Now().UTC(),
	}
	r.channels[channelID] = append(r.channels[channelID], msg)
	r.appendEventLocked(CollaborationEvent{
		Type:      CollaborationEventChannelMessage,
		AgentName: fromAgent,
		ChannelID: channelID,
		MessageID: msg.ID,
		Payload: map[string]any{
			"to_agent": toAgent,
			"content":  content,
		},
	})
	return msg, nil
}

func (r *CollaborationRuntime) ReadMessages(channelID, afterID, toAgent string, limit int) ([]ChannelMessage, error) {
	if r == nil {
		return nil, fmt.Errorf("collaboration runtime is required")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channel_id is required")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	source := r.channels[channelID]
	messages := make([]ChannelMessage, 0, len(source))
	include := afterID == ""
	for _, msg := range source {
		if !include {
			if msg.ID == afterID {
				include = true
			}
			continue
		}
		if toAgent != "" && msg.ToAgent != "" && msg.ToAgent != toAgent {
			continue
		}
		messages = append(messages, cloneChannelMessage(msg))
		if limit > 0 && len(messages) >= limit {
			break
		}
	}
	return messages, nil
}

func (r *CollaborationRuntime) EventsAfter(afterID string, limit int) []CollaborationEvent {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	events := make([]CollaborationEvent, 0, len(r.events))
	include := afterID == ""
	for _, event := range r.events {
		if !include {
			if event.ID == afterID {
				include = true
			}
			continue
		}
		events = append(events, cloneCollaborationEvent(event))
		if limit > 0 && len(events) >= limit {
			break
		}
	}
	return events
}

func (r *CollaborationRuntime) publishAgentFailure(agentName, channelID, taskID string, err error) {
	r.publish(CollaborationEvent{
		Type:      CollaborationEventAgentFailed,
		AgentName: agentName,
		ChannelID: channelID,
		Payload: map[string]any{
			"task_id": taskID,
			"error":   err.Error(),
		},
	})
}

func (r *CollaborationRuntime) publish(event CollaborationEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.appendEventLocked(event)
}

func (r *CollaborationRuntime) appendEventLocked(event CollaborationEvent) {
	event.ID = r.nextLocked("evt")
	event.CreatedAt = time.Now().UTC()
	event.Payload = cloneMap(event.Payload)
	r.events = append(r.events, event)
}

func (r *CollaborationRuntime) nextLocked(prefix string) string {
	r.nextID++
	return fmt.Sprintf("%s-%d", prefix, r.nextID)
}

func cloneChannelMessage(msg ChannelMessage) ChannelMessage {
	msg.Metadata = cloneMap(msg.Metadata)
	return msg
}

func cloneCollaborationEvent(event CollaborationEvent) CollaborationEvent {
	event.Payload = cloneMap(event.Payload)
	return event
}
