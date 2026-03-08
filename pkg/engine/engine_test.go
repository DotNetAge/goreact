package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/ray/goreact/pkg/cache"
	"github.com/ray/goreact/pkg/tool/builtin"
)

// MockLLMWithError 模拟一个会返回错误的LLM客户端
type MockLLMWithError struct {
	ErrorCount int
	MaxErrors  int
}

func (m *MockLLMWithError) Generate(prompt string) (string, error) {
	if m.ErrorCount < m.MaxErrors {
		m.ErrorCount++
		return "", errors.New("LLM unavailable")
	}
	return "Thought: This is a test\nFinal Answer: Test response", nil
}

func TestEngine_ErrorRetry(t *testing.T) {
	// 创建一个会失败3次的LLM客户端
	llmClient := &MockLLMWithError{
		MaxErrors: 3,
	}

	// 创建引擎，设置重试次数为3
	eng := New(
		WithLLMClient(llmClient),
		WithMaxRetries(3),
		WithRetryInterval(10*time.Millisecond),
	)

	// 注册工具
	eng.RegisterTools(
		builtin.NewCalculator(),
		builtin.NewEcho(),
	)

	// 执行任务
	result := eng.Execute("Test task", nil)

	// 验证任务成功
	if !result.Success {
		t.Errorf("Expected success, got error: %v", result.Error)
	}

	// 验证LLM被调用了4次（3次失败 + 1次成功）
	if llmClient.ErrorCount != 3 {
		t.Errorf("Expected 3 errors, got %d", llmClient.ErrorCount)
	}
}

func TestEngine_GracefulDegradation(t *testing.T) {
	// 创建一个总是失败的LLM客户端
	llmClient := &MockLLMWithError{
		MaxErrors: 10, // 超过最大重试次数
	}

	// 创建引擎，设置重试次数为2
	eng := New(
		WithLLMClient(llmClient),
		WithMaxRetries(2),
		WithRetryInterval(10*time.Millisecond),
	)

	// 注册工具
	eng.RegisterTools(
		builtin.NewCalculator(),
		builtin.NewEcho(),
		builtin.NewDateTime(),
	)

	// 执行计算任务
	result := eng.Execute("Calculate 10 + 5", nil)

	// 验证任务成功
	if !result.Success {
		t.Errorf("Expected success with graceful degradation, got error: %v", result.Error)
	}
}

func TestEngine_CacheErrorRecovery(t *testing.T) {
	// 创建缓存
	memCache := cache.NewMemoryCache(
		cache.WithMaxSize(100),
		cache.WithDefaultTTL(1*time.Hour),
	)

	// 先执行一次任务，缓存结果
	eng1 := New(
		WithCache(memCache),
	)

	eng1.RegisterTools(
		builtin.NewEcho(),
	)

	// 第一次执行
	result1 := eng1.Execute("Test cached task", nil)
	if !result1.Success {
		t.Errorf("Expected success for first execution, got error: %v", result1.Error)
	}

	// 创建一个总是失败的LLM客户端
	llmClient := &MockLLMWithError{
		MaxErrors: 10,
	}

	// 创建新引擎，使用失败的LLM客户端但共享缓存
	eng2 := New(
		WithLLMClient(llmClient),
		WithCache(memCache),
		WithMaxRetries(2),
	)

	eng2.RegisterTools(
		builtin.NewEcho(),
	)

	// 第二次执行，应该从缓存获取结果
	result2 := eng2.Execute("Test cached task", nil)
	if !result2.Success {
		t.Errorf("Expected success from cache, got error: %v", result2.Error)
	}

	// 验证结果来自缓存
	if cached, ok := result2.Metadata["cached"]; !ok || cached != true {
		t.Error("Expected result to be from cache")
	}
}
