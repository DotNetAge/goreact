package tools

import (
	"context"
	"fmt"

	"github.com/DotNetAge/goreact/core"
)

// ModelListTool lists all available models with their names and descriptions.
type ModelListTool struct {
	registry core.ModelRegistry
}

// NewModelListTool creates a ModelListTool.
func NewModelListTool(registry core.ModelRegistry) *ModelListTool {
	return &ModelListTool{registry: registry}
}

func (t *ModelListTool) Info() *core.ToolInfo {
	return &core.ToolInfo{
		Name:        "ModelList",
		Description: "List all available models with their names and descriptions. Use this before CreateAgent to pick the right model for a new agent.",
		Prompt: `Query all models registered in the system. Each entry shows: name and description.

Use this tool when:
- Creating a new agent and you need to choose an appropriate model for its role.
- Checking what models are available before configuring a delegated agent.
- The model name from ModelList goes into CreateAgent's "model" parameter.`,
		Tags:       []string{"model", "list", "query", "config"},
		IsReadOnly: true,
	}
}

func (t *ModelListTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	if t.registry == nil {
		return nil, fmt.Errorf("model registry not configured")
	}

	names := t.registry.List()
	if len(names) == 0 {
		return map[string]any{
			"models": []map[string]any{},
			"count":  0,
			"message": "No models registered",
		}, nil
	}

	var models []map[string]any
	for _, name := range names {
		cfg, err := t.registry.Get(name)
		entry := map[string]any{"name": name}
		if err == nil && cfg != nil {
			entry["description"] = cfg.Description
		}
		models = append(models, entry)
	}

	return map[string]any{
		"models": models,
		"count":  len(models),
	}, nil
}
