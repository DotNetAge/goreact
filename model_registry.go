package goreact

import (
	"fmt"
	"os"

	"github.com/DotNetAge/goreact/core"
	"gopkg.in/yaml.v3"
)

type ModelsConfig struct {
	Models []core.ModelConfig `yaml:"models"`
}

type ModelRegistry struct {
	settingFile string
	models      map[string]*core.ModelConfig
}

func LoadModels(path string) (*ModelRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read models file: %w", err)
	}

	// var configs []core.ModelConfig
	var configs ModelsConfig
	if err := yaml.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal models YAML: %w", err)
	}

	models := make(map[string]*core.ModelConfig, len(configs.Models))
	for _, cfg := range configs.Models {
		if cfg.Name == "" {
			return nil, fmt.Errorf("model config missing name")
		}
		models[cfg.Name] = &cfg
	}

	return &ModelRegistry{
		settingFile: path,
		models:      models,
	}, nil
}

func (m *ModelRegistry) Get(name string) *core.ModelConfig {
	return m.models[name]
}
func (m *ModelRegistry) List() []*core.ModelConfig {
	var result []*core.ModelConfig
	for _, model := range m.models {
		result = append(result, model)
	}
	return result
}

func (m *ModelRegistry) Save(model *core.ModelConfig) error {
	if model.Name == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	// if model.ID == "" {
	// 	return fmt.Errorf("model name cannot be empty")
	// }

	if m.models == nil {
		m.models = make(map[string]*core.ModelConfig)
	}
	m.models[model.Name] = model

	configs := make([]core.ModelConfig, 0, len(m.models))
	for _, cfg := range m.models {
		if cfg == nil {
			continue // skip nil entries, though they shouldn't exist
		}
		configs = append(configs, *cfg)
	}

	data, err := yaml.Marshal(configs)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	if err := os.WriteFile(m.settingFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write models file: %w", err)
	}
	return nil
}

func (m *ModelRegistry) Models() []core.ModelConfig {
	if m.models == nil {
		return nil
	}
	configs := make([]core.ModelConfig, 0, len(m.models))
	for _, cfg := range m.models {
		if cfg == nil {
			continue // skip nil entries, though they shouldn't exist
		}
		configs = append(configs, *cfg)
	}
	return configs
}
