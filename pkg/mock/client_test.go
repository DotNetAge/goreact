package mock

import (
	"context"
	"sync"
	"testing"

	gochatcore "github.com/DotNetAge/gochat/pkg/core"
)

func TestNewMockClient(t *testing.T) {
	client := NewMockClient([]string{"response1", "response2"})
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if len(client.responses) != 2 {
		t.Errorf("Expected 2 responses, got %d", len(client.responses))
	}
}

func TestMockClient_Chat(t *testing.T) {
	t.Run("with responses", func(t *testing.T) {
		client := NewMockClient([]string{"response1", "response2"})
		messages := []gochatcore.Message{gochatcore.NewUserMessage("hello")}

		resp, err := client.Chat(context.Background(), messages)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if resp.Content != "response1" {
			t.Errorf("Expected 'response1', got %q", resp.Content)
		}
		if client.CallCount != 1 {
			t.Errorf("Expected call count 1, got %d", client.CallCount)
		}
	})

	t.Run("second response", func(t *testing.T) {
		client := NewMockClient([]string{"response1", "response2"})
		messages := []gochatcore.Message{gochatcore.NewUserMessage("hello")}

		client.Chat(context.Background(), messages)
		resp, _ := client.Chat(context.Background(), messages)
		if resp.Content != "response2" {
			t.Errorf("Expected 'response2', got %q", resp.Content)
		}
	})

	t.Run("no more responses", func(t *testing.T) {
		client := NewMockClient([]string{"response1"})
		messages := []gochatcore.Message{gochatcore.NewUserMessage("hello")}

		client.Chat(context.Background(), messages)
		_, err := client.Chat(context.Background(), messages)
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("empty responses", func(t *testing.T) {
		client := NewMockClient([]string{})
		messages := []gochatcore.Message{gochatcore.NewUserMessage("hello")}

		resp, err := client.Chat(context.Background(), messages)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if resp.Content != "Mock response" {
			t.Errorf("Expected 'Mock response', got %q", resp.Content)
		}
	})

	t.Run("thread safety", func(t *testing.T) {
		client := NewMockClient([]string{"response1", "response2", "response3"})
		messages := []gochatcore.Message{gochatcore.NewUserMessage("hello")}

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				client.Chat(context.Background(), messages)
			}()
		}
		wg.Wait()

		if client.CallCount != 10 {
			t.Errorf("Expected 10 calls, got %d", client.CallCount)
		}
	})
}

func TestMockClient_ChatStream(t *testing.T) {
	t.Run("successful stream", func(t *testing.T) {
		client := NewMockClient([]string{"streamed response"})
		messages := []gochatcore.Message{gochatcore.NewUserMessage("hello")}

		stream, err := client.ChatStream(context.Background(), messages)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if stream == nil {
			t.Error("Expected non-nil stream")
		}
	})

	t.Run("stream error", func(t *testing.T) {
		client := NewMockClient([]string{})
		messages := []gochatcore.Message{gochatcore.NewUserMessage("hello")}

		_, err := client.ChatStream(context.Background(), messages)
		if err != nil {
			t.Errorf("Expected no error from Chat (stream uses Chat internally), got %v", err)
		}
	})
}

func TestMockClient_ImplementsInterface(t *testing.T) {
	var _ gochatcore.Client = (*MockClient)(nil)
}

func TestMockClient_EmptyResponses(t *testing.T) {
	client := NewMockClient(nil)
	messages := []gochatcore.Message{gochatcore.NewUserMessage("hello")}

	resp, err := client.Chat(context.Background(), messages)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if resp.Content != "Mock response" {
		t.Errorf("Expected 'Mock response', got %q", resp.Content)
	}
}

func TestMockClient_Usage(t *testing.T) {
	client := NewMockClient([]string{"response"})
	messages := []gochatcore.Message{gochatcore.NewUserMessage("hello")}

	resp, _ := client.Chat(context.Background(), messages)
	if resp.Usage == nil {
		t.Error("Expected usage to be set")
	}
	if resp.Usage.TotalTokens != len("response") {
		t.Errorf("Expected %d tokens, got %d", len("response"), resp.Usage.TotalTokens)
	}
}

type errorClient struct {
	err error
}

func (e *errorClient) Chat(ctx context.Context, messages []gochatcore.Message, opts ...gochatcore.Option) (*gochatcore.Response, error) {
	return nil, e.err
}

func (e *errorClient) ChatStream(ctx context.Context, messages []gochatcore.Message, opts ...gochatcore.Option) (*gochatcore.Stream, error) {
	return nil, e.err
}