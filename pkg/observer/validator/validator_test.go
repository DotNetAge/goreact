package validator

import (
	"testing"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/types"
)

func TestHTTPStatusRule200(t *testing.T) {
	v := New(WithHTTPStatusRule())
	ctx := core.NewContext()

	vr := v.Validate(&types.ExecutionResult{
		Success: true,
		Output:  `{"status": 200, "data": "ok"}`,
	}, ctx)

	if !vr.IsValid {
		t.Errorf("HTTP 200 should be valid: %v", vr.Issues)
	}
}

func TestHTTPStatusRule404(t *testing.T) {
	v := New(WithHTTPStatusRule())
	ctx := core.NewContext()

	vr := v.Validate(&types.ExecutionResult{
		Success: true,
		Output:  `{"status": 404, "body": "Not Found"}`,
	}, ctx)

	if vr.IsValid {
		t.Error("HTTP 404 should be invalid")
	}
	if len(vr.Issues) == 0 {
		t.Error("should have issues")
	}
}

func TestHTTPStatusRule500(t *testing.T) {
	v := New(WithHTTPStatusRule())
	ctx := core.NewContext()

	vr := v.Validate(&types.ExecutionResult{
		Success: true,
		Output:  `{"status": 500, "body": "Internal Server Error"}`,
	}, ctx)

	if vr.IsValid {
		t.Error("HTTP 500 should be invalid")
	}
}

func TestErrorPatternRule(t *testing.T) {
	v := New(WithErrorPatternRule())
	ctx := core.NewContext()

	// 包含 error
	vr := v.Validate(&types.ExecutionResult{
		Success: true,
		Output:  `{"error": "invalid API key", "code": 401}`,
	}, ctx)
	if vr.IsValid {
		t.Error("output with 'error' should be invalid")
	}

	// 正常输出
	vr = v.Validate(&types.ExecutionResult{
		Success: true,
		Output:  `{"data": "hello"}`,
	}, ctx)
	if !vr.IsValid {
		t.Errorf("normal output should be valid: %v", vr.Issues)
	}
}

func TestEmptyResultRule(t *testing.T) {
	v := New(WithEmptyResultRule())
	ctx := core.NewContext()

	tests := []struct {
		name    string
		output  any
		isValid bool
		hasSugg bool
	}{
		{"empty array", "[]", true, true},
		{"empty string", "", true, true},
		{"nil", nil, true, true},
		{"normal", "hello", true, false},
		{"array with data", "[1,2,3]", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vr := v.Validate(&types.ExecutionResult{
				Success: true,
				Output:  tt.output,
			}, ctx)
			if vr.IsValid != tt.isValid {
				t.Errorf("expected valid=%v, got %v", tt.isValid, vr.IsValid)
			}
			if tt.hasSugg && len(vr.Suggestions) == 0 {
				t.Error("expected suggestions for empty result")
			}
		})
	}
}

func TestCompositeRules(t *testing.T) {
	v := New(
		WithHTTPStatusRule(),
		WithErrorPatternRule(),
		WithEmptyResultRule(),
	)
	ctx := core.NewContext()

	// 多个问题
	vr := v.Validate(&types.ExecutionResult{
		Success: true,
		Output:  `{"status": 500, "error": "server crash"}`,
	}, ctx)

	if vr.IsValid {
		t.Error("should be invalid with multiple issues")
	}
	if len(vr.Issues) < 2 {
		t.Errorf("expected at least 2 issues, got %d: %v", len(vr.Issues), vr.Issues)
	}
}

func TestFailedExecution(t *testing.T) {
	v := New(WithHTTPStatusRule())
	ctx := core.NewContext()

	vr := v.Validate(&types.ExecutionResult{
		Success: false,
		Error:   nil,
	}, ctx)

	if vr.IsValid {
		t.Error("failed execution should be invalid")
	}
}
