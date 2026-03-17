package mock

import (
	"context"
	"fmt"
	"sync"

	gochatcore "github.com/DotNetAge/gochat/pkg/core"
)

// MockClient 模拟 LLM 客户端
type MockClient struct {
	responses []string
	index     int
	mu        sync.Mutex
	CallCount int
}

// NewMockClient 创建模拟客户端
func NewMockClient(responses []string) *MockClient {
	return &MockClient{
		responses: responses,
	}
}

// Chat 实现 core.Client 接口
func (m *MockClient) Chat(ctx context.Context, messages []gochatcore.Message, opts ...gochatcore.Option) (*gochatcore.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CallCount++

	if len(m.responses) == 0 {
		return &gochatcore.Response{
			Content: "Mock response",
			Usage:   &gochatcore.Usage{TotalTokens: 10},
		}, nil
	}

	if m.index >= len(m.responses) {
		return nil, fmt.Errorf("no more mock responses available")
	}

	resp := m.responses[m.index]
	m.index++

	return &gochatcore.Response{
		Content: resp,
		Usage:   &gochatcore.Usage{TotalTokens: len(resp)},
	}, nil
}

// ChatStream 实现 core.Client 接口
func (m *MockClient) ChatStream(ctx context.Context, messages []gochatcore.Message, opts ...gochatcore.Option) (*gochatcore.Stream, error) {
	return nil, fmt.Errorf("ChatStream not implemented in mock")
}
