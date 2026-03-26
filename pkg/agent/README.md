# Agent & Model 设计与应用层抽象

本文档详细阐述了 `pkg/agent` 与 `pkg/model` 两个功能包的设计哲学与架构定位。

## 1. 核心理念：技术引擎与应用业务的分层

在 GoReAct 架构中，为了支持系统化的扩展和降低最终用户的接入成本，我们将框架划分为**技术引擎层 (Engine Layer)** 和 **应用业务层 (Application Layer)**。

*   **技术引擎层 (`engine.Reactor`)**：面向框架开发者。它是一个纯粹的 State Machine（状态机），关注于 `Think -> Act -> Observe -> Terminate` 的微观流转。它需要被精细地注入各种组件（Thinker、Actor、Client、Memory 等）。
*   **应用业务层 (`agent.Agent` & `model.Model`)**：面向终端应用开发者。应用开发者不需要了解底层的 Pipeline 如何运转，他们只需要定义“角色（Agent）”和“大脑（Model）”。

### 1.1 从数据到运行时的蜕变
`Agent` 和 `Model` 在其最原始的形态下是**数据（配置）实体**，它们非常适合被序列化为 YAML 或存储在数据库中：
*   **Model (配置)**：定义了 `Provider`, `ModelID`, `APIKey` 等。
*   **Agent (配置)**：定义了 `Name`, `System Prompt`, 引用的 `ModelName` 以及绑定的 `Tools`。

**最优雅的闭环设计在于：**
应用层通过统一的 `Manager` 获取一个 Agent 时，获取的不再是一个死的数据结构，而是一个**已经被工厂自动装配好的、内部封装了 `engine.Reactor` 的高阶可执行实例**。

---

## 2. Model 模块 (`pkg/model`)

Model 模块负责统一管理系统内所有可用的大语言模型（LLM）配置。

### 2.1 Model 实体
`Model` 结构体是具体 LLM 的配置定义，包含：
- **基础路由**：`Provider` (如 openai, anthropic, ollama)、`ModelID`、`BaseURL`。
- **认证与参数**：`APIKey`、`Temperature`、`MaxTokens`、`Timeout`。
- **能力标识 (Features)**：通过 `ModelFeatures` 明确标识该模型是否支持视觉、工具调用、流式输出等。

### 2.2 Model Manager 与工厂模式
`model.Manager` 是一个全局注册中心，充当**工厂 (Factory)** 角色：
应用层调用 `Manager.CreateLLMClient(modelName)`，Manager 会根据对应的 `Model` 配置，动态初始化并返回一个实现了 `gochatcore.Client` 接口的真实网络客户端实体。

---

## 3. Agent 模块 (`pkg/agent`)：应用层的真正入口

Agent 是对 `Reactor` 的业务级封装，是最终用户与框架交互的**唯一高级入口**。

### 3.1 Agent 实例的完整形态
在应用层看来，一个 Agent 实例内部已经包含了：
1. **System Prompt (SOP)**：角色与行为规程。
2. **LLM Client**：通过 `ModelName` 从 `model.Manager` 处兑换来的真实大脑。
3. **Reactor Engine**：内部私有的执行引擎。

**用户交互心智的转变：**
用户不再需要手动组装 Thinker、Actor，也不用直接调用 `Reactor.Run()`。用户只需要：
```go
// 1. 从 Manager 获取一个组装好的 Agent
myAgent := agentManager.Select("DataAnalyst")

// 2. 直接与 Agent 对话 (支持纯文本或多模态)
response, err := myAgent.Chat(context.Background(), "帮我分析这组数据...")

// 或者带有文件附件的多模态对话
response, err := myAgent.ChatWithFiles(context.Background(), "分析这个报表", []string{"./report.pdf"})
```

### 3.2 Agent Manager (RAG Agent Manager) 与发现机制
`agent.Manager` 绝不仅仅是一个简单的 Map 缓存，它在宏大架构中扮演着 **RAG Agent Manager** 的关键角色。
当系统面临一个未知或复杂任务，且注册了海量的 Agent 技能库时，Manager 可以利用基于大模型的语义匹配（Semantic Search）甚至向量检索，像 RAG（检索增强生成）一样，根据任务描述的上下文，从庞大的 Agent 库中动态“检索”出最适合处理该任务的 Agent 实例进行注入和调用。

---

## 4. 模式驱动编排 (Pattern-Driven Orchestration)

GoReAct 弃用了不切实际的“智能体即工具 (AAAT)”嵌套模式，转而支持更稳健的编排模式：

- **Master-Sub (主从编排)**: `Agent` 内部可作为 Master 调用其他子组件或协同者，通过显式的任务分发保证逻辑可控。
- **Evolution (动作编译)**: `Agent` 会根据历史成功的推理轨迹生成 `CompiledAction`（肌肉记忆），通过 `EvolutionPipeline` 实现极速执行。
- **语义化召回**: 通过 GoRAG 对接，Agent 在执行过程中能够根据当前上下文语义化地召回最匹配的工具与技能，而非硬编码。

---

## 5. 当前代码库落实状态 (Phase 4 已闭环)

**Phase 4 已经完成了向此设计范式的核心重构：**

1. **AgentBuilder**：已实现完整的装配工厂逻辑，能够自动向 `Agent` 内部注入 `Reactor`、`MemoryBank` 与 `SkillManager`。
2. **Evolution Pipeline**：已实现完整的动作编译与自适应执行路径，支持 `CompiledAction` 的存储与检索。
3. **三态转化**：建立了从源码态 (Markdown Skill) 到编译态 (CompiledAction) 再到执行态 (Fast-Path) 的完整生命周期管理。