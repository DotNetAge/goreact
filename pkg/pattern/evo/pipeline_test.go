package evo

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/memory"
	"github.com/ray/goreact/pkg/pattern/mastersub"
)

// MockLogger 简易日志记录器
type MockLogger struct{}

func (l *MockLogger) Info(msg string, args ...any)  {}
func (l *MockLogger) Warn(msg string, args ...any)  {}
func (l *MockLogger) Error(err error, msg string, args ...any) {}
func (l *MockLogger) Debug(msg string, args ...any) {}

// mockThinker 包装一个响应序列
type mockThinker struct {
	responses []string
	index     int
}

func (m *mockThinker) Think(ctx *core.PipelineContext) error {
	if m.index >= len(m.responses) {
		return fmt.Errorf("no more mock responses")
	}
	resp := m.responses[m.index]
	m.index++

	ctx.FinalResult = resp
	ctx.IsFinished = true
	return nil
}

// mockActor 模拟执行
type mockActor struct {
	weatherResponse string
}

func (a *mockActor) Act(ctx *core.PipelineContext) error {
	lastTrace := ctx.LastTrace()
	if lastTrace != nil && lastTrace.Action != nil && lastTrace.Action.Name == "weather" {
		lastTrace.Observation = &core.Observation{
			Data:      a.weatherResponse,
			IsSuccess: true,
		}
	}
	return nil
}

// mockObserver 模拟观察
type mockObserver struct{}

func (o *mockObserver) Observe(ctx *core.PipelineContext) error {
	// 已经在 mockActor 中模拟了填充 Observation.Data
	return nil
}

// mockSubReactor 简易实现用于测试
type mockSubReactor struct {
	actor *mockActor
}

func (s *mockSubReactor) Execute(ctx context.Context, task mastersub.Task) (mastersub.TaskResult, error) {
	pctx := core.NewPipelineContext(ctx, task.ID, "")
	pctx.AppendTrace(&core.Trace{
		Action: &core.Action{Name: "weather"},
	})
	s.actor.Act(pctx)
	
	lastTrace := pctx.LastTrace()
	return mastersub.TaskResult{
		TaskID:  task.ID,
		Success: true,
		Answer:  lastTrace.Observation.Data,
		Traces:  []core.Trace{*lastTrace},
	}, nil
}

func TestEvolutionPipeline_FullLoop(t *testing.T) {
	ctx := context.Background()
	logger := core.DefaultLogger()

	// 1. 准备 Mock 组件
	actor := &mockActor{weatherResponse: "The weather in Beijing is Sunny, 25C."}
	observer := &mockObserver{}
	
	thinker := &mockThinker{
		responses: []string{
			// 1. Master Plan
			`[{"id": "t1", "title": "Check Weather", "description": "Check weather in Beijing", "dependencies": [], "is_composite": false}]`,
			// 2. Compiler Compiled Graph
			`{"skill_name": "weather_skill", "steps": [{"id": "s1", "tool_name": "weather", "input_template": "{\"city\": \"Beijing\"}", "expected_observation": "Sunny", "description": "Fetch weather"}]}`,
			// 3. Escalation Answer (Detective)
			"The fingerprint was Sunny, but now it's Rainy. I have updated my understanding.",
		},
	}

	sub := &mockSubReactor{actor: actor}
	
	// 初始化
	master := mastersub.NewMaster(thinker)
	compiler := NewCompiler(thinker)
	memBank := memory.NewDefaultMemoryBank()

	pipeline := NewEvolutionPipeline(
		master,
		sub,
		compiler,
		memBank,
		actor,
		observer,
		thinker,
		logger,
	)

	// --- 场景 1: 冷启动 (Cold Start) ---
	t.Run("ColdStart_And_Compile", func(t *testing.T) {
		skillName := "weather_skill"
		input := map[string]any{"city": "Beijing"}

		result, err := pipeline.Execute(ctx, skillName, input)
		if err != nil {
			t.Fatalf("Cold start failed: %v", err)
		}

		if !strings.Contains(result, "Sunny") {
			t.Errorf("Expected result to contain Sunny, got: %s", result)
		}

		// 检查肌肉记忆是否已固化
		graph, err := memBank.Muscle().LoadCompiledAction(ctx, skillName)
		if err != nil || graph == nil {
			t.Fatal("Muscle memory should have a compiled graph after successful execution")
		}
	})

	// --- 场景 2: 快径执行 (Fast-Path) ---
	t.Run("FastPath_Execution", func(t *testing.T) {
		skillName := "weather_skill"
		input := map[string]any{"city": "Beijing"}

		initialIndex := thinker.index

		result, err := pipeline.Execute(ctx, skillName, input)
		if err != nil {
			t.Fatalf("Fast-path failed: %v", err)
		}

		if !strings.Contains(result, "successful") {
			t.Errorf("Expected successful fast-path message, got: %s", result)
		}

		if thinker.index != initialIndex {
			t.Errorf("Thinker was called during Fast-Path: index moved from %d to %d", initialIndex, thinker.index)
		}
	})

	// --- 场景 3: 升级逻辑 (Escalation) ---
	t.Run("Escalation_On_Mismatch", func(t *testing.T) {
		skillName := "weather_skill"
		input := map[string]any{"city": "Beijing"}

		// 模拟环境变化
		actor.weatherResponse = "The weather in Beijing is Rainy, 18C."

		result, err := pipeline.Execute(ctx, skillName, input)
		if err != nil {
			t.Fatalf("Escalation failed: %v", err)
		}

		if !strings.Contains(result, "Rainy") {
			t.Errorf("Expected detective's answer to handle the rainy situation, got: %s", result)
		}
	})
}
