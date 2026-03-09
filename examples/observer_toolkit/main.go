package main

import (
	"fmt"

	"github.com/ray/goreact/pkg/core"
	observerpresets "github.com/ray/goreact/pkg/core/observer/presets"
	"github.com/ray/goreact/pkg/observer/detector"
	"github.com/ray/goreact/pkg/observer/feedback"
	"github.com/ray/goreact/pkg/observer/validator"
	"github.com/ray/goreact/pkg/types"
)

func main() {
	fmt.Println("=== Observer Toolkit 示例 ===")

	ctx := core.NewContext()

	// ============================================================
	// 1. Smart Feedback Generator
	// ============================================================
	fmt.Println("\n--- 1. Smart Feedback Generator ---")

	gen := feedback.NewSmartGenerator()

	// 成功结果
	fb := gen.Generate(&types.ExecutionResult{
		Success:  true,
		Output:   300,
		Metadata: map[string]any{"tool_name": "calculator"},
	}, ctx)
	fmt.Printf("成功反馈: %s\n", fb)

	// HTTP 404 结果
	fb = gen.Generate(&types.ExecutionResult{
		Success:  true,
		Output:   `{"status": 404, "body": "Not Found"}`,
		Metadata: map[string]any{"tool_name": "http"},
	}, ctx)
	fmt.Printf("HTTP 404: %s\n", fb)

	// 失败结果
	fb = gen.Generate(&types.ExecutionResult{
		Success:  false,
		Error:    fmt.Errorf("connection refused"),
		Metadata: map[string]any{"tool_name": "http"},
	}, ctx)
	fmt.Printf("连接失败: %s\n", fb)

	// 空结果
	fb = gen.Generate(&types.ExecutionResult{
		Success:  true,
		Output:   "[]",
		Metadata: map[string]any{"tool_name": "search"},
	}, ctx)
	fmt.Printf("空结果: %s\n", fb)

	// ============================================================
	// 2. Result Validator
	// ============================================================
	fmt.Println("\n--- 2. Result Validator ---")

	v := validator.New(
		validator.WithHTTPStatusRule(),
		validator.WithErrorPatternRule(),
		validator.WithEmptyResultRule(),
	)

	// HTTP 200 正常
	vr := v.Validate(&types.ExecutionResult{
		Success: true,
		Output:  `{"status": 200, "data": [1,2,3]}`,
	}, ctx)
	fmt.Printf("HTTP 200: valid=%v\n", vr.IsValid)

	// HTTP 404
	vr = v.Validate(&types.ExecutionResult{
		Success: true,
		Output:  `{"status": 404, "body": "Not Found"}`,
	}, ctx)
	fmt.Printf("HTTP 404: valid=%v, issues=%v\n", vr.IsValid, vr.Issues)

	// 输出包含 error
	vr = v.Validate(&types.ExecutionResult{
		Success: true,
		Output:  `{"error": "invalid API key", "code": 401}`,
	}, ctx)
	fmt.Printf("含 error: valid=%v, issues=%v\n", vr.IsValid, vr.Issues)

	// 空结果
	vr = v.Validate(&types.ExecutionResult{
		Success: true,
		Output:  "[]",
	}, ctx)
	fmt.Printf("空结果: valid=%v, issues=%v, suggestions=%v\n", vr.IsValid, vr.Issues, vr.Suggestions)

	// ============================================================
	// 3. Loop Detector
	// ============================================================
	fmt.Println("\n--- 3. Loop Detector ---")

	ld := detector.NewLoopDetector(
		detector.WithMaxRepeats(2),
		detector.WithWindowSize(5),
	)

	// 第 1 次执行
	pattern := ld.Record("http", map[string]any{"url": "https://api.example.com"}, false)
	fmt.Printf("第 1 次: detected=%v\n", pattern.Detected)

	// 第 2 次执行（相同参数，相同失败）
	pattern = ld.Record("http", map[string]any{"url": "https://api.example.com"}, false)
	fmt.Printf("第 2 次: detected=%v\n", pattern.Detected)
	if pattern.Detected {
		fmt.Printf("  类型: %s\n", pattern.Type)
		fmt.Printf("  建议: %s\n", pattern.Suggestion)
	}

	// 不同参数不会触发
	pattern = ld.Record("http", map[string]any{"url": "https://other.com"}, false)
	fmt.Printf("不同参数: detected=%v\n", pattern.Detected)

	// 成功的操作不会触发
	ld2 := detector.NewLoopDetector(detector.WithMaxRepeats(2))
	ld2.Record("calculator", map[string]any{"a": 1}, true)
	pattern = ld2.Record("calculator", map[string]any{"a": 1}, true)
	fmt.Printf("成功重复: detected=%v\n", pattern.Detected)

	// ============================================================
	// 4. Observer Presets
	// ============================================================
	fmt.Println("\n--- 4. Observer Presets ---")

	// SmartObserver
	smart := observerpresets.NewSmartObserver()
	fb2, _ := smart.Observe(&types.ExecutionResult{
		Success:  true,
		Output:   42,
		Metadata: map[string]any{"tool_name": "calculator"},
	}, ctx)
	fmt.Printf("SmartObserver: continue=%v, msg=%s\n", fb2.ShouldContinue, fb2.Message)

	// StrictObserver
	strict := observerpresets.NewStrictObserver()
	fb2, _ = strict.Observe(&types.ExecutionResult{
		Success: true,
		Output:  `{"error": "invalid key"}`,
		Metadata: map[string]any{"tool_name": "http"},
	}, ctx)
	fmt.Printf("StrictObserver: continue=%v, msg=%s\n", fb2.ShouldContinue, fb2.Message)

	fmt.Println("\n=== 示例完成 ===")
}
