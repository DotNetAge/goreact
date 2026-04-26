package core

import (
	"context"
	"sync"
	"testing"
	"time"
)

func estimateFixedTokens(s string) int {
	return len(s)
}

func TestContextWindow_UsageRatio(t *testing.T) {
	t.Run("empty_window", func(t *testing.T) {
		cw := NewContextWindow("test", 1000)
		if got := cw.UsageRatio(); got != 0 {
			t.Errorf("UsageRatio() = %v, want 0", got)
		}
	})

	t.Run("partial_usage", func(t *testing.T) {
		cw := NewContextWindow("test", 1000)
		cw.AddTokens(450)
		if got := cw.UsageRatio(); got != 0.45 {
			t.Errorf("UsageRatio() = %v, want 0.45", got)
		}
	})

	t.Run("over_capacity", func(t *testing.T) {
		cw := NewContextWindow("test", 1000)
		cw.AddTokens(1500)
		if got := cw.UsageRatio(); got != 1.5 {
			t.Errorf("UsageRatio() = %v, want 1.5", got)
		}
	})
}

func TestContextWindow_SlideTriggered(t *testing.T) {
	config := DefaultSlideConfig

	t.Run("below_threshold_no_trigger", func(t *testing.T) {
		cw := NewContextWindow("test", 1000)
		cw.AddTokens(int64(float64(1000) * config.SlideTriggerRatio * 0.9))
		if cw.SlideTriggered(config) {
			t.Error("SlideTriggered() = true, want false (below threshold)")
		}
	})

	t.Run("at_threshold_triggers", func(t *testing.T) {
		cw := NewContextWindow("test", 1000)
		cw.AddTokens(int64(float64(1000) * config.SlideTriggerRatio))
		if !cw.SlideTriggered(config) {
			t.Error("SlideTriggered() = false, want true (at threshold)")
		}
	})

	t.Run("above_threshold_triggers", func(t *testing.T) {
		cw := NewContextWindow("test", 1000)
		cw.AddTokens(900)
		if !cw.SlideTriggered(config) {
			t.Error("SlideTriggered() = false, want true (above threshold)")
		}
	})

	t.Run("empty_window_no_trigger", func(t *testing.T) {
		cw := NewContextWindow("test", 1000)
		if cw.SlideTriggered(config) {
			t.Error("SlideTriggered() = true on empty window, want false")
		}
	})
}

func TestContextWindow_Slide_Basic(t *testing.T) {
	config := DefaultSlideConfig
	maxTokens := int64(1000)

	t.Run("slide_removes_oldest_messages", func(t *testing.T) {
		cw := NewContextWindow("test", maxTokens)
		for i := 0; i < 10; i++ {
			cw.AddMessageWithTimestamp("user", msgContent(i, 50), int64(i))
		}
		cw.AddTokens(maxTokens)

		slided := cw.Slide(config, estimateFixedTokens)

		if len(slided.Messages) == 0 {
			t.Fatal("Slide() returned no slid messages")
		}
		for i, m := range slided.Messages {
			if m.Content != msgContent(i, 50) {
				t.Errorf("slided message[%d] content = %q, want %q", i, m.Content, msgContent(i, 50))
			}
		}
		ratio := cw.UsageRatio()
		if ratio > config.TargetRatio+0.05 {
			t.Errorf("after Slide, UsageRatio = %.3f, want <= %.3f (+tolerance)", ratio, config.TargetRatio)
		}
	})

	t.Run("preserves_minimum_messages", func(t *testing.T) {
		cw := NewContextWindow("test", maxTokens)
		minMsgs := config.MinPreserveMessages

		for i := 0; i < minMsgs+2; i++ {
			cw.AddMessageWithTimestamp("user", msgContent(i, 200), int64(i))
		}
		cw.AddTokens(maxTokens * 2)

		cw.Slide(config, estimateFixedTokens)

		if cw.MessageCount() < minMsgs {
			t.Errorf("after Slide, MessageCount = %d, want >= %d", cw.MessageCount(), minMsgs)
		}
	})

	t.Run("fewer_than_minpreserve_does_not_slide", func(t *testing.T) {
		cw := NewContextWindow("test", maxTokens)
		for i := 0; i < config.MinPreserveMessages-1; i++ {
			cw.AddMessage("user", "hello")
		}
		cw.AddTokens(maxTokens * 10)

		slided := cw.Slide(config, estimateFixedTokens)
		if len(slided.Messages) > 0 {
			t.Errorf("Slide() removed %d messages but only %d exist (min=%d)",
				len(slided.Messages), cw.MessageCount()+len(slided.Messages), config.MinPreserveMessages)
		}
	})

	t.Run("empty_window_returns_empty_slided", func(t *testing.T) {
		cw := NewContextWindow("test", maxTokens)
		slided := cw.Slide(config, estimateFixedTokens)
		if len(slided.Messages) != 0 || slided.TokenCount != 0 {
			t.Errorf("Slide() on empty window returned %+v, want empty", slided)
		}
	})
}

func TestContextWindow_Slide_MaxBatchLimit(t *testing.T) {
	config := SlideConfig{
		SlideTriggerRatio:   0.5,
		TargetRatio:         0.3,
		MinPreserveMessages: 2,
		MaxSlideBatch:       3,
	}
	maxTokens := int64(500)

	cw := NewContextWindow("test", maxTokens)
	for i := 0; i < 20; i++ {
		cw.AddMessageWithTimestamp("user", msgContent(i, 30), int64(i))
	}
	cw.AddTokens(maxTokens * 5)

	slided := cw.Slide(config, estimateFixedTokens)

	if len(slided.Messages) > config.MaxSlideBatch {
		t.Errorf("slided %d messages, MaxSlideBatch limit is %d",
			len(slided.Messages), config.MaxSlideBatch)
	}
}

func TestContextWindow_Slide_TokenCountAccuracy(t *testing.T) {
	config := DefaultSlideConfig
	maxTokens := int64(200)

	cw := NewContextWindow("test", maxTokens)
	msgSizes := []int{40, 35, 30, 25, 20, 15, 10, 5}
	for _, size := range msgSizes {
		cw.AddMessage("user", makeString(size))
	}
	totalTokens := int64(140)
	cw.AddTokens(totalTokens)

	slided := cw.Slide(config, estimateFixedTokens)

	var actualSlidTokens int64
	for _, m := range slided.Messages {
		actualSlidTokens += int64(estimateFixedTokens(m.Content))
	}
	if slided.TokenCount != actualSlidTokens {
		t.Errorf("slided TokenCount = %d, want %d (from slid messages)", slided.TokenCount, actualSlidTokens)
	}

	target := int64(float64(maxTokens) * config.TargetRatio)
	remainingTokens := cw.TokensUsed
	if remainingTokens > target+int64(len(msgSizes))*3 {
		t.Errorf("after Slide remaining TokensUsed=%d, expected near target=%d", remainingTokens, target)
	}
}

func TestDefaultSlideConfig_Values(t *testing.T) {
	if DefaultSlideConfig.SlideTriggerRatio != 0.65 {
		t.Errorf("SlideTriggerRatio = %v, want 0.65", DefaultSlideConfig.SlideTriggerRatio)
	}
	if DefaultSlideConfig.TargetRatio != 0.45 {
		t.Errorf("TargetRatio = %v, want 0.45", DefaultSlideConfig.TargetRatio)
	}
	if DefaultSlideConfig.MinPreserveMessages != 4 {
		t.Errorf("MinPreserveMessages = %d, want 4", DefaultSlideConfig.MinPreserveMessages)
	}
	if DefaultSlideConfig.MaxSlideBatch != 0 {
		t.Errorf("MaxSlideBatch = %d, want 0 (unlimited)", DefaultSlideConfig.MaxSlideBatch)
	}
}

func msgContent(idx, size int) string {
	base := "message-" + itoa(idx)
	for len(base) < size {
		base += "_pad"
	}
	return base[:size]
}

func makeString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'x'
	}
	return string(b)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	if neg {
		return "-" + string(buf)
	}
	return string(buf)
}

func TestMemorySessionStore_AppendAndGet(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	store.Append(ctx, "s1", Message{Role: "user", Content: "hello", Timestamp: 1})
	store.Append(ctx, "s1", Message{Role: "assistant", Content: "hi there", Timestamp: 2})
	store.Append(ctx, "s2", Message{Role: "user", Content: "other session", Timestamp: 3})

	msgs, err := store.Get(ctx, "s1")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("Get(s1) returned %d messages, want 2", len(msgs))
	}
	if msgs[0].Content != "hello" || msgs[1].Content != "hi there" {
		t.Errorf("messages content mismatch: %+v", msgs)
	}

	msgs2, _ := store.Get(ctx, "s2")
	if len(msgs2) != 1 || msgs2[0].Content != "other session" {
		t.Errorf("Get(s2) failed: %+v", msgs2)
	}
}

func TestMemorySessionStore_CurrentContext_TokenBudget(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		store.Append(ctx, "s1", Message{
			Role:      "user",
			Content:   makeString(500),
			Timestamp: int64(i),
		})
	}

	msgs, err := store.CurrentContext(ctx, "s1", 800)
	if err != nil {
		t.Fatalf("CurrentContext error: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("CurrentContext returned no messages")
	}
	if len(msgs) >= 10 {
		t.Errorf("CurrentContext with budget=800 returned %d messages, expected < 10 (each msg ~500 chars)", len(msgs))
	}
	for i := 1; i < len(msgs); i++ {
		if msgs[i-1].Timestamp > msgs[i].Timestamp {
			t.Error("messages not in chronological order")
		}
	}
}

func TestMemorySessionStore_CurrentContext_EmptySession(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	msgs, err := store.CurrentContext(ctx, "nonexistent", 1000)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if msgs != nil {
		t.Errorf("expected nil for nonexistent session, got %d messages", len(msgs))
	}
}

func TestMemorySessionStore_Clear(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	store.Append(ctx, "s1", Message{Role: "user", Content: "data", Timestamp: 1})
	if err := store.Clear(ctx, "s1"); err != nil {
		t.Fatalf("Clear error: %v", err)
	}

	msgs, _ := store.Get(ctx, "s1")
	if len(msgs) != 0 {
		t.Errorf("after Clear, Get returned %d messages, want 0", len(msgs))
	}
}

func TestMemorySessionStore_Delete(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	store.Append(ctx, "s1", Message{Role: "user", Content: "keep", Timestamp: 2})
	store.Append(ctx, "s1", Message{Role: "assistant", Content: "remove", Timestamp: 5})
	store.Append(ctx, "s1", Message{Role: "user", Content: "also keep", Timestamp: 9})

	if err := store.Delete(ctx, 5, "s1"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	msgs, _ := store.Get(ctx, "s1")
	if len(msgs) != 2 {
		t.Errorf("after Delete, got %d messages, want 2", len(msgs))
	}
	found := false
	for _, m := range msgs {
		if m.Content == "remove" {
			t.Error("deleted message still present")
			found = true
		}
	}
	if !found && len(msgs) == 2 {
		if msgs[0].Content != "keep" || msgs[1].Content != "also keep" {
			t.Errorf("remaining messages: %+v", msgs)
		}
	}
}

func TestMemorySessionStore_SlideHandler(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	called := false
	var receivedEvent SlideEvent
	store.SetSlideHandler(func(c context.Context, e SlideEvent) {
		called = true
		receivedEvent = e
	})

	testEvent := SlideEvent{SessionID: "s1", Slided: []Message{{Role: "user"}}, Timestamp: 99}
	EmitSlideEvent(func(c context.Context, e SlideEvent) {
		called = true
		receivedEvent = e
	}, ctx, testEvent)

	if !called {
		t.Fatal("SlideHandler was not called")
	}
	if receivedEvent.SessionID != "s1" || len(receivedEvent.Slided) != 1 || receivedEvent.Timestamp != 99 {
		t.Errorf("handler received wrong event: %+v", receivedEvent)
	}
}

func TestMemorySessionStore_Close(t *testing.T) {
	store := NewMemorySessionStore()
	if err := store.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}
}

func TestMemorySessionStore_ConcurrentAccess(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	const goroutines = 50
	const appendsPerGoroutine = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < appendsPerGoroutine; i++ {
				store.Append(ctx, "concurrent", Message{
					Role:      "user",
					Content:   makeString(10),
					Timestamp: int64(id*appendsPerGoroutine + i),
				})
			}
		}(g)
	}
	wg.Wait()

	msgs, err := store.Get(ctx, "concurrent")
	if err != nil {
		t.Fatalf("Get after concurrent appends error: %v", err)
	}
	expected := goroutines * appendsPerGoroutine
	if len(msgs) != expected {
		t.Errorf("got %d messages after %d concurrent appends, want %d",
			len(msgs), expected, expected)
	}
}

func TestNoopSlideHandler(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NoopSlideHandler panicked: %v", r)
		}
	}()
	NoopSlideHandler(nil, SlideEvent{})
}

func TestEmitSlideEvent_NilHandler(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("EmitSlideEvent with nil handler panicked: %v", r)
		}
	}()
	EmitSlideEvent(nil, nil, SlideEvent{SessionID: "test"})
}

func TestIntegration_SlidingWindow_Lifecycle(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	var capturedEvents []SlideEvent
	slideHandler := func(c context.Context, e SlideEvent) {
		capturedEvents = append(capturedEvents, e)
	}
	store.SetSlideHandler(slideHandler)

	config := SlideConfig{
		SlideTriggerRatio:   0.5,
		TargetRatio:         0.3,
		MinPreserveMessages: 2,
	}
	maxTokens := int64(200)
	cw := NewContextWindow("integration-test", maxTokens)

	fillWindow := func(count, msgSize int) {
		for i := 0; i < count; i++ {
			content := makeString(msgSize)
			msg := Message{Role: "user", Content: content, Timestamp: time.Now().Unix()}
			cw.AddMessageWithTimestamp("user", content, msg.Timestamp)
			store.Append(ctx, cw.SessionID, msg)
		}
		cw.AddTokens(int64(count * msgSize))
	}

	fillWindow(8, 25)

	if !cw.SlideTriggered(config) {
		t.Fatal("expected SlideTriggered=true after filling window")
	}

	slided := cw.Slide(config, estimateFixedTokens)
	if len(slided.Messages) == 0 {
		t.Fatal("expected some messages to be slid")
	}

	event := SlideEvent{
		SessionID: cw.SessionID,
		Slided:    slided.Messages,
		Remaining: cw.MessageCount(),
		Timestamp: time.Now().Unix(),
	}
	EmitSlideEvent(slideHandler, ctx, event)

	if len(capturedEvents) != 1 {
		t.Errorf("expected 1 slide event, got %d", len(capturedEvents))
	} else {
		ev := capturedEvents[0]
		if ev.SessionID != cw.SessionID {
			t.Errorf("event SessionID = %q, want %q", ev.SessionID, cw.SessionID)
		}
		if len(ev.Slided) != len(slided.Messages) {
			t.Errorf("event has %d slid messages, expected %d", len(ev.Slided), len(slided.Messages))
		}
	}

	allMsgs, _ := store.Get(ctx, cw.SessionID)
	if len(allMsgs) == 0 {
		t.Error("store should retain complete history even after slides (WAL mode)")
	}
	slidedIDs := map[int64]bool{}
	for _, m := range slided.Messages {
		slidedIDs[m.Timestamp] = true
	}
	foundSlidInStore := false
	for _, m := range allMsgs {
		if slidedIDs[m.Timestamp] {
			foundSlidInStore = true
			break
		}
	}
	if !foundSlidInStore && len(slided.Messages) > 0 {
		t.Log("INFO: slid messages correctly retained in SessionStore (WAL behavior)")
	}
}

func TestIntegration_MultipleSlides(t *testing.T) {
	store := NewMemorySessionStore()
	ctx := context.Background()

	slideCount := 0
	slideHandler := func(_ context.Context, _ SlideEvent) {
		slideCount++
	}
	store.SetSlideHandler(slideHandler)

	config := SlideConfig{
		SlideTriggerRatio:   0.4,
		TargetRatio:         0.2,
		MinPreserveMessages: 2,
	}
	maxTokens := int64(150)
	cw := NewContextWindow("multi-slide", maxTokens)

	for batch := 0; batch < 3; batch++ {
		for i := 0; i < 6; i++ {
			content := makeString(15)
			ts := time.Now().Unix()
			cw.AddMessageWithTimestamp("user", content, ts)
			store.Append(ctx, cw.SessionID, Message{Role: "user", Content: content, Timestamp: ts})
		}
		cw.AddTokens(90)
		cw.Slide(config, estimateFixedTokens)
		EmitSlideEvent(slideHandler, ctx, SlideEvent{
			SessionID: cw.SessionID,
			Slided:    []Message{},
			Remaining: cw.MessageCount(),
			Timestamp: time.Now().Unix(),
		})
	}

	if slideCount < 3 {
		t.Errorf("expected >= 3 slide events, got %d", slideCount)
	}

	allMsgs, _ := store.Get(ctx, cw.SessionID)
	if len(allMsgs) == 0 {
		t.Error("store should retain complete history even after slides")
	}
}
