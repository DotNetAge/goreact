package goreact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DotNetAge/goreact/core"
	"gopkg.in/yaml.v3"
)

type AgentRegistry struct {
	path   string
	agents map[string]*core.AgentConfig
}

// LoadFrom loads all agent definition files (.md) from the specified directory,
// parses their YAML frontmatter and system prompt body, and returns an AgentRegistry.
func LoadAgentsFrom(dir string) (*AgentRegistry, error) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	registry := &AgentRegistry{
		path:   absPath,
		agents: make(map[string]*core.AgentConfig),
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", absPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			filePath := filepath.Join(absPath, entry.Name())
			agent, err := parseAgentFile(filePath)
			if err != nil {
				// skip invalid files but log?
				continue
			}
			registry.agents[agent.Name] = agent
		}
	}
	return registry, nil
}

// parseAgentFile reads a markdown file, extracts YAML frontmatter and body.
func parseAgentFile(filePath string) (*core.AgentConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// normalize line endings to \n
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	content = strings.TrimLeft(content, "\n\r\t ")

	// expect first line to be "---"
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("invalid agent file format, missing frontmatter delimiter")
	}
	// find the next "---" line after the first
	lines := strings.Split(content, "\n")
	var frontmatterLines []string
	var bodyLines []string
	delimCount := 0
	inBody := false
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			delimCount++
			if delimCount == 2 {
				inBody = true
				continue
			}
			continue
		}
		if i == 0 {
			continue // skip first delimiter line
		}
		if !inBody {
			frontmatterLines = append(frontmatterLines, line)
		} else {
			bodyLines = append(bodyLines, line)
		}
	}
	if delimCount < 2 {
		return nil, fmt.Errorf("invalid agent file format, missing closing frontmatter delimiter")
	}
	frontmatterYAML := strings.Join(frontmatterLines, "\n")
	body := strings.Join(bodyLines, "\n")
	body = strings.TrimSpace(body)

	// parse YAML frontmatter into a map first
	var meta map[string]any
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	agent := &core.AgentConfig{}
	// map fields
	if nameVal, ok := meta["name"].(string); ok {
		agent.Name = nameVal
	}
	if domainVal, ok := meta["domain"].(string); ok {
		agent.Domain = domainVal
	} else if titleVal, ok := meta["title"].(string); ok {
		agent.Domain = titleVal
	}
	if descVal, ok := meta["description"].(string); ok {
		agent.Description = descVal
	}
	if modelVal, ok := meta["model"].(string); ok {
		agent.Model = modelVal
	}
	// tools can be a string (comma-separated) or a slice
	if toolsVal, ok := meta["tools"]; ok {
		switch v := toolsVal.(type) {
		case string:
			// split by comma and trim spaces
			if v != "" {
				parts := strings.Split(v, ",")
				for _, p := range parts {
					agent.Tools = append(agent.Tools, strings.TrimSpace(p))
				}
			}
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					agent.Tools = append(agent.Tools, s)
				}
			}
		}
	}
	// also check for "allowed-tools" as alternative key
	if allowedVal, ok := meta["allowed-tools"]; ok && len(agent.Tools) == 0 {
		switch v := allowedVal.(type) {
		case string:
			if v != "" {
				parts := strings.Split(v, ",")
				for _, p := range parts {
					agent.Tools = append(agent.Tools, strings.TrimSpace(p))
				}
			}
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					agent.Tools = append(agent.Tools, s)
				}
			}
		}
	}
	agent.SystemPrompt = body
	return agent, nil
}

func (r *AgentRegistry) Get(name string) *core.AgentConfig {
	return r.agents[name]
}

func (r *AgentRegistry) List() []*core.AgentConfig {
	var agents []*core.AgentConfig
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}
	return agents
}

// It does not store the config in the registry.
func (r *AgentRegistry) Read(file string) (*core.AgentConfig, error) {
	absPath := filepath.Join(r.path, file)
	return parseAgentFile(absPath)
}

// Remove removes an agent from the registry and deletes its markdown file.
func (r *AgentRegistry) Remove(name string) error {
	_, exists := r.agents[name]
	if !exists {
		return fmt.Errorf("agent %s not found", name)
	}
	fileName := strings.ToLower(name) + ".md"
	filePath := filepath.Join(r.path, fileName)
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file %s: %w", filePath, err)
	}
	delete(r.agents, name)
	return nil
}

// SaveTo saves an agent config as a markdown file in the registry directory.
func (r *AgentRegistry) SaveTo(agent *core.AgentConfig) error {
	if agent.Name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}
	fileName := strings.ToLower(agent.Name) + ".md"
	filePath := filepath.Join(r.path, fileName)

	// prepare frontmatter
	meta := make(map[string]any)
	meta["name"] = agent.Name
	if agent.Domain != "" {
		meta["domain"] = agent.Domain
	}
	if agent.Description != "" {
		meta["description"] = agent.Description
	}
	if agent.Model != "" {
		meta["model"] = agent.Model
	}
	if len(agent.Tools) > 0 {
		meta["tools"] = agent.Tools
	}

	yamlData, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML frontmatter: %w", err)
	}

	content := fmt.Sprintf("---\n%s---\n%s", string(yamlData), agent.SystemPrompt)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}
	r.agents[agent.Name] = agent
	return nil
}
