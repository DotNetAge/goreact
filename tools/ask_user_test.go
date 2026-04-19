package tools

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/DotNetAge/goreact/core"
)

func TestAskUser_Info(t *testing.T) {
	tool := NewAskUserTool()
	info := tool.Info()

	if info.Name != "ask_user" {
		t.Errorf("expected tool name 'ask_user', got %q", info.Name)
	}
	if !info.IsReadOnly {
		t.Error("expected IsReadOnly to be true")
	}
	if len(info.Parameters) == 0 {
		t.Error("expected parameters to be defined")
	}

	var hasQuestion bool
	for _, p := range info.Parameters {
		if p.Name == "question" && p.Required {
			hasQuestion = true
		}
	}
	if !hasQuestion {
		t.Error("expected 'question' parameter to be required")
	}
}

func TestAskUser_ExecuteBlocking(t *testing.T) {
	tool := NewAskUserTool().(*AskUser)

	var wg sync.WaitGroup
	var result string
	var execErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := tool.Execute(context.Background(), map[string]any{
			"question": "What is your name?",
		})
		execErr = err
		if res != nil {
			result = res.(string)
		}
	}()

	time.Sleep(50 * time.Millisecond)

	if !tool.IsWaiting() {
		t.Error("expected tool to be waiting for input")
	}

	err := tool.Respond("Alice")
	if err != nil {
		t.Fatalf("Respond failed: %v", err)
	}

	wg.Wait()

	if execErr != nil {
		t.Fatalf("Execute returned error: %v", execErr)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestAskUser_RespondWithoutBlocking(t *testing.T) {
	tool := NewAskUserTool().(*AskUser)

	err := tool.Respond("answer")
	if err == nil {
		t.Error("expected error when responding to a non-blocking tool")
	}
}

func TestAskUser_CancelViaContext(t *testing.T) {
	tool := NewAskUserTool().(*AskUser)

	ctx, cancel := context.WithCancel(context.Background())

	var execErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, execErr = tool.Execute(ctx, map[string]any{
			"question": "test?",
		})
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	<-done

	if execErr == nil {
		t.Error("expected error from context cancellation")
	}
	if tool.IsWaiting() {
		t.Error("tool should not be waiting after cancellation")
	}
}

func TestAskUser_RespondError(t *testing.T) {
	tool := NewAskUserTool().(*AskUser)

	var execErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, execErr = tool.Execute(context.Background(), map[string]any{
			"question": "test?",
		})
	}()

	time.Sleep(50 * time.Millisecond)

	_ = tool.RespondError(context.DeadlineExceeded)

	<-done

	if execErr == nil {
		t.Error("expected error from RespondError")
	}
}

func TestAskUser_WaitWithTimeout(t *testing.T) {
	tool := NewAskUserTool().(*AskUser)

	err := tool.WaitWithTimeout(1 * time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		tool.Execute(context.Background(), map[string]any{"question": "test?"})
	}()

	time.Sleep(50 * time.Millisecond)

	err = tool.WaitWithTimeout(50 * time.Millisecond)
	if err == nil {
		t.Error("expected timeout error")
	}

	tool.Respond("done")

	err = tool.WaitWithTimeout(2 * time.Second)
	if err != nil {
		t.Fatalf("unexpected error after respond: %v", err)
	}

	<-done
}

func TestAskUser_MissingParam(t *testing.T) {
	tool := NewAskUserTool()

	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing question parameter")
	}
}

func TestAskUser_WithEventEmitter(t *testing.T) {
	tool := NewAskUserTool().(*AskUser)

	var receivedEvent core.ReactEvent
	var mu sync.Mutex
	tool.SetEventEmitter(func(e core.ReactEvent) {
		mu.Lock()
		receivedEvent = e
		mu.Unlock()
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		tool.Execute(context.Background(), map[string]any{
			"question": "What color is the sky?",
		})
	}()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if receivedEvent.Type != core.ClarifyNeeded {
		t.Errorf("expected ClarifyNeeded event, got %s", receivedEvent.Type)
	}
	if receivedEvent.Data != "What color is the sky?" {
		t.Errorf("unexpected event data: %v", receivedEvent.Data)
	}
	mu.Unlock()

	tool.Respond("Blue")
	wg.Wait()
}
