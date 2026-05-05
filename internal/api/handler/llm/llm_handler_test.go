package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/plutolove233/co-dream/internal/api/service"
	"github.com/plutolove233/co-dream/pkg/types"
)

type fakeStreamer struct {
	events []types.CompleteEvent
	err    error
	req    service.LLMStreamRequest
}

type closeNotifyRecorder struct {
	*httptest.ResponseRecorder
}

func (r closeNotifyRecorder) CloseNotify() <-chan bool {
	return make(chan bool)
}

func (s *fakeStreamer) Stream(_ context.Context, req service.LLMStreamRequest) (<-chan types.CompleteEvent, error) {
	s.req = req
	if s.err != nil {
		return nil, s.err
	}

	events := make(chan types.CompleteEvent, len(s.events))
	for _, event := range s.events {
		events <- event
	}
	close(events)
	return events, nil
}

func TestChatStreamWritesSSEEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)

	streamer := &fakeStreamer{
		events: []types.CompleteEvent{
			{Delta: "hello"},
			{Result: &types.CompleteResult{
				Content:      "hello",
				FinishReason: "stop",
				Usage:        &types.TokenUsage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
			}},
		},
	}
	api := NewAPIWithStreamer(streamer)

	router := gin.New()
	router.POST("/stream", api.ChatStream)

	body := `{"system":"be concise","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/stream", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := &closeNotifyRecorder{ResponseRecorder: httptest.NewRecorder()}

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if ct := resp.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %q", ct)
	}

	got := resp.Body.String()
	for _, want := range []string{
		"event:delta",
		`data:{"delta":"hello"}`,
		"event:result",
		`"content":"hello"`,
		"event:done",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected response to contain %q, got %q", want, got)
		}
	}
	if streamer.req.System != "be concise" {
		t.Fatalf("expected request system to be bound, got %q", streamer.req.System)
	}
}

func TestChatStreamRequiresMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/stream", NewAPIWithStreamer(&fakeStreamer{}).ChatStream)

	req := httptest.NewRequest(http.MethodPost, "/stream", strings.NewReader(`{"messages":[]}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), `"code":"4000"`) {
		t.Fatalf("expected parameter error response, got %q", resp.Body.String())
	}
}
