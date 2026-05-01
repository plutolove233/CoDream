package interfaces

import (
	"context"

	"github.com/plutolove233/co-dream/pkg/types"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input []byte) (string, error)
	Parameters() jsonschema.Definition
	Metadata() types.ToolMetadata
}

// ToolRegistry defines the interface for tool lookup.
type ToolRegistry interface {
	Get(name string) (Tool, bool)
	EnabledTools() []Tool
	Register(t Tool) error
	Unregister(name string)
	Filter(allowed []string) ToolRegistry
	List() []Tool
}
