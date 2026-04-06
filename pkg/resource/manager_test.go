package resource

import (
	"testing"
	"time"

	goreactskill "github.com/DotNetAge/goreact/pkg/skill"
)

func TestNewResourceManager(t *testing.T) {
	rm := NewResourceManager()

	if rm == nil {
		t.Fatal("NewResourceManager() returned nil")
	}
	if rm.agents == nil {
		t.Error("agents should not be nil")
	}
	if rm.tools == nil {
		t.Error("tools should not be nil")
	}
	if rm.skills == nil {
		t.Error("skills should not be nil")
	}
	if rm.models == nil {
		t.Error("models should not be nil")
	}
}

func TestResourceManager_RegisterAgent(t *testing.T) {
	rm := NewResourceManager()

	err := rm.RegisterAgent("assistant", map[string]any{"name": "assistant"})
	if err != nil {
		t.Errorf("RegisterAgent() error = %v", err)
	}

	// Test duplicate registration
	err = rm.RegisterAgent("assistant", map[string]any{"name": "assistant2"})
	if err == nil {
		t.Error("RegisterAgent() should return error for duplicate")
	}
}

func TestResourceManager_UnregisterAgent(t *testing.T) {
	rm := NewResourceManager()
	rm.RegisterAgent("assistant", map[string]any{})

	rm.UnregisterAgent("assistant")

	_, exists := rm.GetAgent("assistant")
	if exists {
		t.Error("Agent should not exist after unregister")
	}
}

func TestResourceManager_GetAgent(t *testing.T) {
	rm := NewResourceManager()
	rm.RegisterAgent("assistant", map[string]any{"model": "gpt-4"})

	agent, exists := rm.GetAgent("assistant")
	if !exists {
		t.Fatal("Agent should exist")
	}
	if agent.(map[string]any)["model"] != "gpt-4" {
		t.Errorf("Agent model = %v, want 'gpt-4'", agent.(map[string]any)["model"])
	}

	_, exists = rm.GetAgent("nonexistent")
	if exists {
		t.Error("Nonexistent agent should not exist")
	}
}

func TestResourceManager_RegisterTool(t *testing.T) {
	rm := NewResourceManager()

	err := rm.RegisterTool("read_file", map[string]any{"name": "read_file"})
	if err != nil {
		t.Errorf("RegisterTool() error = %v", err)
	}

	// Test duplicate
	err = rm.RegisterTool("read_file", map[string]any{})
	if err == nil {
		t.Error("RegisterTool() should return error for duplicate")
	}
}

func TestResourceManager_GetTool(t *testing.T) {
	rm := NewResourceManager()
	rm.RegisterTool("bash", map[string]any{"type": "cli"})

	tool, exists := rm.GetTool("bash")
	if !exists {
		t.Fatal("Tool should exist")
	}
	if tool.(map[string]any)["type"] != "cli" {
		t.Errorf("Tool type = %v, want 'cli'", tool.(map[string]any)["type"])
	}
}

func TestResourceManager_RegisterSkill(t *testing.T) {
	rm := NewResourceManager()

	skill := goreactskill.NewSkill("code-review", "Reviews code", "assistant")
	err := rm.RegisterSkill("code-review", skill)
	if err != nil {
		t.Errorf("RegisterSkill() error = %v", err)
	}
}

func TestResourceManager_GetSkill(t *testing.T) {
	rm := NewResourceManager()
	skill := goreactskill.NewSkill("test-skill", "Test skill", "assistant")
	rm.RegisterSkill("test-skill", skill)

	s, exists := rm.GetSkill("test-skill")
	if !exists {
		t.Fatal("Skill should exist")
	}
	if s.(*goreactskill.Skill).Name != "test-skill" {
		t.Errorf("Skill name = %q, want 'test-skill'", s.(*goreactskill.Skill).Name)
	}
}

func TestResourceManager_RegisterModel(t *testing.T) {
	rm := NewResourceManager()

	model := &Model{
		Name:              "gpt-4",
		Provider:          "openai",
		ProviderModelName: "gpt-4-turbo",
		Temperature:       0.7,
		MaxTokens:         4096,
	}

	err := rm.RegisterModel("gpt-4", model)
	if err != nil {
		t.Errorf("RegisterModel() error = %v", err)
	}
}

func TestResourceManager_GetModel(t *testing.T) {
	rm := NewResourceManager()
	model := &Model{Name: "gpt-4", Provider: "openai"}
	rm.RegisterModel("gpt-4", model)

	m, exists := rm.GetModel("gpt-4")
	if !exists {
		t.Fatal("Model should exist")
	}
	if m.(*Model).Provider != "openai" {
		t.Errorf("Provider = %q, want 'openai'", m.(*Model).Provider)
	}
}

func TestResourceManager_RegisterAgentTools(t *testing.T) {
	rm := NewResourceManager()
	rm.RegisterAgent("assistant", map[string]any{})
	rm.RegisterTool("read_file", map[string]any{})
	rm.RegisterTool("bash", map[string]any{})

	err := rm.RegisterAgentTools("assistant", []string{"read_file", "bash"})
	if err != nil {
		t.Errorf("RegisterAgentTools() error = %v", err)
	}

	tools := rm.GetToolsByAgent("assistant")
	if len(tools) != 2 {
		t.Errorf("len(tools) = %d, want 2", len(tools))
	}
}

func TestResourceManager_RegisterAgentTools_AgentNotFound(t *testing.T) {
	rm := NewResourceManager()
	rm.RegisterTool("read_file", map[string]any{})

	err := rm.RegisterAgentTools("nonexistent", []string{"read_file"})
	if err == nil {
		t.Error("Should return error for nonexistent agent")
	}
}

func TestResourceManager_RegisterAgentTools_ToolNotFound(t *testing.T) {
	rm := NewResourceManager()
	rm.RegisterAgent("assistant", map[string]any{})

	err := rm.RegisterAgentTools("assistant", []string{"nonexistent"})
	if err == nil {
		t.Error("Should return error for nonexistent tool")
	}
}

func TestResourceManager_RegisterAgentSkills(t *testing.T) {
	rm := NewResourceManager()
	rm.RegisterAgent("assistant", map[string]any{})
	rm.RegisterSkill("code-review", goreactskill.NewSkill("code-review", "test", "assistant"))

	err := rm.RegisterAgentSkills("assistant", []string{"code-review"})
	if err != nil {
		t.Errorf("RegisterAgentSkills() error = %v", err)
	}

	skills := rm.GetSkillsByAgent("assistant")
	if len(skills) != 1 {
		t.Errorf("len(skills) = %d, want 1", len(skills))
	}
}

func TestResourceManager_GetAllResources(t *testing.T) {
	rm := NewResourceManager()
	rm.RegisterAgent("assistant", map[string]any{})
	rm.RegisterTool("bash", map[string]any{})
	rm.RegisterSkill("test", goreactskill.NewSkill("test", "test", "assistant"))
	rm.RegisterModel("gpt-4", &Model{})

	resources := rm.GetAllResources()

	if len(resources) != 4 {
		t.Errorf("len(resources) = %d, want 4", len(resources))
	}
	if len(resources["agents"]) != 1 {
		t.Errorf("len(agents) = %d, want 1", len(resources["agents"]))
	}
}

func TestResourceManager_Clear(t *testing.T) {
	rm := NewResourceManager()
	rm.RegisterAgent("assistant", map[string]any{})
	rm.RegisterTool("bash", map[string]any{})

	rm.Clear()

	if len(rm.agents) != 0 {
		t.Errorf("len(agents) = %d, want 0", len(rm.agents))
	}
	if len(rm.tools) != 0 {
		t.Errorf("len(tools) = %d, want 0", len(rm.tools))
	}
}

func TestResourceManager_SetPaths(t *testing.T) {
	rm := NewResourceManager()

	rm.SetDocumentPath("/docs")
	rm.SetSkillPath("/skills")
	rm.SetToolPath("/tools")

	if rm.DocumentPath != "/docs" {
		t.Errorf("DocumentPath = %q, want '/docs'", rm.DocumentPath)
	}
	if rm.SkillPath != "/skills" {
		t.Errorf("SkillPath = %q, want '/skills'", rm.SkillPath)
	}
	if rm.ToolPath != "/tools" {
		t.Errorf("ToolPath = %q, want '/tools'", rm.ToolPath)
	}
}

func TestResourceManager_SkillExecutionPlan(t *testing.T) {
	rm := NewResourceManager()

	plan := goreactskill.NewSkillExecutionPlan("test-skill")
	rm.SetSkillExecutionPlan(plan)

	cached, exists := rm.GetSkillExecutionPlan("test-skill")
	if !exists {
		t.Fatal("Execution plan should exist")
	}
	if cached.SkillName != "test-skill" {
		t.Errorf("SkillName = %q, want 'test-skill'", cached.SkillName)
	}

	rm.ClearPlanCache()
	_, exists = rm.GetSkillExecutionPlan("test-skill")
	if exists {
		t.Error("Plan should not exist after cache clear")
	}
}

func TestResourceManager_GetSkillTyped(t *testing.T) {
	rm := NewResourceManager()
	skill := goreactskill.NewSkill("typed-skill", "Test", "assistant")
	rm.RegisterSkill("typed-skill", skill)

	s, exists := rm.GetSkillTyped("typed-skill")
	if !exists {
		t.Fatal("Skill should exist")
	}
	if s.Name != "typed-skill" {
		t.Errorf("Name = %q, want 'typed-skill'", s.Name)
	}

	_, exists = rm.GetSkillTyped("nonexistent")
	if exists {
		t.Error("Nonexistent skill should not exist")
	}
}

func TestModel(t *testing.T) {
	model := &Model{
		Name:              "gpt-4",
		Provider:          "openai",
		ProviderModelName: "gpt-4-turbo-preview",
		BaseURL:           "https://api.openai.com/v1",
		Temperature:       0.7,
		MaxTokens:         4096,
		Timeout:           30 * time.Second,
		Features: ModelFeatures{
			Vision:    true,
			ToolCall:  true,
			Streaming: true,
		},
	}

	if model.Name != "gpt-4" {
		t.Errorf("Name = %q, want 'gpt-4'", model.Name)
	}
	if !model.Features.Vision {
		t.Error("Features.Vision should be true")
	}
	if !model.Features.ToolCall {
		t.Error("Features.ToolCall should be true")
	}
}

func TestModelFeatures(t *testing.T) {
	features := ModelFeatures{
		Vision:    true,
		ToolCall:  true,
		Streaming: false,
	}

	if !features.Vision {
		t.Error("Vision should be true")
	}
	if !features.ToolCall {
		t.Error("ToolCall should be true")
	}
	if features.Streaming {
		t.Error("Streaming should be false")
	}
}

func TestGlobalFunctions(t *testing.T) {
	// Test global registration
	err := RegisterAgent("global-agent", map[string]any{"test": true})
	if err != nil {
		t.Errorf("RegisterAgent() error = %v", err)
	}

	agent, exists := GetAgent("global-agent")
	if !exists {
		t.Fatal("Global agent should exist")
	}
	if agent.(map[string]any)["test"] != true {
		t.Error("Global agent data incorrect")
	}

	// Test global tools
	RegisterTool("global-tool", map[string]any{})
	_, exists = GetTool("global-tool")
	if !exists {
		t.Error("Global tool should exist")
	}

	// Test global skills
	RegisterSkill("global-skill", goreactskill.NewSkill("global-skill", "test", "assistant"))
	_, exists = GetSkill("global-skill")
	if !exists {
		t.Error("Global skill should exist")
	}

	// Test global models
	RegisterModel("global-model", &Model{Name: "global-model"})
	_, exists = GetModel("global-model")
	if !exists {
		t.Error("Global model should exist")
	}
}

func TestResourceManager_Load(t *testing.T) {
	rm := NewResourceManager()
	rm.RegisterAgent("test", map[string]any{})

	err := rm.Load()
	if err != nil {
		t.Errorf("Load() error = %v", err)
	}
}
