package agent

import (
	"testing"
	"time"

	"github.com/DotNetAge/goreact/pkg/common"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Name != common.DefaultAgentName {
		t.Errorf("Name = %q, want %q", config.Name, common.DefaultAgentName)
	}
	if config.Domain != common.DefaultDomain {
		t.Errorf("Domain = %q, want %q", config.Domain, common.DefaultDomain)
	}
	if config.Model != common.DefaultModel {
		t.Errorf("Model = %q, want %q", config.Model, common.DefaultModel)
	}
	if config.MaxSteps != common.DefaultMaxSteps {
		t.Errorf("MaxSteps = %d, want %d", config.MaxSteps, common.DefaultMaxSteps)
	}
	if config.MaxRetries != common.DefaultMaxRetries {
		t.Errorf("MaxRetries = %d, want %d", config.MaxRetries, common.DefaultMaxRetries)
	}
	if !config.EnableReflection {
		t.Error("EnableReflection should be true by default")
	}
	if !config.EnablePlanning {
		t.Error("EnablePlanning should be true by default")
	}
}

func TestNewBaseAgent(t *testing.T) {
	config := &Config{
		Name:        "test-agent",
		Domain:      "testing",
		Description: "Test agent",
		Model:       "gpt-4",
	}
	agent := NewBaseAgent(config)

	if agent.Name() != "test-agent" {
		t.Errorf("Name() = %q, want 'test-agent'", agent.Name())
	}
	if agent.Domain() != "testing" {
		t.Errorf("Domain() = %q, want 'testing'", agent.Domain())
	}
	if agent.Description() != "Test agent" {
		t.Errorf("Description() = %q, want 'Test agent'", agent.Description())
	}
	if agent.Model() != "gpt-4" {
		t.Errorf("Model() = %q, want 'gpt-4'", agent.Model())
	}
}

func TestNewBaseAgent_NilConfig(t *testing.T) {
	agent := NewBaseAgent(nil)

	if agent == nil {
		t.Fatal("NewBaseAgent(nil) returned nil")
	}
	// Should use default config
	if agent.Name() != common.DefaultAgentName {
		t.Errorf("Name() = %q, want %q", agent.Name(), common.DefaultAgentName)
	}
}

func TestBaseAgent_Skills(t *testing.T) {
	config := &Config{
		Name:   "test",
		Skills: []string{"code-review", "refactor"},
	}
	agent := NewBaseAgent(config)

	skills := agent.Skills()
	if len(skills) != 2 {
		t.Errorf("len(Skills()) = %d, want 2", len(skills))
	}
}

func TestBaseAgent_PromptTemplate(t *testing.T) {
	template := "You are a helpful assistant."
	config := &Config{
		Name:           "test",
		PromptTemplate: template,
	}
	agent := NewBaseAgent(config)

	if agent.PromptTemplate() != template {
		t.Errorf("PromptTemplate() = %q, want %q", agent.PromptTemplate(), template)
	}
}

func TestBaseAgent_Config(t *testing.T) {
	config := &Config{
		Name:    "test",
		MaxSteps: 20,
	}
	agent := NewBaseAgent(config)

	returnedConfig := agent.Config()
	if returnedConfig == nil {
		t.Fatal("Config() returned nil")
	}
	if returnedConfig.MaxSteps != 20 {
		t.Errorf("Config().MaxSteps = %d, want 20", returnedConfig.MaxSteps)
	}
}

func TestNewInput(t *testing.T) {
	input := NewInput("What is the weather?", "file1.txt", "file2.txt")

	if input.Question != "What is the weather?" {
		t.Errorf("Question = %q, want 'What is the weather?'", input.Question)
	}
	if len(input.Files) != 2 {
		t.Errorf("len(Files) = %d, want 2", len(input.Files))
	}
	if input.Context == nil {
		t.Error("Context should not be nil")
	}
}

func TestInput_WithContext(t *testing.T) {
	input := NewInput("test question")
	input.WithContext("key1", "value1")
	input.WithContext("key2", 42)

	if input.Context["key1"] != "value1" {
		t.Errorf("Context[key1] = %v, want 'value1'", input.Context["key1"])
	}
	if input.Context["key2"] != 42 {
		t.Errorf("Context[key2] = %v, want 42", input.Context["key2"])
	}
}

func TestResult(t *testing.T) {
	result := &Result{
		Answer:      "The answer is 42",
		Confidence:  0.95,
		Status:      common.StatusCompleted,
		SessionName: "session-123",
		TokenUsage: &common.TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		Duration: 2 * time.Second,
	}

	if result.Answer != "The answer is 42" {
		t.Errorf("Answer = %q, want 'The answer is 42'", result.Answer)
	}
	if result.Confidence != 0.95 {
		t.Errorf("Confidence = %f, want 0.95", result.Confidence)
	}
	if result.Status != common.StatusCompleted {
		t.Errorf("Status = %q, want 'completed'", result.Status)
	}
}

func TestResult_WithError(t *testing.T) {
	result := &Result{
		Status: common.StatusFailed,
		Error:  "Something went wrong",
	}

	if result.Status != common.StatusFailed {
		t.Errorf("Status = %q, want 'failed'", result.Status)
	}
	if result.Error != "Something went wrong" {
		t.Errorf("Error = %q, want 'Something went wrong'", result.Error)
	}
}

func TestResult_WithPendingQuestion(t *testing.T) {
	result := &Result{
		Status: common.StatusPaused,
		PendingQuestion: &PendingQuestion{
			ID:     "q-123",
			Type:   common.QuestionTypeConfirmation,
			Question: "Do you want to continue?",
			Options: []string{"Yes", "No"},
			DefaultAnswer: "Yes",
		},
	}

	if result.Status != common.StatusPaused {
		t.Errorf("Status = %q, want 'paused'", result.Status)
	}
	if result.PendingQuestion == nil {
		t.Fatal("PendingQuestion should not be nil")
	}
	if result.PendingQuestion.Type != common.QuestionTypeConfirmation {
		t.Errorf("PendingQuestion.Type = %q, want 'confirmation'", result.PendingQuestion.Type)
	}
}

func TestPendingQuestion(t *testing.T) {
	pq := &PendingQuestion{
		ID:            "q-001",
		Type:          common.QuestionTypeAuthorization,
		Question:      "Allow access to sensitive data?",
		Options:       []string{"Allow", "Deny"},
		DefaultAnswer: "Deny",
		Context:       map[string]any{"resource": "database"},
	}

	if pq.ID != "q-001" {
		t.Errorf("ID = %q, want 'q-001'", pq.ID)
	}
	if pq.Type != common.QuestionTypeAuthorization {
		t.Errorf("Type = %q, want 'authorization'", pq.Type)
	}
	if len(pq.Options) != 2 {
		t.Errorf("len(Options) = %d, want 2", len(pq.Options))
	}
	if pq.DefaultAnswer != "Deny" {
		t.Errorf("DefaultAnswer = %q, want 'Deny'", pq.DefaultAnswer)
	}
}

func TestConfig_Tools(t *testing.T) {
	config := &Config{
		Name:  "test",
		Tools: []string{"read_file", "bash", "write_file"},
	}

	if len(config.Tools) != 3 {
		t.Errorf("len(Tools) = %d, want 3", len(config.Tools))
	}
}

func TestConfig_Metadata(t *testing.T) {
	config := &Config{
		Name: "test",
		Metadata: map[string]any{
			"version": "1.0",
			"author":  "test",
		},
	}

	if config.Metadata["version"] != "1.0" {
		t.Errorf("Metadata[version] = %v, want '1.0'", config.Metadata["version"])
	}
	if config.Metadata["author"] != "test" {
		t.Errorf("Metadata[author] = %v, want 'test'", config.Metadata["author"])
	}
}
