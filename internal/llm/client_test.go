package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/plutolove233/co-dream/pkg/types"
	openai "github.com/sashabaranov/go-openai"
)

func TestHandleCompletionStreamResponseHandlesChunksWithoutUsage(t *testing.T) {
	events := make(chan types.CompleteEvent, 2)
	var content strings.Builder
	var finishReason string
	usage := &types.TokenUsage{}
	var toolCallOrder []int
	toolCallsByIdx := map[int]*partialToolCall{}

	ok := handleCompletionStreamResponse(context.Background(), events, openai.ChatCompletionStreamResponse{
		Choices: []openai.ChatCompletionStreamChoice{
			{
				Delta: openai.ChatCompletionStreamChoiceDelta{
					Content: "hello",
				},
			},
		},
	}, &content, &finishReason, usage, &toolCallOrder, toolCallsByIdx)
	if !ok {
		t.Fatal("expected stream response handling to continue")
	}

	event := <-events
	if event.Delta != "hello" {
		t.Fatalf("expected delta event, got %+v", event)
	}
	if content.String() != "hello" {
		t.Fatalf("expected content to be accumulated, got %q", content.String())
	}
	if *usage != (types.TokenUsage{}) {
		t.Fatalf("expected zero usage when response usage is nil, got %+v", usage)
	}
}

func TestHandleCompletionStreamResponseCollectsUsageFromChoiceLessChunk(t *testing.T) {
	events := make(chan types.CompleteEvent, 1)
	var content strings.Builder
	var finishReason string
	usage := &types.TokenUsage{}
	var toolCallOrder []int
	toolCallsByIdx := map[int]*partialToolCall{}

	ok := handleCompletionStreamResponse(context.Background(), events, openai.ChatCompletionStreamResponse{
		Choices: nil,
		Usage: &openai.Usage{
			PromptTokens:     2,
			CompletionTokens: 3,
			TotalTokens:      5,
		},
	}, &content, &finishReason, usage, &toolCallOrder, toolCallsByIdx)
	if !ok {
		t.Fatal("expected stream response handling to continue")
	}

	if len(events) != 0 {
		t.Fatalf("expected no delta event for choice-less usage chunk, got %d events", len(events))
	}
	if *usage != (types.TokenUsage{PromptTokens: 2, CompletionTokens: 3, TotalTokens: 5}) {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}
