package thinker_test

import (
	"errors"
	"testing"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/core/thinker/middlewares"
	"github.com/ray/goreact/pkg/types"
)

// mockThinker 用于测试的简单 Thinker
type mockThinker struct {
	shouldFail bool
}

func (m *mockThinker) Think(task string, ctx *core.Context) (*types.Thought, error) {
	if m.shouldFail {
		return nil, errors.New("mock error")
	}
	return &types.Thought{
		Reasoning:    "mock reasoning",
		ShouldFinish: true,
		FinalAnswer:  task,
		Metadata:     make(map[string]interface{}),
	}, nil
}

func TestMiddlewareThinker_Basic(t *testing.T) {
	base := &mockThinker{}
	mt := thinker.NewMiddlewareThinker(base)

	ctx := core.NewContext()
	thought, err := mt.Think("test", ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if thought.FinalAnswer != "test" {
		t.Errorf("Expected 'test', got '%s'", thought.FinalAnswer)
	}
}

func TestMiddlewareThinker_WithMiddleware(t *testing.T) {
	base := &mockThinker{}
	mt := thinker.NewMiddlewareThinker(base)

	// 添加一个简单的中间件
	called := false
	mt.Use(func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			called = true
			return next(task, ctx)
		}
	})

	ctx := core.NewContext()
	_, err := mt.Think("test", ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !called {
		t.Error("Expected middleware to be called")
	}
}

func TestMiddlewareThinker_MultipleMiddlewares(t *testing.T) {
	base := &mockThinker{}
	mt := thinker.NewMiddlewareThinker(base)

	order := []string{}

	// 第一个中间件
	mt.Use(func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			order = append(order, "middleware1_before")
			thought, err := next(task, ctx)
			order = append(order, "middleware1_after")
			return thought, err
		}
	})

	// 第二个中间件
	mt.Use(func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			order = append(order, "middleware2_before")
			thought, err := next(task, ctx)
			order = append(order, "middleware2_after")
			return thought, err
		}
	})

	ctx := core.NewContext()
	_, _ = mt.Think("test", ctx)

	// 验证执行顺序（洋葱模型）
	expected := []string{
		"middleware1_before",
		"middleware2_before",
		"middleware2_after",
		"middleware1_after",
	}

	if len(order) != len(expected) {
		t.Errorf("Expected %d steps, got %d", len(expected), len(order))
	}

	for i, step := range expected {
		if i >= len(order) || order[i] != step {
			t.Errorf("Step %d: expected '%s', got '%s'", i, step, order[i])
		}
	}
}

func TestIntentCacheMiddleware(t *testing.T) {
	base := &mockThinker{}
	mt := thinker.NewMiddlewareThinker(base)

	cache := middlewares.NewMemoryIntentCache()
	mt.Use(middlewares.IntentCacheMiddleware(cache, 0.5)) // 降低阈值

	ctx := core.NewContext()

	// 第一次调用
	thought1, _ := mt.Think("test task", ctx)

	// 第二次调用完全相同的任务
	thought2, _ := mt.Think("test task", ctx)

	// 应该从缓存返回
	if cached, ok := thought2.Metadata["cached"]; !ok || !cached.(bool) {
		t.Error("Expected cached result")
	}

	// 验证结果相同
	if thought1.FinalAnswer != thought2.FinalAnswer {
		t.Error("Cached result should be the same")
	}
}

func TestConfidenceMiddleware(t *testing.T) {
	base := &mockThinker{}
	mt := thinker.NewMiddlewareThinker(base)

	mt.Use(middlewares.ConfidenceMiddleware(0.7, nil))

	ctx := core.NewContext()
	thought, err := mt.Think("test", ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 验证置信度被添加
	if _, ok := thought.Metadata["confidence"]; !ok {
		t.Error("Expected confidence in metadata")
	}
}
