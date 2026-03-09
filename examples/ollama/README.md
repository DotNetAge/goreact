# Ollama Integration Example

演示如何集成真实的 LLM 服务（Ollama）。

## 前置条件

1. 安装 Ollama：
```bash
# macOS/Linux
curl -fsSL https://ollama.com/install.sh | sh

# 或访问 https://ollama.com 下载
```

2. 启动 Ollama 服务：
```bash
ollama serve
```

3. 拉取模型：
```bash
# 推荐使用轻量级模型
ollama pull qwen2.5:0.5b

# 或其他模型
ollama pull llama3.2:1b
```

## 运行

```bash
go run main.go
```

## 代码说明

这个示例展示了：

1. **创建 Ollama 客户端**：
```go
llmClient := ollama.NewOllamaClient(
    ollama.WithModel("qwen2.5:0.5b"),
    ollama.WithTemperature(0.7),
    ollama.WithBaseURL("http://localhost:11434"),
)
```

2. **配置 Engine**：
```go
eng := engine.New(
    engine.WithLLMClient(llmClient),
    engine.WithMaxIterations(10),
)
```

3. **执行真实推理**：LLM 会真正理解任务并决定使用哪个工具

## 配置选项

- `WithModel()` - 指定模型名称
- `WithTemperature()` - 控制输出随机性（0.0-1.0）
- `WithBaseURL()` - Ollama 服务地址
- `WithTimeout()` - 请求超时时间

## 推荐模型

| 模型 | 大小 | 速度 | 适用场景 |
|------|------|------|----------|
| qwen2.5:0.5b | 0.5GB | 极快 | 开发测试 |
| llama3.2:1b | 1GB | 快 | 简单任务 |
| qwen2.5:3b | 3GB | 中等 | 生产环境 |

## 下一步

- 查看 [with_cache](../with_cache/) 示例了解如何优化性能
- 查看 [thinker_middleware](../thinker_middleware/) 示例了解高级功能
