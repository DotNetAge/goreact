package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ray/goreact/pkg/tool/builtin"
)

func main() {
	fmt.Println("=== Git Tool 示例 ===")

	// 创建临时目录用于测试
	tempDir := filepath.Join(os.TempDir(), "git-tool-test")
	os.RemoveAll(tempDir) // 清理旧的测试目录
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	gitTool := builtin.NewGit()

	// ============================================================
	// 1. Clone - 克隆仓库
	// ============================================================
	fmt.Println("\n--- 1. Clone Repository ---")

	repoPath := filepath.Join(tempDir, "test-repo")
	result, err := gitTool.Execute(map[string]any{
		"operation": "clone",
		"url":       "https://github.com/golang/example.git",
		"path":      repoPath,
		"depth":     1, // 浅克隆，只克隆最新的提交
	})

	if err != nil {
		fmt.Printf("❌ Clone failed: %v\n", err)
	} else {
		fmt.Printf("✅ Clone successful: %v\n", result)
	}

	// ============================================================
	// 2. Status - 查看状态
	// ============================================================
	fmt.Println("\n--- 2. Repository Status ---")

	result, err = gitTool.Execute(map[string]any{
		"operation": "status",
		"path":      repoPath,
	})

	if err != nil {
		fmt.Printf("❌ Status failed: %v\n", err)
	} else {
		fmt.Printf("✅ Status: %v\n", result)
	}

	// ============================================================
	// 3. Branch - 分支操作
	// ============================================================
	fmt.Println("\n--- 3. Branch Operations ---")

	// 列出所有分支
	result, err = gitTool.Execute(map[string]any{
		"operation": "branch",
		"path":      repoPath,
		"action":    "list",
	})

	if err != nil {
		fmt.Printf("❌ Branch list failed: %v\n", err)
	} else {
		fmt.Printf("✅ Branches: %v\n", result)
	}

	// 创建新分支
	result, err = gitTool.Execute(map[string]any{
		"operation": "branch",
		"path":      repoPath,
		"action":    "create",
		"name":      "test-branch",
	})

	if err != nil {
		fmt.Printf("❌ Branch create failed: %v\n", err)
	} else {
		fmt.Printf("✅ Branch created: %v\n", result)
	}

	// ============================================================
	// 4. Checkout - 切换分支
	// ============================================================
	fmt.Println("\n--- 4. Checkout Branch ---")

	result, err = gitTool.Execute(map[string]any{
		"operation": "checkout",
		"path":      repoPath,
		"branch":    "test-branch",
	})

	if err != nil {
		fmt.Printf("❌ Checkout failed: %v\n", err)
	} else {
		fmt.Printf("✅ Checkout successful: %v\n", result)
	}

	// ============================================================
	// 5. Log - 查看提交历史
	// ============================================================
	fmt.Println("\n--- 5. Commit Log ---")

	result, err = gitTool.Execute(map[string]any{
		"operation": "log",
		"path":      repoPath,
		"limit":     5,
	})

	if err != nil {
		fmt.Printf("❌ Log failed: %v\n", err)
	} else {
		fmt.Printf("✅ Recent commits: %v\n", result)
	}

	// ============================================================
	// 6. Remote - 远程仓库管理
	// ============================================================
	fmt.Println("\n--- 6. Remote Management ---")

	result, err = gitTool.Execute(map[string]any{
		"operation": "remote",
		"path":      repoPath,
		"action":    "list",
	})

	if err != nil {
		fmt.Printf("❌ Remote list failed: %v\n", err)
	} else {
		fmt.Printf("✅ Remotes: %v\n", result)
	}

	// ============================================================
	// 7. 错误处理示例
	// ============================================================
	fmt.Println("\n--- 7. Error Handling ---")

	// 尝试克隆不存在的仓库
	result, err = gitTool.Execute(map[string]any{
		"operation": "clone",
		"url":       "https://github.com/nonexistent/repo.git",
		"path":      filepath.Join(tempDir, "nonexistent"),
	})

	if err != nil {
		fmt.Printf("✅ Expected error caught: %v\n", err)
	}

	// 尝试在不存在的路径上执行操作
	result, err = gitTool.Execute(map[string]any{
		"operation": "status",
		"path":      "/nonexistent/path",
	})

	if err != nil {
		fmt.Printf("✅ Expected error caught: %v\n", err)
	}

	fmt.Println("\n=== 示例完成 ===")
}
