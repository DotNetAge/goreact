package model

import (
	"testing"
)

func TestNewModel(t *testing.T) {
	t.Run("valid model", func(t *testing.T) {
		model, err := NewModel("gpt-4", "openai", "gpt-4")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if model.Name != "gpt-4" {
			t.Errorf("Expected 'gpt-4', got %q", model.Name)
		}
		if model.Provider != "openai" {
			t.Errorf("Expected 'openai', got %q", model.Provider)
		}
		if model.Temperature != DefaultTemperature {
			t.Errorf("Expected default temperature, got %f", model.Temperature)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := NewModel("", "openai", "gpt-4")
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("empty provider", func(t *testing.T) {
		_, err := NewModel("name", "", "model-id")
		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("empty model id", func(t *testing.T) {
		_, err := NewModel("name", "openai", "")
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestModel_WithAPIKey(t *testing.T) {
	model := &Model{Name: "test", Provider: "openai", ModelID: "id"}
	result := model.WithAPIKey("secret")

	if result.APIKey != "secret" {
		t.Errorf("Expected 'secret', got %q", result.APIKey)
	}
	if result != model {
		t.Error("Expected same model instance")
	}
}

func TestModel_WithBaseURL(t *testing.T) {
	model := &Model{Name: "test"}
	result := model.WithBaseURL("https://api.example.com")

	if result.BaseURL != "https://api.example.com" {
		t.Errorf("Expected URL, got %q", result.BaseURL)
	}
}

func TestModel_WithFeatureVision(t *testing.T) {
	model := &Model{}
	result := model.WithFeatureVision(true)

	if !result.Features.Vision {
		t.Error("Expected Vision to be true")
	}
}

func TestModel_WithFeatureToolCalling(t *testing.T) {
	model := &Model{}
	result := model.WithFeatureToolCalling(true)

	if !result.Features.ToolCalling {
		t.Error("Expected ToolCalling to be true")
	}
}

func TestModel_WithFeatureStreaming(t *testing.T) {
	model := &Model{}
	result := model.WithFeatureStreaming(true)

	if !result.Features.Streaming {
		t.Error("Expected Streaming to be true")
	}
}

func TestModel_WithFeatureThinking(t *testing.T) {
	model := &Model{}
	result := model.WithFeatureThinking(true)

	if !result.Features.Thinking {
		t.Error("Expected Thinking to be true")
	}
}

func TestModel_WithFeatureFileAttachment(t *testing.T) {
	model := &Model{}
	result := model.WithFeatureFileAttachment(true)

	if !result.Features.FileAttachment {
		t.Error("Expected FileAttachment to be true")
	}
}

func TestModel_WithTemperature(t *testing.T) {
	t.Run("valid temperature", func(t *testing.T) {
		model := &Model{}
		result, err := model.WithTemperature(1.5)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result.Temperature != 1.5 {
			t.Errorf("Expected 1.5, got %f", result.Temperature)
		}
	})

	t.Run("invalid temperature too low", func(t *testing.T) {
		model := &Model{}
		_, err := model.WithTemperature(-0.1)

		if err == nil {
			t.Error("Expected error")
		}
	})

	t.Run("invalid temperature too high", func(t *testing.T) {
		model := &Model{}
		_, err := model.WithTemperature(2.1)

		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestModel_WithMaxTokens(t *testing.T) {
	t.Run("valid max tokens", func(t *testing.T) {
		model := &Model{}
		result, err := model.WithMaxTokens(1000)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result.MaxTokens != 1000 {
			t.Errorf("Expected 1000, got %d", result.MaxTokens)
		}
	})

	t.Run("invalid max tokens", func(t *testing.T) {
		model := &Model{}
		_, err := model.WithMaxTokens(0)

		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestModel_WithTimeout(t *testing.T) {
	t.Run("valid timeout", func(t *testing.T) {
		model := &Model{}
		result, err := model.WithTimeout(60)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result.Timeout != 60 {
			t.Errorf("Expected 60, got %d", result.Timeout)
		}
	})

	t.Run("invalid timeout", func(t *testing.T) {
		model := &Model{}
		_, err := model.WithTimeout(0)

		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestConstants(t *testing.T) {
	if DefaultTemperature != 0.7 {
		t.Errorf("Expected 0.7, got %f", DefaultTemperature)
	}
	if DefaultMaxTokens != 4096 {
		t.Errorf("Expected 4096, got %d", DefaultMaxTokens)
	}
	if DefaultTimeout != 30 {
		t.Errorf("Expected 30, got %d", DefaultTimeout)
	}
}