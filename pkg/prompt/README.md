# Prompt Toolkit (提示词工程工具箱)

`pkg/prompt` 是 GoReAct 框架中负责提示词构建、格式化与生命周期管理的底层核心组件。它旨在解决 LLM 应用开发中常见的“提示词碎片化”、“Token 溢出”以及“结构化输出不稳定”等痛点。

---

## 1. 核心设计理由 (Design Rationale)

在复杂的 ReAct 循环中，提示词（Prompt）不仅仅是几行文字，它实际上是 Agent 的**运行时配置与约束协议**。设计此包的初衷在于：

- **Context Efficiency (上下文效率)**：大模型的上下文窗口是昂贵的。我们需要精细化控制哪些信息进入上下文（通过 `compression`），以及如何以最节省 Token 的方式呈现工具描述（通过 `formatter`）。
- **Precision & Stability (精度与稳定性)**：通过结构化的 `builder` 和 `Few-Shot` 注入，强制模型遵循特定的推理范式（如 Thought/Action/Observation），降低格式解析失败的概率。
- **Observability (可观测性)**：提示词构建不应是一个“黑盒”。通过 `debug` 和 `counter` 模块，开发者可以实时审计每一轮对话消耗的 Token 构成，从而进行针对性优化。
- **Decoupling (解耦)**：将提示词模板（Templates）、格式化逻辑（Formatters）与业务逻辑（Thinker）分离，使得同一套推理逻辑可以轻松适配不同的模型（如 GPT-4 vs. Claude 3）。

---

## 2. 模块组成 (Package Structure)

### 🚀 [Builder](./builder/) (流式构建器)
提供 Fluent API（流式接口）来组装复杂的 Prompt。支持：
- 系统模板与用户模板的分离渲染。
- 自动聚合 `History`（对话历史）、`Tools`（工具集）和 `Few-Shots`（示例）。
- 动态变量注入（如记忆体召回内容）。

### 🛠️ [Formatter](./formatter/) (格式化器)
负责将结构化的工具定义（Tool Schema）转换为模型最易理解的格式：
- **JSONSchema**: 适合高级模型，参数约束极度精准。
- **Markdown**: 适合阅读，适合具备强文档理解能力的模型。
- **Compact**: 极度节省 Token 的紧凑格式。

### 📉 [Compression](./compression/) (压缩策略)
当上下文超过模型限制或设定的 `maxTokens` 时，提供多种“救生圈”策略：
- **SlidingWindow**: 保留最近的 N 轮对话。
- **Priority-based**: 根据消息角色（如优先保留 System 和最近的 User 指令）进行选择性保留。
- **Hybrid**: 组合多种策略直至满足 Token 约束。

### 🔢 [Counter](./counter/) (Token 计数)
提供不同精度的计数器：
- **SimpleEstimator**: 快速估算（1 token ≈ 4 chars）。
- **UniversalEstimator**: 支持中英混合的启发式精确计数。
- **CachedCounter**: 引入缓存，加速高频 Prompt 的长度计算。

### 🐞 [Debug](./debug/) (审计与调试)
用于追踪提示词的构建细节。它可以输出详细的 **Token 使用报告**，展示 System、User、Tools、History 分别占用了多少百分比，帮助定位“上下文泄露”问题。

---

## 3. 标准工作流 (Typical Workflow)

1. **定义模板**：预设包含 `{{.tools}}` 和 `{{.history}}` 占位符的模板。
2. **收集上下文**：从 Pipeline 或 Memory 中召回相关数据。
3. **流式组装**：
   ```go
   prompt := builder.New().
       WithSystemTemplate(ReActTemplate).
       WithTools(availableTools).
       WithHistory(shortTermHistory).
       WithVariable("memories", recalledData).
       Build()
   ```
4. **Token 审计**：在发送 LLM 前，通过 `counter` 校验长度并记录。
5. **LLM 调用**：将生成的 `System` 和 `User` 提示词传递给 LLM Client。

---

## 4. 与 Thinker 的关系

`Thinker` 是“大脑”，而 `Prompt Toolkit` 是大脑的“语言中枢”。Thinker 负责决定要说什么（决策逻辑），而 Prompt Toolkit 负责如何把这些话组织得最得体、最专业、最省钱。
