package skill

import (
	"testing"
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

func TestNewSkill(t *testing.T) {
	skill := NewSkill("code-review", "Reviews code for best practices", "assistant")

	if skill.Name != "code-review" {
		t.Errorf("Name = %q, want 'code-review'", skill.Name)
	}
	if skill.Description != "Reviews code for best practices" {
		t.Errorf("Description = %q, want 'Reviews code for best practices'", skill.Description)
	}
	if skill.Agent != "assistant" {
		t.Errorf("Agent = %q, want 'assistant'", skill.Agent)
	}
	if skill.Parameters == nil {
		t.Error("Parameters should not be nil")
	}
	if skill.Steps == nil {
		t.Error("Steps should not be nil")
	}
	if skill.AllowedTools == nil {
		t.Error("AllowedTools should not be nil")
	}
	if skill.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
}

func TestSkill_WithIntent(t *testing.T) {
	skill := NewSkill("test", "test", "assistant")
	skill.WithIntent("review_code")

	if skill.Intent != "review_code" {
		t.Errorf("Intent = %q, want 'review_code'", skill.Intent)
	}
}

func TestSkill_WithTemplate(t *testing.T) {
	skill := NewSkill("test", "test", "assistant")
	template := "Analyze the code and provide feedback"
	skill.WithTemplate(template)

	if skill.Template != template {
		t.Errorf("Template = %q, want %q", skill.Template, template)
	}
}

func TestSkill_WithParameter(t *testing.T) {
	skill := NewSkill("test", "test", "assistant")
	skill.WithParameter(Parameter{
		Name:        "language",
		Type:        "string",
		Required:    true,
		Description: "Programming language",
	})

	if len(skill.Parameters) != 1 {
		t.Errorf("len(Parameters) = %d, want 1", len(skill.Parameters))
	}
	if skill.Parameters[0].Name != "language" {
		t.Errorf("Parameter name = %q, want 'language'", skill.Parameters[0].Name)
	}
}

func TestSkill_WithStep(t *testing.T) {
	skill := NewSkill("test", "test", "assistant")
	skill.WithStep(ExecutionStep{
		ToolName:    "read_file",
		Description: "Read the source file",
	})

	if len(skill.Steps) != 1 {
		t.Errorf("len(Steps) = %d, want 1", len(skill.Steps))
	}
	if skill.Steps[0].Index != 0 {
		t.Errorf("Step index = %d, want 0", skill.Steps[0].Index)
	}
	if skill.Steps[0].ToolName != "read_file" {
		t.Errorf("Tool name = %q, want 'read_file'", skill.Steps[0].ToolName)
	}
}

func TestSkill_WithAllowedTools(t *testing.T) {
	skill := NewSkill("test", "test", "assistant")
	tools := []string{"read_file", "write_file", "bash"}
	skill.WithAllowedTools(tools)

	if len(skill.AllowedTools) != 3 {
		t.Errorf("len(AllowedTools) = %d, want 3", len(skill.AllowedTools))
	}
}

func TestSkill_ComputeContentHash(t *testing.T) {
	skill := NewSkill("test", "test", "assistant")
	skill.WithStep(ExecutionStep{ToolName: "read"})
	skill.WithParameter(Parameter{Name: "path"})

	hash := skill.ComputeContentHash()
	if hash == "" {
		t.Error("ComputeContentHash() returned empty string")
	}
}

func TestNewSkillExecutionPlan(t *testing.T) {
	plan := NewSkillExecutionPlan("code-review")

	if plan.Name != "plan-code-review" {
		t.Errorf("Name = %q, want 'plan-code-review'", plan.Name)
	}
	if plan.SkillName != "code-review" {
		t.Errorf("SkillName = %q, want 'code-review'", plan.SkillName)
	}
	if plan.Steps == nil {
		t.Error("Steps should not be nil")
	}
	if plan.ExecutionCount != 0 {
		t.Errorf("ExecutionCount = %d, want 0", plan.ExecutionCount)
	}
}

func TestSkillExecutionPlan_IncrementExecution(t *testing.T) {
	plan := NewSkillExecutionPlan("test")

	// First execution - success
	plan.IncrementExecution(true)
	if plan.ExecutionCount != 1 {
		t.Errorf("ExecutionCount = %d, want 1", plan.ExecutionCount)
	}
	if plan.SuccessRate <= 0 {
		t.Errorf("SuccessRate = %f, should be positive after success", plan.SuccessRate)
	}

	// Second execution - failure
	plan.IncrementExecution(false)
	if plan.ExecutionCount != 2 {
		t.Errorf("ExecutionCount = %d, want 2", plan.ExecutionCount)
	}
}

func TestExecutionStep(t *testing.T) {
	step := ExecutionStep{
		Index:           0,
		ToolName:        "read_file",
		ParamsTemplate:  map[string]any{"path": "{{.Params.file}}"},
		Condition:       "params.file != ''",
		ExpectedOutcome: "File content loaded",
		Description:     "Read the specified file",
		OnError:         "stop",
		MaxRetries:      3,
		RetryDelay:      time.Second,
	}

	if step.Index != 0 {
		t.Errorf("Index = %d, want 0", step.Index)
	}
	if step.ToolName != "read_file" {
		t.Errorf("ToolName = %q, want 'read_file'", step.ToolName)
	}
	if step.OnError != "stop" {
		t.Errorf("OnError = %q, want 'stop'", step.OnError)
	}
	if step.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", step.MaxRetries)
	}
}

func TestGeneratedSkill(t *testing.T) {
	genSkill := &GeneratedSkill{
		Name:          "custom-analyzer",
		Description:   "Analyzes custom data formats",
		Content:       "# Custom Analyzer\n\nSteps...",
		FilePath:      "/skills/custom-analyzer/SKILL.md",
		SourceSession: "session-123",
		Status:        common.GeneratedStatusActive,
	}

	if genSkill.Name != "custom-analyzer" {
		t.Errorf("Name = %q, want 'custom-analyzer'", genSkill.Name)
	}
	if genSkill.Status != common.GeneratedStatusActive {
		t.Errorf("Status = %q, want 'active'", genSkill.Status)
	}
}

func TestSkillNode(t *testing.T) {
	node := &SkillNode{
		Name:         "code-review",
		NodeType:     "Skill",
		Description:  "Reviews code",
		Agent:        "assistant",
		Intent:       "review_code",
		Template:     "Analyze code...",
		AllowedTools: []string{"read_file", "bash"},
	}

	if node.Name != "code-review" {
		t.Errorf("Name = %q, want 'code-review'", node.Name)
	}
	if len(node.AllowedTools) != 2 {
		t.Errorf("len(AllowedTools) = %d, want 2", len(node.AllowedTools))
	}
}

func TestListOptions(t *testing.T) {
	opts := &ListOptions{}

	WithAgent("assistant")(opts)
	if opts.Agent != "assistant" {
		t.Errorf("Agent = %q, want 'assistant'", opts.Agent)
	}

	WithTags([]string{"code", "review"})(opts)
	if len(opts.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(opts.Tags))
	}

	WithLimit(10)(opts)
	if opts.Limit != 10 {
		t.Errorf("Limit = %d, want 10", opts.Limit)
	}

	WithOffset(5)(opts)
	if opts.Offset != 5 {
		t.Errorf("Offset = %d, want 5", opts.Offset)
	}
}

func TestTemplateContext(t *testing.T) {
	ctx := &TemplateContext{
		Session: &SessionState{
			Name:        "session-123",
			Input:       "Review this code",
			CurrentStep: 2,
			Context:     map[string]any{"key": "value"},
		},
		Params: map[string]any{"file": "main.go"},
		Runtime: &RuntimeContext{
			Timestamp:  time.Now(),
			WorkingDir: "/workspace",
			EnvVars:    map[string]string{"HOME": "/home/user"},
		},
	}

	if ctx.Session.Name != "session-123" {
		t.Errorf("Session.Name = %q, want 'session-123'", ctx.Session.Name)
	}
	if ctx.Params["file"] != "main.go" {
		t.Errorf("Params[file] = %v, want 'main.go'", ctx.Params["file"])
	}
	if ctx.Runtime.WorkingDir != "/workspace" {
		t.Errorf("Runtime.WorkingDir = %q, want '/workspace'", ctx.Runtime.WorkingDir)
	}
}

func TestStepResult(t *testing.T) {
	result := &StepResult{
		Index:    0,
		ToolName: "read_file",
		Success:  true,
		Result:   "file content here",
	}

	if result.Index != 0 {
		t.Errorf("Index = %d, want 0", result.Index)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestSkillParser(t *testing.T) {
	parser := NewSkillParser()

	if parser == nil {
		t.Fatal("NewSkillParser() returned nil")
	}

	// Test Parse method
	skill, err := parser.Parse("test content", "/skills/test/SKILL.md")
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}
	if skill == nil {
		t.Fatal("Parse() returned nil skill")
	}
	if skill.Path != "/skills/test/SKILL.md" {
		t.Errorf("Path = %q, want '/skills/test/SKILL.md'", skill.Path)
	}
}
