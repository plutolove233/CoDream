package service

import (
	"context"
	"errors"
	"os"

	"github.com/plutolove233/co-dream/internal/llm"
	"github.com/plutolove233/co-dream/pkg/interfaces"
	"github.com/plutolove233/co-dream/pkg/types"
)

var ErrLLMAPIKeyRequired = errors.New("llm api key is required")

type LLMStreamRequest struct {
	System   string          `json:"system"`
	Model    string          `json:"model"`
	Messages []types.Message `json:"messages" binding:"required,min=1"`
}

type LLMStreamService struct {
	registry interfaces.ToolRegistry
}

func NewLLMStreamService(registry interfaces.ToolRegistry) *LLMStreamService {
	return &LLMStreamService{registry: registry}
}

func (s *LLMStreamService) Stream(ctx context.Context, req LLMStreamRequest) (<-chan types.CompleteEvent, error) {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return nil, ErrLLMAPIKeyRequired
	}

	model := os.Getenv("MODEL_EPIP")
	baseURL := os.Getenv("BASE_URL")

	registry := s.registry

	client := llm.NewClient(apiKey, baseURL, model, registry)
	return client.Complete(ctx, req.Messages, req.System)
}
