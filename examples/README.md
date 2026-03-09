# GoReAct Examples

这里包含了 GoReAct 框架的各种使用示例，从基础到高级，帮助你快速上手。

## 📚 示例列表

### 🚀 入门示例

#### [simple](./simple/) - 最简单的示例
- **难度**：⭐
- **用时**：2 分钟
- **特性**：Mock LLM、Echo 工具
- **适合**：第一次使用 GoReAct

```bash
go run examples/simple/main.go
```

#### [calculator](./calculator/) - 计算器示例
- **难度**：⭐
- **用时**：3 分钟
- **特性**：Calculator 工具、完整 ReAct 循环
- **适合**：了解工具系统

```bash
go run examples/calculator/main.go
```

---

### 🔌 LLM 集成

#### [ollama](./ollama/) - Ollama 集成
- **难度**：⭐⭐
- **用时**：5 分钟
- **特性**：真实 LLM、本地部署
- **前置**：需要安装 Ollama
- **适合**：生产环境使用

```bash
# 1. 启动 Ollama
ollama serve

# 2. 拉取模型
ollama pull qwen2.5:0.5b

# 3. 运行示例
go run examples/ollama/main.go
```

---

### ⚡ 性能优化

#### [with_cache](./with_cache/) - 缓存系统
- **难度**：⭐⭐
- **用时**：5 分钟
- **特性**：内存缓存、TTL、性能对比
- **效果**：~900,000x 性能提升
- **适合**：优化重复任务

```bash
go run examples/with_cache/main.go
```

---

### 🎯 高级特性

#### [thinker_middleware](./thinker_middleware/) - 中间件系统 ⭐ NEW
- **难度**：⭐⭐⭐
- **用时**：10 分钟
- **特性**：
  - 日志记录
  - 自动重试
  - 意图缓存（语义相似度）
  - RAG 知识增强
  - 用户画像
  - 置信度评估
  - 速率限制
- **适合**：构建生产级应用

```bash
go run examples/thinker_middleware/main.go
```

#### [multi_agent](./multi_agent/) - 多智能体协作
- **难度**：⭐⭐⭐
- **用时**：10 分钟
- **特性**：多 Agent 协调、任务分解
- **适合**：复杂任务处理

```bash
go run examples/multi_agent/main.go
```

---

## 🗺️ 学习路径

### 路径 1：快速入门（15 分钟）
```
simple → calculator → ollama
```
适合：想快速了解 GoReAct 基本功能

### 路径 2：性能优化（20 分钟）
```
ollama → with_cache → thinker_middleware
```
适合：关注性能和生产环境部署

### 路径 3：高级功能（30 分钟）
```
calculator → ollama → thinker_middleware → multi_agent
```
适合：深入学习框架的高级特性

---

## 📖 相关文档

- [QUICK_START.md](../docs/QUICK_START.md) - 5 分钟快速开始
- [THINKER_GUIDE.md](../docs/THINKER_GUIDE.md) - Thinker 组件详解
- [MIDDLEWARE_GUIDE.md](../docs/MIDDLEWARE_GUIDE.md) - 中间件系统指南
- [README.md](../README.md) - 主文档

---

## 🎓 按功能查找示例

| 功能 | 示例 |
|------|------|
| Mock LLM 测试 | [simple](./simple/), [calculator](./calculator/) |
| 真实 LLM 集成 | [ollama](./ollama/) |
| 缓存优化 | [with_cache](./with_cache/), [thinker_middleware](./thinker_middleware/) |
| 日志记录 | [thinker_middleware](./thinker_middleware/) |
| 错误重试 | [thinker_middleware](./thinker_middleware/) |
| RAG 增强 | [thinker_middleware](./thinker_middleware/) |
| 速率限制 | [thinker_middleware](./thinker_middleware/) |
| 多智能体 | [multi_agent](./multi_agent/) |

---

## 💡 常见问题

### Q: 我应该从哪个示例开始？
A: 从 [simple](./simple/) 开始，然后按照学习路径 1 进行。

### Q: 如何在生产环境使用？
A: 参考 [ollama](./ollama/) + [with_cache](./with_cache/) + [thinker_middleware](./thinker_middleware/) 的组合。

### Q: 如何自定义中间件？
A: 查看 [thinker_middleware](./thinker_middleware/) 示例中的自定义中间件部分。

### Q: 性能如何优化？
A: 使用缓存（[with_cache](./with_cache/)）和中间件（[thinker_middleware](./thinker_middleware/)）。

---

## 🤝 贡献示例

欢迎贡献新的示例！请确保：
1. 代码简洁易懂
2. 包含独立的 README.md
3. 有清晰的注释
4. 可以独立运行

---

## 📝 示例模板

创建新示例时，可以参考这个结构：

```
examples/your_example/
├── README.md          # 示例说明
├── main.go            # 主程序
└── go.mod (optional)  # 如果有特殊依赖
```

README.md 应包含：
- 功能说明
- 运行方法
- 代码说明
- 适用场景
- 相关文档链接
