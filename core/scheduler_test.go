package core

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestCronScheduler_ScheduleAndList(t *testing.T) {
	s := NewCronScheduler()
	id, err := s.Schedule("daily", "0 9 * * *", "standup summary")
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty task ID")
	}
	if s.TaskCount() != 1 {
		t.Errorf("expected 1 task, got %d", s.TaskCount())
	}

	tasks := s.List()
	if len(tasks) != 1 || tasks[0].Name != "daily" {
		t.Errorf("list mismatch: %+v", tasks)
	}
}

func TestCronScheduler_Get(t *testing.T) {
	s := NewCronScheduler()
	id, _ := s.Schedule("test", "* * * * *", "every minute")

	task := s.Get(id)
	if task == nil || task.Name != "test" {
		t.Errorf("Get failed: %+v", task)
	}
	if s.Get("nonexistent") != nil {
		t.Error("Get for nonexistent ID should return nil")
	}
}

func TestCronScheduler_Unschedule(t *testing.T) {
	s := NewCronScheduler()
	id, _ := s.Schedule("tmp", "* * * * *", "temp")

	if !s.Unschedule(id) {
		t.Error("Unschedule should return true for existing ID")
	}
	if s.Unschedule(id) {
		t.Error("Unschedule should return false for already-removed ID")
	}
	if s.TaskCount() != 0 {
		t.Errorf("expected 0 tasks after unschedule, got %d", s.TaskCount())
	}
}

func TestCronScheduler_EnableDisable(t *testing.T) {
	s := NewCronScheduler()
	id, _ := s.Schedule("test", "* * * * *", "prompt")

	if err := s.Disable(id); err != nil {
		t.Fatalf("Disable failed: %v", err)
	}
	task := s.Get(id)
	if task.Enabled {
		t.Error("task should be disabled")
	}

	if err := s.Enable(id); err != nil {
		t.Fatalf("Enable failed: %v", err)
	}
	task = s.Get(id)
	if !task.Enabled {
		t.Error("task should be enabled")
	}

	if err := s.Enable("nope"); err == nil {
		t.Error("Enable for nonexistent should error")
	}
	if err := s.Disable("nope"); err == nil {
		t.Error("Disable for nonexistent should error")
	}
}

func TestCronScheduler_InvalidExpression(t *testing.T) {
	s := NewCronScheduler()
	_, err := s.Schedule("bad", "invalid cron", "prompt")
	if err == nil {
		t.Fatal("expected error for invalid expression")
	}
	_, err = s.Schedule("bad2", "* * * * * *", "6 fields")
	if err == nil {
		t.Fatal("expected error for 6-field expression")
	}
}

func TestCronScheduler_StartStop(t *testing.T) {
	s := NewCronScheduler()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.Start(ctx)
	if !s.IsRunning() {
		t.Error("should be running after Start")
	}

	s.Start(ctx) // idempotent
	if !s.IsRunning() {
		t.Error("should still be running after double Start")
	}

	s.Stop()
	if s.IsRunning() {
		t.Error("should not be running after Stop")
	}

	s.Stop() // idempotent
}

func TestCronScheduler_SetNextRunAt(t *testing.T) {
	s := NewCronScheduler()
	id, _ := s.Schedule("test", "0 0 1 1 *", "yearly")

	past := time.Now().Add(-time.Hour)
	err := s.SetNextRunAt(id, past)
	if err != nil {
		t.Fatalf("SetNextRunAt failed: %v", err)
	}
	task := s.Get(id)
	if !task.NextRunAt.Equal(past) {
		t.Errorf("NextRunAt mismatch: got %v, want %v", task.NextRunAt, past)
	}

	if err = s.SetNextRunAt("nope", time.Now()); err == nil {
		t.Error("SetNextRunAt for nonexistent should error")
	}
}

func TestCronScheduler_CheckAndFire_Callback(t *testing.T) {
	s := NewCronScheduler()

	var firedTask ScheduledTask
	var firedOnce sync.Once
	done := make(chan struct{})

	s.SetCallback(func(_ context.Context, task ScheduledTask) {
		firedOnce.Do(func() {
			firedTask = task
			close(done)
		})
	})

	id, _ := s.Schedule("fire-test", "* * * * *", "test prompt")
	s.SetNextRunAt(id, time.Now().Add(-time.Second))

	s.CheckAndFire()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("callback did not fire within timeout")
	}

	if firedTask.ID != id {
		t.Errorf("expected callback to fire for task %s, got %+v", id, firedTask)
	}
}

func TestParseCronExpression_EdgeCases(t *testing.T) {
	tests := []struct {
		expr   string
		err    bool
		fields int
	}{
		{"* * * * *", false, 5},
		{"0 9 * * 1-5", false, 5},
		{"*/15 * * * *", false, 5},
		{"0 0 1 1 *", false, 5},
		{"30 */2 1-15 * 1-5", false, 5},
		{"0,30 8,17 * * *", false, 5},
		{"", true, 0},
		{"* * *", true, 0},
		{"* * * * * *", true, 0},
		{"60 * * * *", true, 0},
		{"* 25 * * *", true, 0},
		{"* * 32 * *", true, 0},
		{"* * * 13 *", true, 0},
		{"* * * * 7", true, 0},
	}
	for _, tt := range tests {
		result, err := parseCronExpression(tt.expr)
		if (err != nil) != tt.err {
			t.Errorf("parseCronExpression(%q): err=%v, wantErr=%v", tt.expr, err, tt.err)
			continue
		}
		if !tt.err && len(result) != tt.fields {
			t.Errorf("parseCronExpression(%q): expected %d fields, got %d", tt.expr, tt.fields, len(result))
		}
	}
}

func TestScheduledTask_Fields(t *testing.T) {
	now := time.Now()
	task := ScheduledTask{
		ID:         "1",
		Name:       "test",
		Expression: "* * * * *",
		Prompt:     "hello",
		Enabled:    true,
		CreatedAt:  now,
		NextRunAt:  now.Add(time.Minute),
		RunCount:   3,
	}
	if task.ID != "1" || task.Prompt != "hello" || task.RunCount != 3 {
		t.Errorf("field mismatch: %+v", task)
	}
}

func TestCronScheduler_ConcurrentAccess(t *testing.T) {
	s := NewCronScheduler()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Schedule("task", "* * * * *", "prompt")
			s.List()
			s.TaskCount()
		}(i)
	}
	wg.Wait()
}
