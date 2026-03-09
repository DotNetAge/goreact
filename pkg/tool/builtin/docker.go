package builtin

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Docker Docker 容器管理工具
type Docker struct{}

// NewDocker 创建 Docker 工具
func NewDocker() *Docker {
	return &Docker{}
}

// Name 返回工具名称
func (d *Docker) Name() string {
	return "docker"
}

// Description 返回工具描述
func (d *Docker) Description() string {
	return "Docker container management: run, ps, stop, start, restart, rm, logs, exec, images, pull, push, build"
}

// Execute 执行 Docker 操作
func (d *Docker) Execute(params map[string]any) (any, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'operation' parameter")
	}

	switch operation {
	// 容器管理
	case "run":
		return d.run(params)
	case "ps":
		return d.ps(params)
	case "stop":
		return d.stop(params)
	case "start":
		return d.start(params)
	case "restart":
		return d.restart(params)
	case "rm":
		return d.rm(params)
	case "logs":
		return d.logs(params)
	case "exec":
		return d.exec(params)
	case "inspect":
		return d.inspect(params)
	case "stats":
		return d.stats(params)

	// 镜像管理
	case "images":
		return d.images(params)
	case "pull":
		return d.pull(params)
	case "push":
		return d.push(params)
	case "build":
		return d.build(params)
	case "rmi":
		return d.rmi(params)
	case "tag":
		return d.tag(params)

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// run 运行容器
func (d *Docker) run(params map[string]any) (any, error) {
	image, ok := params["image"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'image' parameter")
	}

	args := []string{"run"}

	// 后台运行
	detach := true
	if v, ok := params["detach"].(bool); ok {
		detach = v
	}
	if detach {
		args = append(args, "-d")
	}

	// 容器名称
	if name, ok := params["name"].(string); ok && name != "" {
		args = append(args, "--name", name)
	}

	// 端口映射
	if ports, ok := params["ports"].([]any); ok {
		for _, p := range ports {
			if port, ok := p.(string); ok {
				args = append(args, "-p", port)
			}
		}
	}

	// 卷挂载
	if volumes, ok := params["volumes"].([]any); ok {
		for _, v := range volumes {
			if vol, ok := v.(string); ok {
				args = append(args, "-v", vol)
			}
		}
	}

	// 环境变量
	if env, ok := params["env"].(map[string]any); ok {
		for k, v := range env {
			args = append(args, "-e", fmt.Sprintf("%s=%v", k, v))
		}
	}

	// 重启策略
	if restart, ok := params["restart"].(string); ok && restart != "" {
		args = append(args, "--restart", restart)
	}

	// 网络
	if network, ok := params["network"].(string); ok && network != "" {
		args = append(args, "--network", network)
	}

	args = append(args, image)

	// 容器命令
	if cmd, ok := params["command"].(string); ok && cmd != "" {
		args = append(args, strings.Fields(cmd)...)
	}

	output, err := d.runDockerCommand(args...)
	if err != nil {
		return nil, d.formatError("run", err, output)
	}

	return map[string]any{
		"success":      true,
		"message":      "Container started successfully",
		"container_id": strings.TrimSpace(output),
	}, nil
}

// ps 列出容器
func (d *Docker) ps(params map[string]any) (any, error) {
	args := []string{"ps"}

	// 显示所有容器
	if all, ok := params["all"].(bool); ok && all {
		args = append(args, "-a")
	}

	// 格式化输出
	args = append(args, "--format", "{{.ID}}|{{.Names}}|{{.Image}}|{{.Status}}|{{.Ports}}")

	output, err := d.runDockerCommand(args...)
	if err != nil {
		return nil, d.formatError("ps", err, output)
	}

	return map[string]any{
		"success": true,
		"output":  output,
	}, nil
}

// stop 停止容器
func (d *Docker) stop(params map[string]any) (any, error) {
	container, ok := params["container"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'container' parameter")
	}

	output, err := d.runDockerCommand("stop", container)
	if err != nil {
		return nil, d.formatError("stop", err, output)
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Container '%s' stopped", container),
	}, nil
}

// start 启动容器
func (d *Docker) start(params map[string]any) (any, error) {
	container, ok := params["container"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'container' parameter")
	}

	output, err := d.runDockerCommand("start", container)
	if err != nil {
		return nil, d.formatError("start", err, output)
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Container '%s' started", container),
	}, nil
}

// restart 重启容器
func (d *Docker) restart(params map[string]any) (any, error) {
	container, ok := params["container"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'container' parameter")
	}

	output, err := d.runDockerCommand("restart", container)
	if err != nil {
		return nil, d.formatError("restart", err, output)
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Container '%s' restarted", container),
	}, nil
}

// rm 删除容器
func (d *Docker) rm(params map[string]any) (any, error) {
	container, ok := params["container"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'container' parameter")
	}

	args := []string{"rm"}

	// 强制删除
	if force, ok := params["force"].(bool); ok && force {
		args = append(args, "-f")
	}

	args = append(args, container)

	output, err := d.runDockerCommand(args...)
	if err != nil {
		return nil, d.formatError("rm", err, output)
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Container '%s' removed", container),
	}, nil
}

// logs 查看容器日志
func (d *Docker) logs(params map[string]any) (any, error) {
	container, ok := params["container"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'container' parameter")
	}

	args := []string{"logs"}

	// 显示最后 N 行
	tail := 100
	if t, ok := params["tail"].(float64); ok && t > 0 {
		tail = int(t)
	}
	args = append(args, "--tail", fmt.Sprintf("%d", tail))

	// 显示时间戳
	args = append(args, "--timestamps")

	args = append(args, container)

	output, err := d.runDockerCommand(args...)
	if err != nil {
		return nil, d.formatError("logs", err, output)
	}

	return map[string]any{
		"success": true,
		"output":  output,
	}, nil
}

// exec 在容器中执行命令
func (d *Docker) exec(params map[string]any) (any, error) {
	container, ok := params["container"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'container' parameter")
	}

	command, ok := params["command"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'command' parameter")
	}

	args := []string{"exec"}

	// 交互模式
	if interactive, ok := params["interactive"].(bool); ok && interactive {
		args = append(args, "-i")
	}

	// 分配伪终端
	if tty, ok := params["tty"].(bool); ok && tty {
		args = append(args, "-t")
	}

	args = append(args, container)
	args = append(args, strings.Fields(command)...)

	output, err := d.runDockerCommand(args...)
	if err != nil {
		return nil, d.formatError("exec", err, output)
	}

	return map[string]any{
		"success": true,
		"output":  output,
	}, nil
}

// inspect 查看容器详情
func (d *Docker) inspect(params map[string]any) (any, error) {
	container, ok := params["container"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'container' parameter")
	}

	output, err := d.runDockerCommand("inspect", container)
	if err != nil {
		return nil, d.formatError("inspect", err, output)
	}

	return map[string]any{
		"success": true,
		"output":  output,
	}, nil
}

// stats 查看资源使用
func (d *Docker) stats(params map[string]any) (any, error) {
	args := []string{"stats", "--no-stream"}

	// 指定容器
	if container, ok := params["container"].(string); ok && container != "" {
		args = append(args, container)
	}

	output, err := d.runDockerCommand(args...)
	if err != nil {
		return nil, d.formatError("stats", err, output)
	}

	return map[string]any{
		"success": true,
		"output":  output,
	}, nil
}

// images 列出镜像
func (d *Docker) images(params map[string]any) (any, error) {
	args := []string{"images"}

	// 显示所有镜像
	if all, ok := params["all"].(bool); ok && all {
		args = append(args, "-a")
	}

	// 格式化输出
	args = append(args, "--format", "{{.ID}}|{{.Repository}}|{{.Tag}}|{{.Size}}|{{.CreatedSince}}")

	output, err := d.runDockerCommand(args...)
	if err != nil {
		return nil, d.formatError("images", err, output)
	}

	return map[string]any{
		"success": true,
		"output":  output,
	}, nil
}

// pull 拉取镜像
func (d *Docker) pull(params map[string]any) (any, error) {
	image, ok := params["image"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'image' parameter")
	}

	// 标签
	if tag, ok := params["tag"].(string); ok && tag != "" {
		image = image + ":" + tag
	}

	output, err := d.runDockerCommand("pull", image)
	if err != nil {
		return nil, d.formatError("pull", err, output)
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Image '%s' pulled successfully", image),
		"output":  output,
	}, nil
}

// push 推送镜像
func (d *Docker) push(params map[string]any) (any, error) {
	image, ok := params["image"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'image' parameter")
	}

	// 标签
	if tag, ok := params["tag"].(string); ok && tag != "" {
		image = image + ":" + tag
	}

	output, err := d.runDockerCommand("push", image)
	if err != nil {
		return nil, d.formatError("push", err, output)
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Image '%s' pushed successfully", image),
		"output":  output,
	}, nil
}

// build 构建镜像
func (d *Docker) build(params map[string]any) (any, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'path' parameter")
	}

	tag, ok := params["tag"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'tag' parameter")
	}

	args := []string{"build", "-t", tag}

	// Dockerfile 文件名
	if file, ok := params["file"].(string); ok && file != "" {
		args = append(args, "-f", file)
	}

	// 构建参数
	if buildArgs, ok := params["build_args"].(map[string]any); ok {
		for k, v := range buildArgs {
			args = append(args, "--build-arg", fmt.Sprintf("%s=%v", k, v))
		}
	}

	args = append(args, path)

	output, err := d.runDockerCommand(args...)
	if err != nil {
		return nil, d.formatError("build", err, output)
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Image '%s' built successfully", tag),
		"output":  output,
	}, nil
}

// rmi 删除镜像
func (d *Docker) rmi(params map[string]any) (any, error) {
	image, ok := params["image"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'image' parameter")
	}

	args := []string{"rmi"}

	// 强制删除
	if force, ok := params["force"].(bool); ok && force {
		args = append(args, "-f")
	}

	args = append(args, image)

	output, err := d.runDockerCommand(args...)
	if err != nil {
		return nil, d.formatError("rmi", err, output)
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Image '%s' removed", image),
	}, nil
}

// tag 标记镜像
func (d *Docker) tag(params map[string]any) (any, error) {
	source, ok := params["source"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'source' parameter")
	}

	target, ok := params["target"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'target' parameter")
	}

	output, err := d.runDockerCommand("tag", source, target)
	if err != nil {
		return nil, d.formatError("tag", err, output)
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Image tagged: %s -> %s", source, target),
	}, nil
}

// runDockerCommand 执行 Docker 命令
func (d *Docker) runDockerCommand(args ...string) (string, error) {
	cmd := exec.Command("docker", args...)

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
func (d *Docker) formatError(operation string, err error, output string) error {
	msg := fmt.Sprintf("Docker %s failed: %v", operation, err)

	if output != "" {
		msg += "\nOutput: " + output
	}

	// 提供友好的建议
	suggestions := d.getSuggestions(operation, output)
	if suggestions != "" {
		msg += "\n\nSuggestions:\n" + suggestions
	}

	return fmt.Errorf("%s", msg)
}

// getSuggestions 根据错误输出提供建议
func (d *Docker) getSuggestions(operation string, output string) string {
	output = strings.ToLower(output)

	if strings.Contains(output, "port is already allocated") {
		return "1. Check running containers: docker ps\n" +
			"2. Use a different port\n" +
			"3. Stop the container using this port"
	}

	if strings.Contains(output, "no such image") {
		return "1. Pull the image: docker pull <image>\n" +
			"2. Check image name and tag\n" +
			"3. List available images: docker images"
	}

	if strings.Contains(output, "no such container") {
		return "1. List running containers: docker ps\n" +
			"2. List all containers: docker ps -a\n" +
			"3. Check container name or ID"
	}

	if strings.Contains(output, "cannot connect to the docker daemon") {
		return "1. Start Docker daemon\n" +
			"2. Check Docker service status\n" +
			"3. Verify Docker is installed"
	}

	if strings.Contains(output, "permission denied") {
		return "1. Run with sudo (not recommended)\n" +
			"2. Add user to docker group: sudo usermod -aG docker $USER\n" +
			"3. Restart your session"
	}

	if strings.Contains(output, "conflict") || strings.Contains(output, "already in use") {
		return "1. Use a different name\n" +
			"2. Remove the existing container/image\n" +
			"3. Use --force flag if appropriate"
	}

	return ""
}
