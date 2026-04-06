// Package resource provides resource management for the goreact framework.
package resource

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	goreactskill "github.com/DotNetAge/goreact/pkg/skill"
)

// ResourceManager manages all resources (agents, tools, skills, models)
type ResourceManager struct {
	mu      sync.RWMutex
	agents  map[string]any // Would be map[string]*agent.Agent
	tools   map[string]any // Would be map[string]*tool.Tool
	skills  map[string]any // Would be map[string]*skill.Skill
	models  map[string]any // Would be map[string]*llm.Model
	
	// Paths
	DocumentPath string
	SkillPath    string
	ToolPath     string
}

// NewResourceManager creates a new ResourceManager
func NewResourceManager() *ResourceManager {
	return &ResourceManager{
		agents: make(map[string]any),
		tools:  make(map[string]any),
		skills: make(map[string]any),
		models: make(map[string]any),
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
	
	// Would filter tools by agent
	return []string{}
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
	
	// Would filter skills by agent
	return []string{}
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
			content, err := ioutil.ReadFile(path)
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

	// Similar to ScanSkills - would scan for TOOL.md or tool definitions
	return nil
}

// ScanAgents scans a directory for agent definitions
func (rm *ResourceManager) ScanAgents(agentPath string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Similar to ScanSkills - would scan for AGENT.md definitions
	return nil
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

	// Would retrieve from plan cache
	return nil, false
}

// SetSkillExecutionPlan caches an execution plan for a skill
func (rm *ResourceManager) SetSkillExecutionPlan(plan *goreactskill.SkillExecutionPlan) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Would store in plan cache
}
