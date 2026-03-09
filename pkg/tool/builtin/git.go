package builtin

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Git Git 版本控制工具
type Git struct{}

// NewGit 创建 Git 工具
func NewGit() *Git {
	return &Git{}
}

// Name 返回工具名称
func (g *Git) Name() string {
	return "git"
}

// Description 返回工具描述
func (g *Git) Description() string {
	return "Git version control operations: clone, pull, push, commit, status, branch, checkout, log, remote"
}

// Execute 执行 Git 操作
func (g *Git) Execute(params map[string]interface{}) (interface{}, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'operation' parameter")
	}

	switch operation {
	case "clone":
		return g.clone(params)
	case "pull":
		return g.pull(params)
	case "push":
		return g.push(params)
	case "commit":
		return g.commit(params)
	case "status":
		return g.status(params)
	case "branch":
		return g.branch(params)
	case "checkout":
		return g.checkout(params)
	case "log":
		return g.log(params)
	case "remote":
		return g.remote(params)
	case "fetch":
		return g.fetch(params)
	case "add":
		return g.add(params)
	case "diff":
		return g.diff(params)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// clone 克隆仓库
func (g *Git) clone(params map[string]interface{}) (interface{}, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'url' parameter")
	}

	path, _ := params["path"].(string)
	if path == "" {
		path = "."
	}

	args := []string{"clone"}

	// 分支
	if branch, ok := params["branch"].(string); ok && branch != "" {
		args = append(args, "-b", branch)
	}

	// 深度（浅克隆）
	if depth, ok := params["depth"].(float64); ok && depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%.0f", depth))
	}

	args = append(args, url, path)

	output, err := g.runGitCommand("", args...)
	if err != nil {
		return nil, g.formatError("clone", err, output)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Repository cloned successfully",
		"path":    path,
		"output":  output,
	}, nil
}

// pull 拉取更新
func (g *Git) pull(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	args := []string{"pull"}

	remote, _ := params["remote"].(string)
	if remote == "" {
		remote = "origin"
	}

	branch, _ := params["branch"].(string)

	if branch != "" {
		args = append(args, remote, branch)
	}

	// rebase
	if rebase, ok := params["rebase"].(bool); ok && rebase {
		args = append(args, "--rebase")
	}

	output, err := g.runGitCommand(path, args...)
	if err != nil {
		return nil, g.formatError("pull", err, output)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Pull completed successfully",
		"output":  output,
	}, nil
}

// push 推送更改
func (g *Git) push(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	args := []string{"push"}

	remote, _ := params["remote"].(string)
	if remote == "" {
		remote = "origin"
	}

	branch, _ := params["branch"].(string)

	// 强制推送（危险操作）
	if force, ok := params["force"].(bool); ok && force {
		args = append(args, "--force")
	}

	if branch != "" {
		args = append(args, remote, branch)
	} else {
		args = append(args, remote)
	}

	output, err := g.runGitCommand(path, args...)
	if err != nil {
		return nil, g.formatError("push", err, output)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Push completed successfully",
		"output":  output,
	}, nil
}

// commit 提交更改
func (g *Git) commit(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	message, ok := params["message"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'message' parameter")
	}

	args := []string{"commit", "-m", message}

	// 修改上次提交
	if amend, ok := params["amend"].(bool); ok && amend {
		args = append(args, "--amend")
	}

	output, err := g.runGitCommand(path, args...)
	if err != nil {
		return nil, g.formatError("commit", err, output)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Commit created successfully",
		"output":  output,
	}, nil
}

// status 查看状态
func (g *Git) status(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	output, err := g.runGitCommand(path, "status", "--porcelain", "-b")
	if err != nil {
		return nil, g.formatError("status", err, output)
	}

	return map[string]interface{}{
		"success": true,
		"output":  output,
	}, nil
}

// branch 分支操作
func (g *Git) branch(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	action, _ := params["action"].(string)
	if action == "" {
		action = "list"
	}

	var args []string
	var output string
	var err error

	switch action {
	case "list":
		args = []string{"branch", "-a"}
		output, err = g.runGitCommand(path, args...)

	case "create":
		name, ok := params["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' parameter for branch create")
		}
		args = []string{"branch", name}
		output, err = g.runGitCommand(path, args...)

	case "delete":
		name, ok := params["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'name' parameter for branch delete")
		}
		args = []string{"branch", "-d", name}
		output, err = g.runGitCommand(path, args...)

	case "rename":
		oldName, ok1 := params["old_name"].(string)
		newName, ok2 := params["new_name"].(string)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("missing 'old_name' or 'new_name' parameter for branch rename")
		}
		args = []string{"branch", "-m", oldName, newName}
		output, err = g.runGitCommand(path, args...)

	default:
		return nil, fmt.Errorf("unknown branch action: %s", action)
	}

	if err != nil {
		return nil, g.formatError("branch", err, output)
	}

	return map[string]interface{}{
		"success": true,
		"action":  action,
		"output":  output,
	}, nil
}

// checkout 切换分支
func (g *Git) checkout(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	branch, ok := params["branch"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'branch' parameter")
	}

	args := []string{"checkout"}

	// 如果不存在则创建
	if create, ok := params["create"].(bool); ok && create {
		args = append(args, "-b")
	}

	args = append(args, branch)

	output, err := g.runGitCommand(path, args...)
	if err != nil {
		return nil, g.formatError("checkout", err, output)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Switched to branch '%s'", branch),
		"output":  output,
	}, nil
}

// log 查看提交历史
func (g *Git) log(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	args := []string{"log", "--oneline"}

	// 限制数量
	limit := 10
	if l, ok := params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}
	args = append(args, fmt.Sprintf("-n%d", limit))

	// 按作者过滤
	if author, ok := params["author"].(string); ok && author != "" {
		args = append(args, fmt.Sprintf("--author=%s", author))
	}

	// 日期范围
	if since, ok := params["since"].(string); ok && since != "" {
		args = append(args, fmt.Sprintf("--since=%s", since))
	}
	if until, ok := params["until"].(string); ok && until != "" {
		args = append(args, fmt.Sprintf("--until=%s", until))
	}

	output, err := g.runGitCommand(path, args...)
	if err != nil {
		return nil, g.formatError("log", err, output)
	}

	return map[string]interface{}{
		"success": true,
		"output":  output,
	}, nil
}

// remote 远程仓库操作
func (g *Git) remote(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	action, _ := params["action"].(string)
	if action == "" {
		action = "list"
	}

	var args []string
	var output string
	var err error

	switch action {
	case "list":
		args = []string{"remote", "-v"}
		output, err = g.runGitCommand(path, args...)

	case "add":
		name, ok1 := params["name"].(string)
		url, ok2 := params["url"].(string)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("missing 'name' or 'url' parameter for remote add")
		}
		args = []string{"remote", "add", name, url}
		output, err = g.runGitCommand(path, args...)

	case "remove":
		name, ok := params["name"].(string)
		if !ok {
			return nil, fmt.Errorf("missing 'name' parameter for remote remove")
		}
		args = []string{"remote", "remove", name}
		output, err = g.runGitCommand(path, args...)

	case "set-url":
		name, ok1 := params["name"].(string)
		url, ok2 := params["url"].(string)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("missing 'name' or 'url' parameter for remote set-url")
		}
		args = []string{"remote", "set-url", name, url}
		output, err = g.runGitCommand(path, args...)

	default:
		return nil, fmt.Errorf("unknown remote action: %s", action)
	}

	if err != nil {
		return nil, g.formatError("remote", err, output)
	}

	return map[string]interface{}{
		"success": true,
		"action":  action,
		"output":  output,
	}, nil
}

// fetch 获取远程更新
func (g *Git) fetch(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	args := []string{"fetch"}

	remote, _ := params["remote"].(string)
	if remote != "" {
		args = append(args, remote)
	}

	// 删除不存在的远程分支
	if prune, ok := params["prune"].(bool); ok && prune {
		args = append(args, "--prune")
	}

	output, err := g.runGitCommand(path, args...)
	if err != nil {
		return nil, g.formatError("fetch", err, output)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Fetch completed successfully",
		"output":  output,
	}, nil
}

// add 添加到暂存区
func (g *Git) add(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	args := []string{"add"}

	// 添加所有文件
	if all, ok := params["all"].(bool); ok && all {
		args = append(args, "-A")
	} else {
		// 添加指定文件
		files, ok := params["files"].([]interface{})
		if !ok || len(files) == 0 {
			return nil, fmt.Errorf("missing 'files' parameter or 'all' flag")
		}
		for _, f := range files {
			if file, ok := f.(string); ok {
				args = append(args, file)
			}
		}
	}

	output, err := g.runGitCommand(path, args...)
	if err != nil {
		return nil, g.formatError("add", err, output)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Files added to staging area",
		"output":  output,
	}, nil
}

// diff 查看差异
func (g *Git) diff(params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	args := []string{"diff"}

	// 查看暂存区差异
	if staged, ok := params["staged"].(bool); ok && staged {
		args = append(args, "--staged")
	}

	// 指定文件
	if file, ok := params["file"].(string); ok && file != "" {
		args = append(args, file)
	}

	output, err := g.runGitCommand(path, args...)
	if err != nil {
		return nil, g.formatError("diff", err, output)
	}

	return map[string]interface{}{
		"success": true,
		"output":  output,
	}, nil
}

// runGitCommand 执行 Git 命令
func (g *Git) runGitCommand(workDir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if workDir != "" {
		cmd.Dir = workDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}

	return strings.TrimSpace(output), err
}

// formatError 格式化错误消息
func (g *Git) formatError(operation string, err error, output string) error {
	msg := fmt.Sprintf("Git %s failed: %v", operation, err)

	if output != "" {
		msg += "\nOutput: " + output
	}

	// 提供友好的建议
	suggestions := g.getSuggestions(operation, output)
	if suggestions != "" {
		msg += "\n\nSuggestions:\n" + suggestions
	}

	return fmt.Errorf("%s", msg)
}

// getSuggestions 根据错误输出提供建议
func (g *Git) getSuggestions(operation string, output string) string {
	output = strings.ToLower(output)

	if strings.Contains(output, "could not resolve host") || strings.Contains(output, "network") {
		return "1. Check your internet connection\n" +
			"2. Verify the repository URL\n" +
			"3. Check if the Git server is accessible"
	}

	if strings.Contains(output, "authentication failed") || strings.Contains(output, "permission denied") {
		return "1. Check your Git credentials\n" +
			"2. Verify SSH key is configured\n" +
			"3. Use HTTPS with personal access token"
	}

	if strings.Contains(output, "conflict") {
		return "1. Resolve conflicts manually\n" +
			"2. Use 'git status' to see conflicted files\n" +
			"3. After resolving, use 'git add' and 'git commit'"
	}

	if strings.Contains(output, "not a git repository") {
		return "1. Check if the path is correct\n" +
			"2. Initialize a Git repository with 'git init'\n" +
			"3. Clone an existing repository"
	}

	if strings.Contains(output, "already exists") {
		return "1. Use a different path or name\n" +
			"2. Remove the existing file/directory\n" +
			"3. Use --force flag if you want to overwrite (dangerous)"
	}

	return ""
}
