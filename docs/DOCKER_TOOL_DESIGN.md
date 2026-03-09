# Docker 工具设计文档

## 概述

Docker 是 DevOps 场景中最核心的工具，用于容器化应用的开发、部署和管理。我们需要提供一个 Schema-based 的 Docker 工具，让 LLM 能够轻松地执行 Docker 操作。

---

## 核心痛点

### 痛点 1：命令复杂，参数众多

**问题：**
```bash
# 复杂的 docker run 命令
docker run -d --name myapp -p 8080:80 -v /data:/app/data -e ENV=prod --restart=always nginx:latest

# 容易出错的参数
docker run -p 80:8080  # 端口映射顺序错误
docker run -v data:/app  # 卷挂载路径错误
```

**解决方案：**
- Schema 验证参数
- 提供友好的参数名称
- 自动补全默认值

### 痛点 2：错误消息不友好

**问题：**
```
Error response from daemon: driver failed programming external connectivity on endpoint myapp:
Bind for 0.0.0.0:8080 failed: port is already allocated
```

**解决方案：**
```
❌ Failed to start container
Reason: Port 8080 is already in use
Suggestions:
1. Check running containers: docker ps
2. Use a different port
3. Stop the container using port 8080
```

### 痛点 3：危险操作缺少保护

**问题：**
- `docker rm -f $(docker ps -aq)` 删除所有容器
- `docker system prune -a` 删除所有未使用的资源
- `docker rmi -f` 强制删除镜像

**解决方案：**
- 危险操作需要确认
- 提供安全模式
- 记录所有操作

---

## 功能设计

### 1. 容器管理

#### 1.1 Run（运行容器）

```go
docker.Run(image, options)
```

**参数：**
- `image` (string, required) - 镜像名称
- `name` (string, optional) - 容器名称
- `ports` ([]string, optional) - 端口映射 ["8080:80"]
- `volumes` ([]string, optional) - 卷挂载 ["/data:/app/data"]
- `env` (map, optional) - 环境变量 {"ENV": "prod"}
- `detach` (bool, optional) - 后台运行，默认 true
- `restart` (string, optional) - 重启策略：no, always, on-failure
- `network` (string, optional) - 网络名称

**示例：**
```json
{
  "operation": "run",
  "image": "nginx:latest",
  "name": "my-nginx",
  "ports": ["8080:80"],
  "volumes": ["/data:/usr/share/nginx/html"],
  "env": {"ENV": "production"},
  "detach": true,
  "restart": "always"
}
```

#### 1.2 Ps（列出容器）

```go
docker.Ps(options)
```

**参数：**
- `all` (bool, optional) - 显示所有容器（包括停止的）
- `filter` (map, optional) - 过滤条件 {"status": "running"}

**返回：**
```json
{
  "containers": [
    {
      "id": "abc123",
      "name": "my-nginx",
      "image": "nginx:latest",
      "status": "Up 2 hours",
      "ports": ["0.0.0.0:8080->80/tcp"]
    }
  ]
}
```

#### 1.3 Stop/Start/Restart（容器控制）

```go
docker.Stop(container)
docker.Start(container)
docker.Restart(container)
```

**参数：**
- `container` (string, required) - 容器 ID 或名称

#### 1.4 Rm（删除容器）

```go
docker.Rm(container, options)
```

**参数：**
- `container` (string, required) - 容器 ID 或名称
- `force` (bool, optional) - 强制删除运行中的容器

#### 1.5 Logs（查看日志）

```go
docker.Logs(container, options)
```

**参数：**
- `container` (string, required) - 容器 ID 或名称
- `tail` (int, optional) - 显示最后 N 行，默认 100
- `follow` (bool, optional) - 持续输出日志
- `since` (string, optional) - 显示指定时间之后的日志

#### 1.6 Exec（在容器中执行命令）

```go
docker.Exec(container, command, options)
```

**参数：**
- `container` (string, required) - 容器 ID 或名称
- `command` (string, required) - 要执行的命令
- `interactive` (bool, optional) - 交互模式
- `tty` (bool, optional) - 分配伪终端

#### 1.7 Inspect（查看容器详情）

```go
docker.Inspect(container)
```

**参数：**
- `container` (string, required) - 容器 ID 或名称

#### 1.8 Stats（查看资源使用）

```go
docker.Stats(container, options)
```

**参数：**
- `container` (string, optional) - 容器 ID 或名称，不指定则显示所有
- `no_stream` (bool, optional) - 只显示一次，不持续更新

---

### 2. 镜像管理

#### 2.1 Images（列出镜像）

```go
docker.Images(options)
```

**参数：**
- `all` (bool, optional) - 显示所有镜像（包括中间层）
- `filter` (map, optional) - 过滤条件

**返回：**
```json
{
  "images": [
    {
      "id": "abc123",
      "repository": "nginx",
      "tag": "latest",
      "size": "142MB",
      "created": "2 weeks ago"
    }
  ]
}
```

#### 2.2 Pull（拉取镜像）

```go
docker.Pull(image, options)
```

**参数：**
- `image` (string, required) - 镜像名称
- `tag` (string, optional) - 标签，默认 "latest"

#### 2.3 Push（推送镜像）

```go
docker.Push(image, options)
```

**参数：**
- `image` (string, required) - 镜像名称
- `tag` (string, optional) - 标签

#### 2.4 Build（构建镜像）

```go
docker.Build(path, options)
```

**参数：**
- `path` (string, required) - Dockerfile 所在路径
- `tag` (string, required) - 镜像标签
- `file` (string, optional) - Dockerfile 文件名，默认 "Dockerfile"
- `build_args` (map, optional) - 构建参数

#### 2.5 Rmi（删除镜像）

```go
docker.Rmi(image, options)
```

**参数：**
- `image` (string, required) - 镜像 ID 或名称
- `force` (bool, optional) - 强制删除

#### 2.6 Tag（标记镜像）

```go
docker.Tag(source, target)
```

**参数：**
- `source` (string, required) - 源镜像
- `target` (string, required) - 目标镜像

---

### 3. 网络管理

#### 3.1 Network Ls（列出网络）

```go
docker.NetworkLs()
```

#### 3.2 Network Create（创建网络）

```go
docker.NetworkCreate(name, options)
```

**参数：**
- `name` (string, required) - 网络名称
- `driver` (string, optional) - 驱动类型：bridge, overlay, host

#### 3.3 Network Rm（删除网络）

```go
docker.NetworkRm(name)
```

---

### 4. 卷管理

#### 4.1 Volume Ls（列出卷）

```go
docker.VolumeLs()
```

#### 4.2 Volume Create（创建卷）

```go
docker.VolumeCreate(name, options)
```

**参数：**
- `name` (string, required) - 卷名称
- `driver` (string, optional) - 驱动类型

#### 4.3 Volume Rm（删除卷）

```go
docker.VolumeRm(name)
```

---

## Schema 定义

```go
var dockerSchema = schema.Define(
    // 操作类型
    schema.Param("operation", schema.String, "Docker operation").
        Enum(
            // 容器管理
            "run", "ps", "stop", "start", "restart", "rm",
            "logs", "exec", "inspect", "stats",
            // 镜像管理
            "images", "pull", "push", "build", "rmi", "tag",
            // 网络管理
            "network-ls", "network-create", "network-rm",
            // 卷管理
            "volume-ls", "volume-create", "volume-rm",
        ).
        Required(),

    // 容器/镜像标识
    schema.Param("container", schema.String, "Container ID or name").
        RequiredIf("operation", "stop", "start", "restart", "rm", "logs", "exec", "inspect"),

    schema.Param("image", schema.String, "Image name").
        RequiredIf("operation", "run", "pull", "push", "rmi"),

    // Run 参数
    schema.Param("name", schema.String, "Container name"),
    schema.Param("ports", schema.Array, "Port mappings"),
    schema.Param("volumes", schema.Array, "Volume mounts"),
    schema.Param("env", schema.Object, "Environment variables"),
    schema.Param("detach", schema.Boolean, "Run in background").Default(true),
    schema.Param("restart", schema.String, "Restart policy").
        Enum("no", "always", "on-failure", "unless-stopped"),

    // 通用选项
    schema.Param("force", schema.Boolean, "Force operation").Default(false),
)
```

---

## 安全控制

### 1. 危险操作列表

```go
var dangerousOperations = map[string]bool{
    "rm -f":           true,
    "rmi -f":          true,
    "system prune -a": true,
    "volume rm":       true,
}
```

### 2. 资源限制

```go
type DockerTool struct {
    maxContainers int  // 最大容器数量
    maxMemory     int  // 最大内存限制
    maxCPU        int  // 最大 CPU 限制
}
```

---

## 错误处理

### 1. 端口占用

```go
if strings.Contains(err.Error(), "port is already allocated") {
    return schema.NewUserError(
        "Port is already in use\n"+
        "Suggestions:\n"+
        "1. Check running containers: docker ps\n"+
        "2. Use a different port\n"+
        "3. Stop the container using this port",
    )
}
```

### 2. 镜像不存在

```go
if strings.Contains(err.Error(), "No such image") {
    return schema.NewUserError(
        "Image not found\n"+
        "Suggestions:\n"+
        "1. Pull the image: docker pull <image>\n"+
        "2. Check image name and tag\n"+
        "3. List available images: docker images",
    )
}
```

### 3. 容器不存在

```go
if strings.Contains(err.Error(), "No such container") {
    return schema.NewUserError(
        "Container not found\n"+
        "Suggestions:\n"+
        "1. List running containers: docker ps\n"+
        "2. List all containers: docker ps -a\n"+
        "3. Check container name or ID",
    )
}
```

---

## 实现优先级

### P0（立即实现）
- ✅ run, ps, stop, start, restart, rm
- ✅ logs, exec
- ✅ images, pull

### P1（短期）
- push, build, rmi, tag
- inspect, stats

### P2（中期）
- network-ls, network-create, network-rm
- volume-ls, volume-create, volume-rm

---

## 总结

### 核心价值
1. **Schema 验证** - 自动验证参数，减少错误
2. **友好错误** - LLM 能理解的错误消息
3. **安全控制** - 阻止危险操作
4. **高频操作** - 覆盖 90% 的日常使用场景

### 记住
> Docker 是 DevOps 核心工具，必须做到：
> - 参数验证严格
> - 错误消息友好
> - 安全控制完善
> - 资源限制合理
