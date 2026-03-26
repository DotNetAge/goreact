package actor

import (
	"context"
	"errors"
	"testing"

	"github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/goreact/pkg/tools"
)

type mockToolForActor struct {
	name    string
executeFunc func(ctx context.Context, input map[string]any) (any, error)
}

func (m *mockToolForActor) Name() string { return m.name }
func (m *mockToolForActor) Description() string { return "mock tool" }
func (m *mockToolForActor) SecurityLevel() tools.SecurityLevel { return tools.LevelSafe }
func (m *mockToolForActor) Execute(ctx context.Context, input map[string]any) (any, error) {
	return m.executeFunc(ctx, input)
}

func TestDefault_Options(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		actor := Default()
		if actor == nil {
			t.Fatal("Expected non-nil actor")
		}
	})

	t.Run("with tool manager", func(t *testing.T) {
		mgr := tools.NewSimpleManager()
		actor := Default(WithToolManager(mgr))
		if actor == nil {
			t.Fatal("Expected non-nil actor")
		}
	})
}

func TestDefaultActor_Act(t *testing.T) {
	t.Run("nil trace", func(t *testing.T) {
		actor := &defaultActor{}
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		err := actor.Act(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("nil action", func(t *testing.T) {
		actor := &defaultActor{}
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{})
		err := actor.Act(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("tool not found", func(t *testing.T) {
		mgr := tools.NewSimpleManager()
		actor := &defaultActor{toolManager: mgr}
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "nonexistent"}})
		err := actor.Act(ctx)
		if err != nil {
			t.Errorf("Expected no error (logs warning), got %v", err)
		}
	})

	t.Run("successful execution", func(t *testing.T) {
		mgr := tools.NewSimpleManager()
		mgr.Register(&mockToolForActor{
			name: "test-tool",
			executeFunc: func(ctx context.Context, input map[string]any) (any, error) {
				return "success", nil
			},
		})
		actor := &defaultActor{toolManager: mgr}
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "test-tool", Input: map[string]any{}}})
		err := actor.Act(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("tool execution error", func(t *testing.T) {
		mgr := tools.NewSimpleManager()
		expectedErr := errors.New("execution failed")
		mgr.Register(&mockToolForActor{
			name: "failing-tool",
			executeFunc: func(ctx context.Context, input map[string]any) (any, error) {
				return nil, expectedErr
			},
		})
		actor := &defaultActor{toolManager: mgr}
		ctx := core.NewPipelineContext(context.Background(), "test", "input")
		ctx.AppendTrace(&core.Trace{Action: &core.Action{Name: "failing-tool", Input: map[string]any{}}})
		err := actor.Act(ctx)
		if err != nil {
			t.Errorf("Expected no error (error stored in context), got %v", err)
		}
	})
}

func TestActor_Interface(t *testing.T) {
	var _ Actor = (*defaultActor)(nil)
}

type mockActor struct {
	actErr error
}

func (m *mockActor) Act(ctx *core.PipelineContext) error {
	return m.actErr
}

func TestMockActor(t *testing.T) {
	m := &mockActor{actErr: errors.New("test error")}
	ctx := core.NewPipelineContext(context.Background(), "test", "input")
	err := m.Act(ctx)
	if err == nil {
		t.Error("Expected error")
	}
}