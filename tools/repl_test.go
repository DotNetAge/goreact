package tools

import (
	"context"
	"testing"
)

func TestREPLTool_Info(t *testing.T) {
	tool := NewREPLTool()
	info := tool.Info()
	if info.Name != "repl" {
		t.Errorf("Name = %q, want %q", info.Name, "repl")
	}
	if len(info.Parameters) == 0 {
		t.Error("expected parameters")
	}
}

func TestREPLTool_Execute_MissingCode(t *testing.T) {
	tool := NewREPLTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error for missing code parameter")
	}
}

func TestREPLTool_Execute_ValidGoCode(t *testing.T) {
	tool := NewREPLTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"code": "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello from repl\")\n}",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	s := result.(string)
	if s == "" {
		t.Error("expected non-empty output from valid Go code")
	}
}

func TestREPLTool_Execute_SyntaxError(t *testing.T) {
	tool := NewREPLTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"code": "package main\n\nfunc main( {\n\tprintln(broken)\n}",
	})
	if err == nil {
		t.Error("expected error for syntax error in Go code")
	}
	if result == nil {
		t.Error("expected non-nil result even on compile error (stderr output)")
	}
}
