package tools

import (
	"fmt"
	"strings"

	"github.com/plutolove233/co-dream/internal/skill"
	"github.com/plutolove233/co-dream/pkg/types"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// SkillInput is the payload accepted by the skill tool.
type SkillInput struct {
	Skill   string `json:"skill" validate:"required"`
	Context string `json:"context"`
}

// LoadSkillTool loads a registered skill into the conversation context.
type LoadSkillTool struct {
	BaseTool[SkillInput]
}

// NewSkillTool creates a skill-loading tool bound to a skill registry.
func NewLoadSkillTool(registry *skill.SkillRegistry) *LoadSkillTool {
	return &LoadSkillTool{
		BaseTool: BaseTool[SkillInput]{
			name:        "load_skill",
			description: "Load a registered skill into the current context.",
			metadata: types.ToolMetadata{
				Category:   types.CategorySystem,
				SafeToSkip: false,
				MaxRetries: 0,
			},
			fn:            skillExecute(registry),
			extraValidate: skillValidate(registry),
		},
	}
}

func (t *LoadSkillTool) Parameters() jsonschema.Definition {
	return jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"skill": {
				Type:        jsonschema.String,
				Description: "The name of the skill to load (must be registered).",
			},
			"context": {
				Type:        jsonschema.String,
				Description: "Optional context to provide when loading the skill.",
			},
		},
		Required: []string{"skill"},
	}
}

func skillExecute(registry *skill.SkillRegistry) func(SkillInput) (string, error) {
	return func(input SkillInput) (string, error) {
		skillName := strings.TrimSpace(input.Skill)
		s, ok := registry.Get(skillName)
		if !ok {
			return "", fmt.Errorf("skill %q is not registered", skillName)
		}

		return skill.BuildContext(s, input.Context), nil
	}
}

func skillValidate(registry *skill.SkillRegistry) func(SkillInput) error {
	return func(input SkillInput) error {
		if registry == nil {
			return fmt.Errorf("skill registry is not configured")
		}
		skillName := strings.TrimSpace(input.Skill)
		if skillName == "" {
			return fmt.Errorf("skill name cannot be empty")
		}
		if _, ok := registry.Get(skillName); !ok {
			return fmt.Errorf("skill %q is not registered", skillName)
		}
		return nil
	}
}
