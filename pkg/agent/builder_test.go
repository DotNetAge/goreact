package agent

import (
	"context"
	"strings"
	"testing"

	chatcore "github.com/DotNetAge/gochat/pkg/core"
	"github.com/DotNetAge/goreact/pkg/tools"
)

// mockLLMClient implements a simple deterministic LLM client for testing.
type mockLLMClient struct{}

func (m *mockLLMClient) Chat(ctx context.Context, messages []chatcore.Message, opts ...chatcore.Option) (*chatcore.Message, error) {
	return nil, nil // Not used in this test
}

// ChatStream mock always emits a final thinking trace that terminates the session
func (m *mockLLMClient) ChatStream(ctx context.Context, messages []chatcore.Message, opts ...chatcore.Option) (*chatcore.Stream, error) {
	lastMessage := ""
	if len(messages) > 0 {
		for _, block := range messages[len(messages)-1].Content {
			lastMessage += block.Text
		}
	}

	response := ""
	if strings.Contains(lastMessage, "execute tool") {
		// Mock LLM asking for a tool execution
		response = "Thought: I need to use the mock tool.\nAction: mock_tool\nActionInput: {\"value\": \"test\"}"
	} else if strings.Contains(lastMessage, "Observation") {
		// Mock LLM receiving tool result and finishing
		response = "Thought: I got the result.\nAction: finish\nActionInput: {\"final_answer\": \"The result is processed.\"}"
	} else {
		// Default to finish immediately
		response = "Thought: I know the answer directly.\nAction: finish\nActionInput: {\"final_answer\": \"Direct Answer\"}"
	}

	// Create a mock stream reader using the gochat channel
	ch := make(chan chatcore.StreamEvent, 1)
	ch <- chatcore.StreamEvent{
		Type:    chatcore.EventContent,
		Content: response,
	}
	close(ch)
	return chatcore.NewStream(ch, nil), nil
}

// mockTool implements a simple tool for testing.
type mockTool struct{}

func (t *mockTool) Name() string                       { return "mock_tool" }
func (t *mockTool) Description() string                { return "A mock tool for testing." }
func (t *mockTool) SecurityLevel() tools.SecurityLevel { return tools.LevelSafe }
func (t *mockTool) Execute(ctx context.Context, input map[string]any) (any, error) {
	return "mock tool executed successfully", nil
}

func TestAgentBuilderAndRunner(t *testing.T) {
	// 我们可以直接验证 Agent 的基础属性是否工作。
	agent := NewAgent("TestAgent", "A test agent.", "You are a test agent.", "mock-model")

	if agent.AgentName != "TestAgent" {
		t.Errorf("Expected runner name 'TestAgent', got '%s'", agent.AgentName)
	}
}
