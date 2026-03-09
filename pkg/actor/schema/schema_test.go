package schema

import (
	"testing"
)

func TestParamBuilder(t *testing.T) {
	p := Param("name", String, "User name").Required()
	if p.name != "name" || p.typ != String || p.description != "User name" || !p.required {
		t.Errorf("Param builder failed: %+v", p)
	}

	p2 := Param("age", Int, "User age").Default(18).Range(1, 150)
	if p2.defaultVal != 18 || p2.min != float64(1) || p2.max != float64(150) {
		t.Errorf("Param builder with default/range failed: %+v", p2)
	}

	p3 := Param("color", String, "Favorite color").Enum("red", "green", "blue")
	if len(p3.enum) != 3 {
		t.Errorf("Param builder with enum failed: %+v", p3)
	}
}

func TestDefine(t *testing.T) {
	s := Define(
		Param("name", String, "Name").Required(),
		Param("age", Int, "Age").Default(18),
	)

	if len(s.params) != 2 {
		t.Errorf("Define should have 2 params, got %d", len(s.params))
	}
}

func TestValidateRequired(t *testing.T) {
	s := Define(
		Param("name", String, "Name").Required(),
		Param("age", Int, "Age").Default(18),
	)

	// 缺少必需参数
	_, err := s.Validate(map[string]any{})
	if err == nil {
		t.Error("should fail when required param is missing")
	}

	// 提供必需参数
	p, err := s.Validate(map[string]any{"name": "Alice"})
	if err != nil {
		t.Errorf("should pass with required param: %v", err)
	}
	if p.GetString("name") != "Alice" {
		t.Errorf("expected 'Alice', got '%s'", p.GetString("name"))
	}
	if p.GetInt("age") != 18 {
		t.Errorf("expected default age 18, got %d", p.GetInt("age"))
	}
}

func TestValidateEnum(t *testing.T) {
	s := Define(
		Param("color", String, "Color").Enum("red", "green", "blue").Required(),
	)

	// 有效值
	_, err := s.Validate(map[string]any{"color": "red"})
	if err != nil {
		t.Errorf("should accept valid enum value: %v", err)
	}

	// 无效值
	_, err = s.Validate(map[string]any{"color": "yellow"})
	if err == nil {
		t.Error("should reject invalid enum value")
	}
}

func TestValidateRange(t *testing.T) {
	s := Define(
		Param("age", Int, "Age").Range(1, 150).Required(),
	)

	// 有效范围
	_, err := s.Validate(map[string]any{"age": 25})
	if err != nil {
		t.Errorf("should accept valid range: %v", err)
	}

	// 超出范围
	_, err = s.Validate(map[string]any{"age": 200})
	if err == nil {
		t.Error("should reject out of range value")
	}

	// 低于范围
	_, err = s.Validate(map[string]any{"age": 0})
	if err == nil {
		t.Error("should reject below range value")
	}
}

func TestTypeConversion(t *testing.T) {
	s := Define(
		Param("count", Number, "Count").Required(),
		Param("name", String, "Name").Required(),
		Param("active", Bool, "Active").Required(),
	)

	tests := []struct {
		name   string
		params map[string]any
		check  func(ValidatedParams) bool
	}{
		{
			name:   "string to number",
			params: map[string]any{"count": "42", "name": "test", "active": true},
			check:  func(p ValidatedParams) bool { return p.GetFloat64("count") == 42.0 },
		},
		{
			name:   "int to number",
			params: map[string]any{"count": 42, "name": "test", "active": true},
			check:  func(p ValidatedParams) bool { return p.GetFloat64("count") == 42.0 },
		},
		{
			name:   "float64 to number",
			params: map[string]any{"count": 42.5, "name": "test", "active": true},
			check:  func(p ValidatedParams) bool { return p.GetFloat64("count") == 42.5 },
		},
		{
			name:   "string bool",
			params: map[string]any{"count": 1, "name": "test", "active": "true"},
			check:  func(p ValidatedParams) bool { return p.GetBool("active") == true },
		},
		{
			name:   "number to string",
			params: map[string]any{"count": 1, "name": 123, "active": true},
			check:  func(p ValidatedParams) bool { return p.GetString("name") == "123" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := s.Validate(tt.params)
			if err != nil {
				t.Errorf("validation failed: %v", err)
				return
			}
			if !tt.check(p) {
				t.Error("type conversion check failed")
			}
		})
	}
}

func TestTypeConversionFailure(t *testing.T) {
	s := Define(
		Param("count", Number, "Count").Required(),
	)

	_, err := s.Validate(map[string]any{"count": "hello"})
	if err == nil {
		t.Error("should fail when string cannot convert to number")
	}
}

func TestToolExecute(t *testing.T) {
	tool := NewTool(
		"calculator",
		"Perform arithmetic",
		Define(
			Param("a", Number, "First").Required(),
			Param("b", Number, "Second").Required(),
		),
		func(p ValidatedParams) (any, error) {
			return p.GetFloat64("a") + p.GetFloat64("b"), nil
		},
	)

	if tool.Name() != "calculator" {
		t.Errorf("expected name 'calculator', got '%s'", tool.Name())
	}

	result, err := tool.Execute(map[string]any{"a": 10, "b": 20})
	if err != nil {
		t.Errorf("execute failed: %v", err)
	}
	if result != 30.0 {
		t.Errorf("expected 30.0, got %v", result)
	}

	// 验证失败时不调用 handler
	_, err = tool.Execute(map[string]any{"a": 10})
	if err == nil {
		t.Error("should fail when required param is missing")
	}
}

func TestUserError(t *testing.T) {
	err := NewUserError("division by zero")
	if err.Error() != "division by zero" {
		t.Errorf("expected 'division by zero', got '%s'", err.Error())
	}

	err2 := NewUserError("invalid value: %d", 42)
	if err2.Error() != "invalid value: 42" {
		t.Errorf("expected 'invalid value: 42', got '%s'", err2.Error())
	}
}

func TestSchemaJSON(t *testing.T) {
	tool := NewTool(
		"test",
		"Test tool",
		Define(
			Param("name", String, "Name").Required(),
			Param("count", Int, "Count").Default(1),
		),
		func(p ValidatedParams) (any, error) { return nil, nil },
	)

	json := tool.SchemaJSON()
	if json == "" {
		t.Error("SchemaJSON should not be empty")
	}
	// 应该包含参数名
	if !containsStr(json, "name") || !containsStr(json, "count") {
		t.Errorf("SchemaJSON should contain param names: %s", json)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
