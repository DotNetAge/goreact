package main

import (
	"fmt"

	"github.com/ray/goreact/pkg/tool/builtin"
)

func main() {
	fmt.Println("=== Docker Tool 示例 ===")

	dockerTool := builtin.NewDocker()

	// ============================================================
	// 1. Pull - 拉取镜像
	// ============================================================
	fmt.Println("\n--- 1. Pull Image ---")

	result, err := dockerTool.Execute(map[string]any{
		"operation": "pull",
		"image":     "nginx",
		"tag":       "alpine",
	})

	if err != nil {
		fmt.Printf("❌ Pull failed: %v\n", err)
	} else {
		fmt.Printf("✅ Pull successful: %v\n", result)
	}

	// ============================================================
	// 2. Images - 列出镜像
	// ============================================================
	fmt.Println("\n--- 2. List Images ---")

	result, err = dockerTool.Execute(map[string]any{
		"operation": "images",
	})

	if err != nil {
		fmt.Printf("❌ Images failed: %v\n", err)
	} else {
		fmt.Printf("✅ Images: %v\n", result)
	}

	// ============================================================
	// 3. Run - 运行容器
	// ============================================================
	fmt.Println("\n--- 3. Run Container ---")

	result, err = dockerTool.Execute(map[string]any{
		"operation": "run",
		"image":     "nginx:alpine",
		"name":      "test-nginx",
		"ports":     []any{"8080:80"},
		"detach":    true,
		"restart":   "unless-stopped",
	})

	if err != nil {
		fmt.Printf("❌ Run failed: %v\n", err)
	} else {
		fmt.Printf("✅ Container started: %v\n", result)
	}

	// ============================================================
	// 4. Ps - 列出容器
	// ============================================================
	fmt.Println("\n--- 4. List Containers ---")

	result, err = dockerTool.Execute(map[string]any{
		"operation": "ps",
		"all":       true,
	})

	if err != nil {
		fmt.Printf("❌ Ps failed: %v\n", err)
	} else {
		fmt.Printf("✅ Containers: %v\n", result)
	}

	// ============================================================
	// 5. Logs - 查看日志
	// ============================================================
	fmt.Println("\n--- 5. Container Logs ---")

	result, err = dockerTool.Execute(map[string]any{
		"operation": "logs",
		"container": "test-nginx",
		"tail":      10,
	})

	if err != nil {
		fmt.Printf("❌ Logs failed: %v\n", err)
	} else {
		fmt.Printf("✅ Logs: %v\n", result)
	}

	// ============================================================
	// 6. Exec - 在容器中执行命令
	// ============================================================
	fmt.Println("\n--- 6. Execute Command ---")

	result, err = dockerTool.Execute(map[string]any{
		"operation": "exec",
		"container": "test-nginx",
		"command":   "nginx -v",
	})

	if err != nil {
		fmt.Printf("❌ Exec failed: %v\n", err)
	} else {
		fmt.Printf("✅ Exec result: %v\n", result)
	}

	// ============================================================
	// 7. Stats - 查看资源使用
	// ============================================================
	fmt.Println("\n--- 7. Container Stats ---")

	result, err = dockerTool.Execute(map[string]any{
		"operation": "stats",
		"container": "test-nginx",
	})

	if err != nil {
		fmt.Printf("❌ Stats failed: %v\n", err)
	} else {
		fmt.Printf("✅ Stats: %v\n", result)
	}

	// ============================================================
	// 8. Stop - 停止容器
	// ============================================================
	fmt.Println("\n--- 8. Stop Container ---")

	result, err = dockerTool.Execute(map[string]any{
		"operation": "stop",
		"container": "test-nginx",
	})

	if err != nil {
		fmt.Printf("❌ Stop failed: %v\n", err)
	} else {
		fmt.Printf("✅ Container stopped: %v\n", result)
	}

	// ============================================================
	// 9. Rm - 删除容器
	// ============================================================
	fmt.Println("\n--- 9. Remove Container ---")

	result, err = dockerTool.Execute(map[string]any{
		"operation": "rm",
		"container": "test-nginx",
		"force":     true,
	})

	if err != nil {
		fmt.Printf("❌ Rm failed: %v\n", err)
	} else {
		fmt.Printf("✅ Container removed: %v\n", result)
	}

	// ============================================================
	// 10. 错误处理示例
	// ============================================================
	fmt.Println("\n--- 10. Error Handling ---")

	// 尝试操作不存在的容器
	result, err = dockerTool.Execute(map[string]any{
		"operation": "stop",
		"container": "nonexistent-container",
	})

	if err != nil {
		fmt.Printf("✅ Expected error caught: %v\n", err)
	}

	fmt.Println("\n=== 示例完成 ===")
}
