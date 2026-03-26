package thinker

import (
	"context"
	"strings"
	"testing"

	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/mock"
	"github.com/DotNetAge/goreact/pkg/tools"
)

func TestDefault_Options(t *testing.T) {
	t.Run("with model name", func(t *testing.T) {
		client := mock.NewMockClient([]string{})
		tk := Default(client, WithModel("gpt-4"))
		if tk == nil {
			t.Fatal("Expected non-nil thinker")
		}
	})

	t.Run("with tool manager", func(t *testing.T) {
		client := mock.NewMockClient([]string{})
		mgr := tools.NewSimpleManager()
		tk := Default(client, WithToolManager(mgr))
		if tk == nil {
			t.Fatal("Expected non-nil thinker")
		}
	})

	t.Run("with system prompt", func(t *testing.T) {
		client := mock.NewMockClient([]string{})
		tk := Default(client, WithSystemPrompt("custom prompt"))
		if tk == nil {
			t.Fatal("Expected non-nil thinker")
		}
	})
}

func TestDefaultThinker_Think(t *testing.T) {
	t.Run("successful think with action", func(t *testing.T) {
		response := `Thought: I need to calculate
Action: calculator
ActionInput: {"expr": "2+2"}`
		client := mock.NewMockClient([]string{response})
		tk := Default(client)
		ctx := core.NewPipelineContext(context.Background(), "test", "calculate 2+2")

		err := tk.Think(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(ctx.Traces) != 1 {
			t.Errorf("Expected 1 trace, got %d", len(ctx.Traces))
		}
	})

	t.Run("think with finish", func(t *testing.T) {
		response := `FinalAnswer: The result is 4`
		client := mock.NewMockClient([]string{response})
		tk := Default(client)
		ctx := core.NewPipelineContext(context.Background(), "test", "what is 2+2")

		err := tk.Think(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !ctx.IsFinished {
			t.Error("Expected IsFinished to be true")
		}
	})

	t.Run("/clear command", func(t *testing.T) {
		client := mock.NewMockClient([]string{})
		tk := Default(client)
		ctx := core.NewPipelineContext(context.Background(), "test", "/clear")
		ctx.AppendTrace(&core.Trace{Thought: "old thought"})

		err := tk.Think(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !ctx.IsFinished {
			t.Error("Expected IsFinished to be true")
		}
		if ctx.FinalResult != "Context cleared. I'm ready for a fresh start." {
			t.Errorf("Unexpected final result: %s", ctx.FinalResult)
		}
		if len(ctx.Traces) != 0 {
			t.Error("Expected traces to be cleared")
		}
	})

	t.Run("/plan command", func(t *testing.T) {
		response := `Step 1: Do first thing
Step 2: Do second thing`
		client := mock.NewMockClient([]string{response})
		tk := Default(client)
		ctx := core.NewPipelineContext(context.Background(), "test", "/plan create a report")

		err := tk.Think(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if ctx.IsFinished {
			t.Error("Expected IsFinished to be false for plan mode")
		}
		if len(ctx.PlanSteps) != 2 {
			t.Errorf("Expected 2 plan steps, got %d", len(ctx.PlanSteps))
		}
	})

	t.Run("/specs command", func(t *testing.T) {
		response := `Spec document content here`
		client := mock.NewMockClient([]string{response})
		tk := Default(client)
		ctx := core.NewPipelineContext(context.Background(), "test", "/specs build an app")

		err := tk.Think(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !ctx.IsFinished {
			t.Error("Expected IsFinished to be true for specs mode")
		}
		if ctx.FinalResult != response {
			t.Errorf("Expected final result to be the raw response")
		}
	})

	t.Run("/compress command", func(t *testing.T) {
		response := `Thought: continuing
Action: search
ActionInput: {"query": "test"}`
		client := mock.NewMockClient([]string{response})
		tk := Default(client)
		ctx := core.NewPipelineContext(context.Background(), "test", "/compress")

		err := tk.Think(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("/json command", func(t *testing.T) {
		response := `{"result": "data"}`
		client := mock.NewMockClient([]string{response})
		tk := Default(client)
		ctx := core.NewPipelineContext(context.Background(), "test", "/json get data")

		err := tk.Think(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !ctx.IsFinished {
			t.Error("Expected IsFinished to be true for json mode")
		}
	})

	t.Run("with tools available", func(t *testing.T) {
		response := `Thought: using tool
Action: bash
ActionInput: {"command": "ls"}`
		client := mock.NewMockClient([]string{response})
		tk := Default(client, WithToolManager(tools.NewSimpleManager()))
		ctx := core.NewPipelineContext(context.Background(), "test", "list files")

		err := tk.Think(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("llm error", func(t *testing.T) {
		client := mock.NewMockClient([]string{})
		tk := Default(client)
		ctx := core.NewPipelineContext(context.Background(), "test", "test")

		err := tk.Think(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("format error reflection", func(t *testing.T) {
		response := `This is not properly formatted`
		client := mock.NewMockClient([]string{response})
		tk := Default(client)
		ctx := core.NewPipelineContext(context.Background(), "test", "test")

		err := tk.Think(ctx)
		if err != nil {
			t.Errorf("Expected no error (format error is handled internally), got %v", err)
		}

		if len(ctx.Traces) > 0 {
			lastTrace := ctx.Traces[len(ctx.Traces)-1]
			if lastTrace.Observation != nil && !lastTrace.Observation.IsSuccess {
				if !strings.Contains(lastTrace.Observation.Data, "Format mismatch") {
					t.Error("Expected format error reflection")
				}
			}
		}
	})
}

func TestDefaultThinker_resolveMode(t *testing.T) {
	tk := &defaultThinker{sysTemplate: "default template"}

	tests := []struct {
		input          string
		expectedMode   string
		expectedTplKey string
	}{
		{"/plan hello", "plan", ""},
		{"/specs hello", "specs", ""},
		{"/json hello", "json", "default template"},
		{"regular input", "react", "default template"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, _ := tk.resolveMode(tt.input)
			if mode != tt.expectedMode {
				t.Errorf("Expected mode %q, got %q", tt.expectedMode, mode)
			}
		})
	}
}

func TestDefaultThinker_createPromptBuilder(t *testing.T) {
	tk := &defaultThinker{
		sysTemplate: "System: {{.task}}",
	}
	ctx := &core.PipelineContext{
		Input:     "test input",
		SessionID: "session1",
		Traces:    []*core.Trace{},
	}
	var toolList []tools.Tool

	pb := tk.createPromptBuilder(ctx, toolList, "template")
	if pb == nil {
		t.Fatal("Expected non-nil prompt builder")
	}
}

func TestDefaultThinker_createPromptBuilder_withHistory(t *testing.T) {
	tk := &defaultThinker{
		sysTemplate: "System: {{.task}}",
	}
	ctx := &core.PipelineContext{
		Input:     "test input",
		SessionID: "session1",
		Traces: []*core.Trace{
			{
				Thought:    "first thought",
				Action:    &core.Action{Name: "tool1", Input: map[string]any{"key": "value"}},
				Observation: &core.Observation{Data: "result", IsSuccess: true},
			},
		},
	}
	var toolList []tools.Tool

	pb := tk.createPromptBuilder(ctx, toolList, "template")
	if pb == nil {
		t.Fatal("Expected non-nil prompt builder")
	}
}

func TestDefaultThinker_processOutput(t *testing.T) {
	tk := &defaultThinker{}

	t.Run("plan mode with valid tasks", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "/plan do something")
		err := tk.processOutput(ctx, "plan", "Step 1: First task\nStep 2: Second task")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if ctx.IsFinished {
			t.Error("Expected IsFinished to be false for plan")
		}
		if len(ctx.PlanSteps) != 2 {
			t.Errorf("Expected 2 plan steps, got %d", len(ctx.PlanSteps))
		}
	})

	t.Run("plan mode with parsing failure falls back", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "/plan do something")
		err := tk.processOutput(ctx, "plan", "not a valid plan")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !ctx.IsFinished {
			t.Error("Expected IsFinished to be true when plan parsing fails")
		}
	})

	t.Run("json mode", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "/json data")
		err := tk.processOutput(ctx, "json", `{"key": "value"}`)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !ctx.IsFinished {
			t.Error("Expected IsFinished to be true for json mode")
		}
		if ctx.FinishReason != "DirectOutput" {
			t.Errorf("Expected FinishReason 'DirectOutput', got %q", ctx.FinishReason)
		}
	})

	t.Run("specs mode", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "/specs build")
		err := tk.processOutput(ctx, "specs", "# Spec Document")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !ctx.IsFinished {
			t.Error("Expected IsFinished to be true for specs mode")
		}
	})

	t.Run("react mode with action", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "do task")
		ctx.CurrentStep = 1
		err := tk.processOutput(ctx, "react", `Thought: thinking
Action: tool
ActionInput: {"input": "value"}`)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(ctx.Traces) != 1 {
			t.Errorf("Expected 1 trace, got %d", len(ctx.Traces))
		}
	})

	t.Run("react mode with final answer", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "answer question")
		ctx.CurrentStep = 1
		err := tk.processOutput(ctx, "react", `Thought: I know
FinalAnswer: It is 42`)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !ctx.IsFinished {
			t.Error("Expected IsFinished to be true")
		}
		if ctx.FinalResult != "It is 42" {
			t.Errorf("Expected final result 'It is 42', got %q", ctx.FinalResult)
		}
		if ctx.FinishReason != "TaskComplete" {
			t.Errorf("Expected FinishReason 'TaskComplete', got %q", ctx.FinishReason)
		}
	})

	t.Run("react mode with parse error triggers reflection", func(t *testing.T) {
		ctx := core.NewPipelineContext(context.Background(), "test", "test")
		ctx.CurrentStep = 1
		err := tk.processOutput(ctx, "react", `invalid format`)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(ctx.Traces) > 0 {
			lastTrace := ctx.Traces[len(ctx.Traces)-1]
			if lastTrace.Observation == nil || lastTrace.Observation.IsSuccess {
				t.Error("Expected failed observation for format error")
			}
		}
	})
}

func TestThinker_Interface(t *testing.T) {
	var _ Thinker = (*defaultThinker)(nil)
}