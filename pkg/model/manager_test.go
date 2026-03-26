package model

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("Expected non-nil manager")
	}
	if m.models == nil {
		t.Error("Expected models map to be initialized")
	}
}

func TestManager_RegisterModel(t *testing.T) {
	m := NewManager()
	model := &Model{Name: "test", Provider: "openai", ModelID: "gpt-4", APIKey: "test", Temperature: 0.7, MaxTokens: 1000, Timeout: 30}

	err := m.RegisterModel(model)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestManager_RegisterModel_Validation(t *testing.T) {
	m := NewManager()

	t.Run("nil model", func(t *testing.T) {
		err := m.RegisterModel(nil)
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		model := &Model{Name: "", Provider: "openai", ModelID: "id"}
		err := m.RegisterModel(model)
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("invalid provider", func(t *testing.T) {
		model := &Model{Name: "test", Provider: "invalid", ModelID: "id", Temperature: 0.7, MaxTokens: 1000, Timeout: 30}
		err := m.RegisterModel(model)
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("invalid temperature", func(t *testing.T) {
		model := &Model{Name: "test", Provider: "openai", ModelID: "id", Temperature: 3.0, MaxTokens: 1000, Timeout: 30}
		err := m.RegisterModel(model)
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("invalid max tokens", func(t *testing.T) {
		model := &Model{Name: "test", Provider: "openai", ModelID: "id", Temperature: 0.7, MaxTokens: 0, Timeout: 30}
		err := m.RegisterModel(model)
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("invalid timeout", func(t *testing.T) {
		model := &Model{Name: "test", Provider: "openai", ModelID: "id", Temperature: 0.7, MaxTokens: 1000, Timeout: 0}
		err := m.RegisterModel(model)
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("openai without api key", func(t *testing.T) {
		model := &Model{Name: "test", Provider: "openai", ModelID: "id", Temperature: 0.7, MaxTokens: 1000, Timeout: 30}
		err := m.RegisterModel(model)
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestManager_GetModel(t *testing.T) {
	m := NewManager()
	m.RegisterModel(&Model{Name: "test", Provider: "openai", ModelID: "gpt-4", APIKey: "test", Temperature: 0.7, MaxTokens: 1000, Timeout: 30})

	t.Run("existing model", func(t *testing.T) {
		model, err := m.GetModel("test")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if model.Name != "test" {
			t.Errorf("Expected 'test', got %q", model.Name)
		}
	})

	t.Run("non-existing model", func(t *testing.T) {
		_, err := m.GetModel("nonexistent")
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := m.GetModel("")
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestManager_ListModels(t *testing.T) {
	m := NewManager()
	m.RegisterModel(&Model{Name: "a", Provider: "openai", ModelID: "id", APIKey: "test", Temperature: 0.7, MaxTokens: 1000, Timeout: 30})
	m.RegisterModel(&Model{Name: "b", Provider: "anthropic", ModelID: "id", APIKey: "test", Temperature: 0.7, MaxTokens: 1000, Timeout: 30})

	models := m.ListModels()
	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}
}

func TestManager_CreateLLMClient(t *testing.T) {
	t.Run("model not found", func(t *testing.T) {
		m := NewManager()
		_, err := m.CreateLLMClient("nonexistent")
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestManager_createClientFromModel(t *testing.T) {
	m := NewManager()

	t.Run("unsupported provider", func(t *testing.T) {
		model := &Model{Name: "test", Provider: "unsupported", ModelID: "id"}
		_, err := m.createClientFromModel(model)
		if err == nil {
			t.Error("Expected error")
		}
	})
}