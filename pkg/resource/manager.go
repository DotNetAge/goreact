// Package resource provides resource management for the goreact framework.
package resource

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	goreactskill "github.com/DotNetAge/goreact/pkg/skill"
)

// ModelFeatures represents model capabilities
type ModelFeatures struct {
	// Vision indicates if the model supports vision/image input
	Vision bool `json:"vision" yaml:"vision"`
	
	// ToolCall indicates if the model supports tool calling
	ToolCall bool `json:"tool_call" yaml:"tool_call"`
	
	// Streaming indicates if the model supports streaming output
	Streaming bool `json:"streaming" yaml:"streaming"`
}

// Model represents a LLM model configuration
type Model struct {
	// Name is the internal name used by GoReAct
	Name string `json:"name" yaml:"name"`
	
	// Provider is the model provider (openai, anthropic, ollama, etc.)
	Provider string `json:"provider" yaml:"provider"`
	
	// ProviderModelName is the model identifier used by the provider
	ProviderModelName string `json:"provider_model_name" yaml:"provider_model_name"`
	
	// BaseURL is the API base URL
	BaseURL string `json:"base_url" yaml:"base_url"`
	
	// APIKey is the API key for authentication
	APIKey string `json:"api_key" yaml:"api_key"`
	
	// Temperature is the model temperature
	Temperature float64 `json:"temperature" yaml:"temperature"`
	
	// MaxTokens is the maximum tokens for the model
	MaxTokens int `json:"max_tokens" yaml:"max_tokens"`
	
	// Timeout is the request timeout
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
	
	// Features contains model capabilities
	Features ModelFeatures `json:"features" yaml:"features"`
}

// ResourceManager manages all resources (agents, tools, skills, models)
type ResourceManager struct {
	mu      sync.RWMutex
	agents  map[string]any // Agent configurations
	tools   map[string]any // Tool configurations
	skills  map[string]any // Skill configurations
	models  map[string]any // Model configurations
	
	// Agent-Tool and Agent-Skill mappings
	agentTools  map[string][]string // agent name -> tool names
	agentSkills map[string][]string // agent name -> skill names
	
	// Plan cache for skill execution
	planCache map[string]*goreactskill.SkillExecutionPlan
	
	// Paths
	DocumentPath string
	SkillPath    string
	ToolPath     string
}

// NewResourceManager creates a new ResourceManager
func NewResourceManager() *ResourceManager {
	return &ResourceManager{
		agents:      make(map[string]any),
		tools:       make(map[string]any),
		skills:      make(map[string]any),
		models:      make(map[string]any),
		agentTools:  make(map[string][]string),
		agentSkills: make(map[string][]string),
		planCache:   make(map[string]*goreactskill.SkillExecutionPlan),
	}
}

// RegisterAgent registers an agent
func (rm *ResourceManager) RegisterAgent(name string, agent any) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if _, exists := rm.agents[name]; exists {
		return fmt.Errorf("agent %s already registered", name)
	}
	
	rm.agents[name] = agent
	return nil
}

// UnregisterAgent unregisters an agent
func (rm *ResourceManager) UnregisterAgent(name string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.agents, name)
}

// GetAgent retrieves an agent by name
func (rm *ResourceManager) GetAgent(name string) (any, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	agent, exists := rm.agents[name]
	return agent, exists
}

// GetAgents returns all registered agents
func (rm *ResourceManager) GetAgents() map[string]any {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	agents := make(map[string]any, len(rm.agents))
	for k, v := range rm.agents {
		agents[k] = v
	}
	return agents
}

// RegisterTool registers a tool
func (rm *ResourceManager) RegisterTool(name string, tool any) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if _, exists := rm.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}
	
	rm.tools[name] = tool
	return nil
}

// UnregisterTool unregisters a tool
func (rm *ResourceManager) UnregisterTool(name string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.tools, name)
}

// GetTool retrieves a tool by name
func (rm *ResourceManager) GetTool(name string) (any, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	tool, exists := rm.tools[name]
	return tool, exists
}

// GetTools returns all registered tools
func (rm *ResourceManager) GetTools() map[string]any {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	tools := make(map[string]any, len(rm.tools))
	for k, v := range rm.tools {
		tools[k] = v
	}
	return tools
}

// GetToolsByAgent returns tools for an agent
func (rm *ResourceManager) GetToolsByAgent(agentName string) []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	if tools, exists := rm.agentTools[agentName]; exists {
		return tools
	}
	return []string{}
}

// RegisterAgentTools associates tools with an agent
func (rm *ResourceManager) RegisterAgentTools(agentName string, toolNames []string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Verify agent exists
	if _, exists := rm.agents[agentName]; !exists {
		return fmt.Errorf("agent %s not registered", agentName)
	}
	
	// Verify all tools exist
	for _, toolName := range toolNames {
		if _, exists := rm.tools[toolName]; !exists {
			return fmt.Errorf("tool %s not registered", toolName)
		}
	}
	
	rm.agentTools[agentName] = toolNames
	return nil
}

// RegisterSkill registers a skill
func (rm *ResourceManager) RegisterSkill(name string, skill any) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if _, exists := rm.skills[name]; exists {
		return fmt.Errorf("skill %s already registered", name)
	}
	
	rm.skills[name] = skill
	return nil
}

// UnregisterSkill unregisters a skill
func (rm *ResourceManager) UnregisterSkill(name string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.skills, name)
}

// GetSkill retrieves a skill by name
func (rm *ResourceManager) GetSkill(name string) (any, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	skill, exists := rm.skills[name]
	return skill, exists
}

// GetSkills returns all registered skills
func (rm *ResourceManager) GetSkills() map[string]any {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	skills := make(map[string]any, len(rm.skills))
	for k, v := range rm.skills {
		skills[k] = v
	}
	return skills
}

// GetSkillsByAgent returns skills for an agent
func (rm *ResourceManager) GetSkillsByAgent(agentName string) []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	if skills, exists := rm.agentSkills[agentName]; exists {
		return skills
	}
	return []string{}
}

// RegisterAgentSkills associates skills with an agent
func (rm *ResourceManager) RegisterAgentSkills(agentName string, skillNames []string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Verify agent exists
	if _, exists := rm.agents[agentName]; !exists {
		return fmt.Errorf("agent %s not registered", agentName)
	}
	
	// Verify all skills exist
	for _, skillName := range skillNames {
		if _, exists := rm.skills[skillName]; !exists {
			return fmt.Errorf("skill %s not registered", skillName)
		}
	}
	
	rm.agentSkills[agentName] = skillNames
	return nil
}

// RegisterModel registers a model
func (rm *ResourceManager) RegisterModel(name string, model any) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if _, exists := rm.models[name]; exists {
		return fmt.Errorf("model %s already registered", name)
	}
	
	rm.models[name] = model
	return nil
}

// UnregisterModel unregisters a model
func (rm *ResourceManager) UnregisterModel(name string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.models, name)
}

// GetModel retrieves a model by name
func (rm *ResourceManager) GetModel(name string) (any, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	model, exists := rm.models[name]
	return model, exists
}

// GetModels returns all registered models
func (rm *ResourceManager) GetModels() map[string]any {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	models := make(map[string]any, len(rm.models))
	for k, v := range rm.models {
		models[k] = v
	}
	return models
}

// SetDocumentPath sets the document path
func (rm *ResourceManager) SetDocumentPath(path string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.DocumentPath = path
}

// SetSkillPath sets the skill path
func (rm *ResourceManager) SetSkillPath(path string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.SkillPath = path
}

// SetToolPath sets the tool path
func (rm *ResourceManager) SetToolPath(path string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.ToolPath = path
}

// Clear clears all resources
func (rm *ResourceManager) Clear() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.agents = make(map[string]any)
	rm.tools = make(map[string]any)
	rm.skills = make(map[string]any)
	rm.models = make(map[string]any)
}

// Load loads all resources into memory for indexing
// This is called by Memory to index resources into GraphDB and VectorDB
func (rm *ResourceManager) Load() error {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	// Resources are already loaded through registration
	// This method returns all resources for Memory to index
	// Memory will call GetAgents, GetTools, GetSkills, GetModels
	// to retrieve all registered resources for indexing
	return nil
}

// GetAllResources returns all resources for Memory indexing
func (rm *ResourceManager) GetAllResources() map[string]map[string]any {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	return map[string]map[string]any{
		"agents": rm.agents,
		"tools":  rm.tools,
		"skills": rm.skills,
		"models": rm.models,
	}
}

// Global resource manager instance
var globalManager = NewResourceManager()

// RegisterAgent registers an agent to the global manager
func RegisterAgent(name string, agent any) error {
	return globalManager.RegisterAgent(name, agent)
}

// UnregisterAgent unregisters an agent from the global manager
func UnregisterAgent(name string) {
	globalManager.UnregisterAgent(name)
}

// GetAgent retrieves an agent from the global manager
func GetAgent(name string) (any, bool) {
	return globalManager.GetAgent(name)
}

// GetAgents returns all registered agents from the global manager
func GetAgents() map[string]any {
	return globalManager.GetAgents()
}

// RegisterTool registers a tool to the global manager
func RegisterTool(name string, tool any) error {
	return globalManager.RegisterTool(name, tool)
}

// UnregisterTool unregisters a tool from the global manager
func UnregisterTool(name string) {
	globalManager.UnregisterTool(name)
}

// GetTool retrieves a tool from the global manager
func GetTool(name string) (any, bool) {
	return globalManager.GetTool(name)
}

// GetTools returns all registered tools from the global manager
func GetTools() map[string]any {
	return globalManager.GetTools()
}

// RegisterSkill registers a skill to the global manager
func RegisterSkill(name string, skill any) error {
	return globalManager.RegisterSkill(name, skill)
}

// UnregisterSkill unregisters a skill from the global manager
func UnregisterSkill(name string) {
	globalManager.UnregisterSkill(name)
}

// GetSkill retrieves a skill from the global manager
func GetSkill(name string) (any, bool) {
	return globalManager.GetSkill(name)
}

// GetSkills returns all registered skills from the global manager
func GetSkills() map[string]any {
	return globalManager.GetSkills()
}

// RegisterModel registers a model to the global manager
func RegisterModel(name string, model any) error {
	return globalManager.RegisterModel(name, model)
}

// UnregisterModel unregisters a model from the global manager
func UnregisterModel(name string) {
	globalManager.UnregisterModel(name)
}

// GetModel retrieves a model from the global manager
func GetModel(name string) (any, bool) {
	return globalManager.GetModel(name)
}

// GetModels returns all registered models from the global manager
func GetModels() map[string]any {
	return globalManager.GetModels()
}

// ScanSkills scans a directory for skill definitions (SKILL.md files)
func (rm *ResourceManager) ScanSkills(skillPath string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.SkillPath = skillPath

	// Walk the skill directory
	return filepath.Walk(skillPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Look for SKILL.md files
		if !info.IsDir() && strings.EqualFold(info.Name(), "SKILL.md") {
			skillDir := filepath.Dir(path)
			skillName := filepath.Base(skillDir)

			// Read and parse SKILL.md
			content, err := os.ReadFile(path)
			if err != nil {
				return nil // Skip files we can't read
			}

			// Parse skill
			parser := goreactskill.NewSkillParser()
			skill, err := parser.Parse(string(content), skillDir)
			if err != nil {
				return nil // Skip parse errors
			}

			// Ensure name matches directory
			skill.Name = skillName

			// Register skill
			rm.skills[skillName] = skill
		}

		return nil
	})
}

// ScanTools scans a directory for tool definitions
func (rm *ResourceManager) ScanTools(toolPath string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.ToolPath = toolPath

	// Check if directory exists
	if _, err := os.Stat(toolPath); os.IsNotExist(err) {
		return fmt.Errorf("tool path does not exist: %s", toolPath)
	}

	// Walk the tool directory looking for tool definitions
	return filepath.Walk(toolPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Look for TOOL.md files or executable scripts
		if !info.IsDir() {
			fileName := info.Name()
			
			// Check for TOOL.md
			if strings.EqualFold(fileName, "TOOL.md") {
				toolDir := filepath.Dir(path)
				toolName := filepath.Base(toolDir)
				
				// Read and parse TOOL.md
				content, err := os.ReadFile(path)
				if err != nil {
					return nil // Skip files we can't read
				}
				
				// Parse tool definition
				tool := rm.parseToolDefinition(string(content), toolName)
				if tool != nil {
					rm.tools[toolName] = tool
				}
			}
			
			// Check for executable scripts
			if strings.HasSuffix(fileName, ".py") || strings.HasSuffix(fileName, ".sh") {
				toolName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
				
				// Create tool entry for script
				rm.tools[toolName] = map[string]any{
					"name":        toolName,
					"path":        path,
					"type":        strings.TrimPrefix(filepath.Ext(fileName), "."),
					"description": fmt.Sprintf("Auto-detected tool: %s", toolName),
				}
			}
		}

		return nil
	})
}

// parseToolDefinition parses a TOOL.md file and returns a tool configuration
func (rm *ResourceManager) parseToolDefinition(content, toolName string) map[string]any {
	// Parse frontmatter and content
	tool := map[string]any{
		"name": toolName,
	}
	
	// Check for frontmatter
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			// Parse frontmatter
			frontmatter := parts[1]
			for _, line := range strings.Split(frontmatter, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				
				kv := strings.SplitN(line, ":", 2)
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					value := strings.TrimSpace(kv[1])
					tool[key] = value
				}
			}
			
			// Store body content
			tool["content"] = strings.TrimSpace(parts[2])
		}
	} else {
		tool["content"] = content
	}
	
	return tool
}

// ScanAgents scans a directory for agent definitions
func (rm *ResourceManager) ScanAgents(agentPath string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Check if directory exists
	if _, err := os.Stat(agentPath); os.IsNotExist(err) {
		return fmt.Errorf("agent path does not exist: %s", agentPath)
	}

	// Walk the agent directory looking for agent definitions
	return filepath.Walk(agentPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Look for AGENT.md files
		if !info.IsDir() && strings.EqualFold(info.Name(), "AGENT.md") {
			agentDir := filepath.Dir(path)
			agentName := filepath.Base(agentDir)
			
			// Read and parse AGENT.md
			content, err := os.ReadFile(path)
			if err != nil {
				return nil // Skip files we can't read
			}
			
			// Parse agent definition
			agent := rm.parseAgentDefinition(string(content), agentName)
			if agent != nil {
				rm.agents[agentName] = agent
			}
		}

		return nil
	})
}

// parseAgentDefinition parses an AGENT.md file and returns an agent configuration
func (rm *ResourceManager) parseAgentDefinition(content, agentName string) map[string]any {
	agent := map[string]any{
		"name": agentName,
	}
	
	// Check for frontmatter
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			// Parse frontmatter
			frontmatter := parts[1]
			for _, line := range strings.Split(frontmatter, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				
				kv := strings.SplitN(line, ":", 2)
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					value := strings.TrimSpace(kv[1])
					
					// Handle array fields
					if key == "tools" || key == "skills" {
						items := strings.Fields(value)
						agent[key] = items
					} else {
						agent[key] = value
					}
				}
			}
			
			// Store system prompt from body
			agent["system_prompt"] = strings.TrimSpace(parts[2])
		}
	} else {
		agent["system_prompt"] = content
	}
	
	return agent
}

// ScanAll scans all resource directories
func (rm *ResourceManager) ScanAll(basePath string) error {
	// Scan skills
	skillPath := filepath.Join(basePath, "skills")
	if _, err := os.Stat(skillPath); err == nil {
		_ = rm.ScanSkills(skillPath)
	}

	// Scan tools
	toolPath := filepath.Join(basePath, "tools")
	if _, err := os.Stat(toolPath); err == nil {
		_ = rm.ScanTools(toolPath)
	}

	// Scan agents
	agentPath := filepath.Join(basePath, "agents")
	if _, err := os.Stat(agentPath); err == nil {
		_ = rm.ScanAgents(agentPath)
	}

	return nil
}

// GetSkillTyped retrieves a typed skill by name
func (rm *ResourceManager) GetSkillTyped(name string) (*goreactskill.Skill, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	skill, exists := rm.skills[name]
	if !exists {
		return nil, false
	}

	if s, ok := skill.(*goreactskill.Skill); ok {
		return s, true
	}

	return nil, false
}

// GetSkillExecutionPlan retrieves a cached execution plan for a skill
func (rm *ResourceManager) GetSkillExecutionPlan(skillName string) (*goreactskill.SkillExecutionPlan, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	plan, exists := rm.planCache[skillName]
	return plan, exists
}

// SetSkillExecutionPlan caches an execution plan for a skill
func (rm *ResourceManager) SetSkillExecutionPlan(plan *goreactskill.SkillExecutionPlan) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if plan != nil && plan.SkillName != "" {
		rm.planCache[plan.SkillName] = plan
	}
}

// ClearPlanCache clears the skill execution plan cache
func (rm *ResourceManager) ClearPlanCache() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.planCache = make(map[string]*goreactskill.SkillExecutionPlan)
}
