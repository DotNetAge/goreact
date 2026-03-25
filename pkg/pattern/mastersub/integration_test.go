package mastersub_test

import (
	"context"
	"testing"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/mock"
	"github.com/ray/goreact/pkg/pattern/mastersub"
	"github.com/ray/goreact/pkg/thinker"
)

func TestMasterSubOrchestration(t *testing.T) {
	ctx := context.Background()
	logger := core.DefaultLogger()

	// 1. 模拟 Master 的拆解响应 (JSON)
	masterResponse := `[
		{"id": "t1", "title": "分析代码", "description": "分析 main.go 的结构", "dependencies": [], "is_composite": false},
		{"id": "t2", "title": "修复 Bug", "description": "根据分析结果修复 main.go", "dependencies": ["t1"], "is_composite": false}
	]`
	
	// 2. 模拟 Sub 的执行响应 (ReAct 格式)
	subResponse1 := `Thought: 我需要读取 main.go。
Action: ls
ActionInput: {"path": "."}
Observation: main.go exists.
Final Answer: 分析完成，main.go 结构清晰。`

	subResponse2 := `Thought: 我现在开始修复。
Action: write
ActionInput: {"path": "main.go", "content": "// Fixed"}
Observation: Success.
Final Answer: Bug 已修复。`

	mockClient := mock.NewMockClient([]string{
		masterResponse, // Master Decompose 调用
		subResponse1,   // Sub 执行 t1
		subResponse2,   // Sub 执行 t2
	})

	// 3. 组装组件
	// 注意：这里为了简单，Master 和 Sub 共用一个默认 Thinker
	tk := thinker.Default(mockClient)
	master := mastersub.NewMaster(tk)
	
	// 创建一个真实的 Reactor 作为 Sub 的内核
	// 为了演示，我们不给 Sub 注入真实的 Actor/Observer，只验证流程
	// 但由于 Reactor.Run 需要这些组件，我们注入 Mock 或简单的实现
	// 这里由于我们只测试 MasterSub 逻辑，我们 mock SubReactor 接口
	
	sub := &mockSubReactor{responses: []string{"Result 1", "Result 2"}}
	
	orchestrator := mastersub.NewOrchestrator(master, sub, logger)

	// 4. 运行
	results, err := orchestrator.Run(ctx, "分析并修复 main.go")
	if err != nil {
		t.Fatalf("Orchestration failed: %v", err)
	}

	// 5. 验证
	if len(results) != 2 {
		t.Errorf("Expected 2 task results, got %d", len(results))
	}
	
	if results[0].TaskID != "t1" || results[1].TaskID != "t2" {
		t.Errorf("Task execution order incorrect")
	}

	logger.Info("Master-Sub Integration Test Passed!")
}

// 辅助 Mock
type mockSubReactor struct {
	responses []string
	index     int
}

func (m *mockSubReactor) Execute(ctx context.Context, task mastersub.Task) (mastersub.TaskResult, error) {
	res := mastersub.TaskResult{
		TaskID:  task.ID,
		Success: true,
		Answer:  m.responses[m.index],
		Traces:  []core.Trace{{Thought: "Mock Step"}},
	}
	m.index++
	return res, nil
}
