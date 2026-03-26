package memory

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestNewDefaultMemoryBank(t *testing.T) {
	bank := NewDefaultMemoryBank()
	if bank == nil {
		t.Fatal("Expected non-nil memory bank")
	}
	if bank.working == nil {
		t.Error("Expected working memory to be initialized")
	}
	if bank.semantic == nil {
		t.Error("Expected semantic memory to be initialized")
	}
	if bank.muscle == nil {
		t.Error("Expected muscle memory to be initialized")
	}
}

func TestDefaultMemoryBank_Interfaces(t *testing.T) {
	bank := NewDefaultMemoryBank()

	var _ MemoryBank = bank
	var _ WorkingMemory = bank.working
	var _ SemanticMemory = bank.semantic
	var _ MuscleMemory[any] = bank.muscle
}

func TestDefaultMemoryBank_Compress(t *testing.T) {
	bank := NewDefaultMemoryBank()
	err := bank.Compress(context.Background(), "session1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestDefaultWorkingMemory_StoreAndRetrieve(t *testing.T) {
	wm := &defaultWorkingMemory{data: make(map[string]any)}

	t.Run("store and retrieve", func(t *testing.T) {
		ctx := context.Background()
		err := wm.Store(ctx, "session1", "key1", "value1")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		val, err := wm.Retrieve(ctx, "session1", "key1")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if val != "value1" {
			t.Errorf("Expected 'value1', got %v", val)
		}
	})

	t.Run("retrieve non-existent", func(t *testing.T) {
		ctx := context.Background()
		val, err := wm.Retrieve(ctx, "session1", "nonexistent")
		if err != nil {
			t.Errorf("Expected no error (returns nil value), got %v", err)
		}
		if val != nil {
			t.Errorf("Expected nil, got %v", val)
		}
	})

	t.Run("concurrent access", func(t *testing.T) {
		ctx := context.Background()
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				wm.Store(ctx, "session1", "key", i)
			}(i)
		}
		wg.Wait()

		val, _ := wm.Retrieve(ctx, "session1", "key")
		_ = val
	})
}

func TestDefaultWorkingMemory_RecallContext(t *testing.T) {
	wm := &defaultWorkingMemory{}
	ctx := context.Background()

	result, err := wm.RecallContext(ctx, "session1", "intent")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}
}

func TestDefaultWorkingMemory_Update(t *testing.T) {
	wm := &defaultWorkingMemory{}
	ctx := context.Background()

	err := wm.Update(ctx, "session1", "key", 0.5)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestDefaultSemanticMemory_RecallKnowledge(t *testing.T) {
	sm := &defaultSemanticMemory{}
	ctx := context.Background()

	result, err := sm.RecallKnowledge(ctx, "intent")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "No semantic knowledge found." {
		t.Errorf("Expected default message, got %q", result)
	}
}

func TestDefaultSemanticMemory_QueryGraph(t *testing.T) {
	sm := &defaultSemanticMemory{}
	ctx := context.Background()

	t.Run("default returns nil", func(t *testing.T) {
		result, err := sm.QueryGraph(ctx, "query", 2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got %v", result)
		}
	})
}

func TestDefaultMuscleMemory_RecallExperience(t *testing.T) {
	mm := &defaultMuscleMemory[any]{data: make(map[string]any)}
	ctx := context.Background()

	result, err := mm.RecallExperience(ctx, "skill1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}
}

func TestDefaultMuscleMemory_DistillExperience(t *testing.T) {
	mm := &defaultMuscleMemory[any]{data: make(map[string]any)}
	ctx := context.Background()

	err := mm.DistillExperience(ctx, "skill1", "action1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestDefaultMuscleMemory_LoadCompiledAction(t *testing.T) {
	mm := &defaultMuscleMemory[string]{data: make(map[string]string)}
	ctx := context.Background()

	t.Run("non-existent returns zero value", func(t *testing.T) {
		result, err := mm.LoadCompiledAction(ctx, "nonexistent")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "" {
			t.Errorf("Expected empty string, got %q", result)
		}
	})

	t.Run("existing returns value", func(t *testing.T) {
		mm.SaveCompiledAction(ctx, "skill1", "compiled action data")
		result, err := mm.LoadCompiledAction(ctx, "skill1")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "compiled action data" {
			t.Errorf("Expected 'compiled action data', got %q", result)
		}
	})
}

func TestDefaultMuscleMemory_SaveCompiledAction(t *testing.T) {
	mm := &defaultMuscleMemory[any]{data: make(map[string]any)}
	ctx := context.Background()

	t.Run("save and load", func(t *testing.T) {
		err := mm.SaveCompiledAction(ctx, "intent1", map[string]any{"key": "value"})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		result, err := mm.LoadCompiledAction(ctx, "intent1")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("Expected map, got %T", result)
		}
		if resultMap["key"] != "value" {
			t.Errorf("Expected 'value', got %v", resultMap["key"])
		}
	})

	t.Run("concurrent saves", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				mm.SaveCompiledAction(ctx, "intent", i)
			}(i)
		}
		wg.Wait()
	})
}

func TestMuscleMemory_Interface(t *testing.T) {
	var _ MuscleMemory[string] = (*defaultMuscleMemory[string])(nil)
}

func TestWorkingMemory_Interface(t *testing.T) {
	var _ WorkingMemory = (*defaultWorkingMemory)(nil)
}

func TestSemanticMemory_Interface(t *testing.T) {
	var _ SemanticMemory = (*defaultSemanticMemory)(nil)
}

func TestMemoryBank_Interface(t *testing.T) {
	var _ MemoryBank = (*DefaultMemoryBank)(nil)
}

type mockWorkingMemory struct {
	store    map[string]any
	recallErr error
	updateErr error
}

func (m *mockWorkingMemory) RecallContext(ctx context.Context, sessionID, intent string) (string, error) {
	return "", m.recallErr
}

func (m *mockWorkingMemory) Update(ctx context.Context, sessionID, key string, deltaWeight float64) error {
	return m.updateErr
}

func (m *mockWorkingMemory) Store(ctx context.Context, sessionID, key string, value any) error {
	if m.store == nil {
		m.store = make(map[string]any)
	}
	m.store[key] = value
	return nil
}

func (m *mockWorkingMemory) Retrieve(ctx context.Context, sessionID, key string) (any, error) {
	if m.store == nil {
		return nil, errors.New("not found")
	}
	v, ok := m.store[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return v, nil
}

type mockSemanticMemory struct {
	recallErr error
	queryErr  error
}

func (m *mockSemanticMemory) RecallKnowledge(ctx context.Context, intent string) (string, error) {
	return "", m.recallErr
}

func (m *mockSemanticMemory) QueryGraph(ctx context.Context, query string, depth int) (any, error) {
	return nil, m.queryErr
}

type mockMuscleMemory struct {
	recallErr  error
	distillErr error
	saveErr    error
	data       map[string]any
}

func (m *mockMuscleMemory) RecallExperience(ctx context.Context, skillName string) (string, error) {
	return "", m.recallErr
}

func (m *mockMuscleMemory) DistillExperience(ctx context.Context, skillName, newAction string) error {
	return m.distillErr
}

func (m *mockMuscleMemory) LoadCompiledAction(ctx context.Context, intent string) (any, error) {
	return nil, m.recallErr
}

func (m *mockMuscleMemory) SaveCompiledAction(ctx context.Context, intent string, sop any) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if m.data == nil {
		m.data = make(map[string]any)
	}
	m.data[intent] = sop
	return nil
}