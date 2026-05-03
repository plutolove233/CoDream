package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type AgentDefinition struct {
	Path         string   `json:"path"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Model        string   `json:"model,omitempty"`
	Tools        []string `json:"tools,omitempty"`
	Instructions string   `json:"instructions"`
}

type agentFrontmatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Model       string   `yaml:"model"`
	Tools       []string `yaml:"tools"`
}

func ParseAgentMarkdown(markdown string, path string) (AgentDefinition, error) {
	content := strings.TrimPrefix(markdown, "---\n")
	if content == markdown {
		return AgentDefinition{}, fmt.Errorf("no frontmatter found in %s", path)
	}

	parts := strings.SplitN(content, "---\n", 2)
	if len(parts) != 2 {
		return AgentDefinition{}, fmt.Errorf("invalid frontmatter format in %s", path)
	}

	var fm agentFrontmatter
	if err := yaml.Unmarshal([]byte(parts[0]), &fm); err != nil {
		return AgentDefinition{}, fmt.Errorf("parse agent frontmatter: %w", err)
	}
	if fm.Name == "" {
		return AgentDefinition{}, fmt.Errorf("agent name is required in %s", path)
	}

	return AgentDefinition{
		Path:         path,
		Name:         fm.Name,
		Description:  fm.Description,
		Model:        fm.Model,
		Tools:        fm.Tools,
		Instructions: strings.TrimSpace(parts[1]),
	}, nil
}

type Loader struct {
	agentsDir string
}

func NewLoader(agentsDir string) *Loader {
	return &Loader{agentsDir: agentsDir}
}

func (l *Loader) LoadAgents() ([]AgentDefinition, error) {
	entries, err := os.ReadDir(l.agentsDir)
	if err != nil {
		return nil, fmt.Errorf("read agents directory: %w", err)
	}

	agents := make([]AgentDefinition, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		agentPath := filepath.Join(l.agentsDir, entry.Name(), "AGENT.md")
		agent, err := l.LoadFromFile(agentPath)
		if err != nil {
			continue
		}
		agents = append(agents, agent)
	}
	return agents, nil
}

func (l *Loader) LoadFromFile(path string) (AgentDefinition, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return AgentDefinition{}, fmt.Errorf("read agent file: %w", err)
	}
	return ParseAgentMarkdown(string(content), path)
}

type Registry struct {
	mu     sync.RWMutex
	agents map[string]AgentDefinition
}

func NewRegistry() *Registry {
	return &Registry{agents: make(map[string]AgentDefinition)}
}

func (r *Registry) Register(agent AgentDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if agent.Name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}
	if _, exists := r.agents[agent.Name]; exists {
		return fmt.Errorf("agent %q is already registered", agent.Name)
	}
	r.agents[agent.Name] = agent
	return nil
}

func (r *Registry) Get(name string) (AgentDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.agents[name]
	return agent, ok
}

func (r *Registry) List() []AgentDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]AgentDefinition, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}
	return agents
}

func (r *Registry) LoadFromDir(dir string) error {
	loader := NewLoader(dir)
	agents, err := loader.LoadAgents()
	if err != nil {
		return err
	}
	for _, agent := range agents {
		if err := r.Register(agent); err != nil {
			continue
		}
	}
	return nil
}
