package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ray/goreact/pkg/tools/builtin"
)

func main() {
	fmt.Println("🕐 Cron Tool Examples")
	fmt.Println("====================")

	// 创建 Cron工具
	cron := builtin.NewCron()
	ctx := context.Background()

	// 示例 1: 验证 cron 表达式
	fmt.Println("\n1️⃣  Validate Cron Expression:")
	result, err := cron.Execute(ctx, map[string]any{
		"operation":  "validate",
		"expression": "*/5 * * * *",
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("   Expression: */5 * * * *\n")
	fmt.Printf("   Valid: %v\n\n", result.(map[string]any)["valid"])

	// 示例 2: 解析 cron 表达式
	fmt.Println("2️⃣  Parse Cron Expression:")
	result, err = cron.Execute(ctx, map[string]any{
		"operation":  "parse",
		"expression": "0 12 * * *",
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	resMap := result.(map[string]any)
	fmt.Printf("   Expression: 0 12 * * *\n")
	fmt.Printf("   Minute: %v\n", resMap["minute"])
	fmt.Printf("   Hour: %v\n", resMap["hour"])
	fmt.Printf("   Day: %v\n", resMap["day"])
	fmt.Printf("   Month: %v\n", resMap["month"])
	fmt.Printf("   Weekday: %v\n\n", resMap["weekday"])

	// 示例 3: 计算下一个执行时间
	fmt.Println("3️⃣  Calculate Next Occurrences:")
	result, err = cron.Execute(ctx, map[string]any{
		"operation":  "next",
		"expression": "0 9 * * 1-5", // 工作日早上 9 点
		"from":       "2026-03-19T10:00:00Z",
		"count":      5.0,
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	times := result.([]string)
	fmt.Printf("   Expression: 0 9 * * 1-5 (Mon-Fri at 09:00)\n")
	fmt.Printf("   From: 2026-03-19T10:00:00Z\n")
	fmt.Printf("   Next 5 occurrences:\n")
	for i, t := range times {
		fmt.Printf("     %d. %s\n", i+1, t)
	}
	fmt.Println()

	// 示例 4: 复杂 cron 表达式
	fmt.Println("4️⃣  Complex Expression:")
	result, err = cron.Execute(ctx, map[string]any{
		"operation":  "parse",
		"expression": "*/15 9-17 * * 1-5",
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	resMap = result.(map[string]any)
	fmt.Printf("   Expression: */15 9-17 * * 1-5\n")
	fmt.Printf("   Meaning: Every 15 minutes from 09:00 to 17:00, Monday to Friday\n")
	fmt.Printf("   Minutes: %v\n", resMap["minute"])
	fmt.Printf("   Hours: %v\n", resMap["hour"])
	fmt.Printf("   Weekdays: %v\n\n", resMap["weekday"])

	// 示例 5: 无效的 cron 表达式
	fmt.Println("5️⃣  Invalid Expression:")
	result, err = cron.Execute(ctx, map[string]any{
		"operation":  "validate",
		"expression": "60 25 * * *", // 无效的小时和分钟
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	resMap = result.(map[string]any)
	fmt.Printf("   Expression: 60 25 * * *\n")
	fmt.Printf("   Valid: %v\n", resMap["valid"])
	fmt.Printf("   Error: %v\n\n", resMap["error"])

	fmt.Println("✅ All examples completed!")
}
