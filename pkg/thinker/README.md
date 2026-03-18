# Thinker (思考者)

`Thinker` 位于 GoReAct 架构（Thinker - Actor - Observer - Terminator）的最前端，是整个 ReAct（Reasoning + Acting）引擎的“大脑”。

它不仅负责处理与用户的首次交互与意图理解，更是每一次 ReAct 循环中负责**推理 (Reasoning)**、**规划 (Planning)** 与 **决策 (Decision Making)** 的核心组件。借鉴业界前沿框架（如 LangChain, LlamaIndex, AutoGen 以及 Reflexion 论文）的设计理念，GoReAct 的 `Thinker` 被设计为一个高度模块化、可扩展的认知流水线。

---

## 核心职责 (Core Responsibilities)

`Thinker` 的工作贯穿于 Agent 的整个生命周期，具体包含以下五大核心能力：

### 1. 意图识别与任务规划 (Intent Recognition & Task Planning)
在用户发起请求的第一时间，Thinker 需要对输入进行深度解析：
- **意图缓存短路 (Intent Caching)：** 快速查找语义相似的历史请求或缓存，若命中则直接返回，降低大模型调用成本。
- **用户意图分类与路由：**
  - **澄清需求：** 判断输入是否含糊不清，是否需要反问用户补充上下文。
  - **模型路由 (Model Routing)：** 根据意图的知识深度和复杂度，动态决定是将任务分配给本地小模型（如 Ollama）以提升速度和隐私，还是分配给云端大模型（如 GPT-4, Claude 3）以处理复杂推理。
  - **周期识别：** 识别当前指令是一次性问答、定时任务还是需要长期驻留的循环监控任务。
- **任务拆解 (Decomposition & Planning)：** 针对宏大或复杂的指令（如“分析苹果公司财报并写一份研报”），Thinker 需要构建前置规划。将复杂目标拆解为顺序执行的子任务队列或有向无环图（DAG），并在执行过程中动态维护这张“计划表”。
- **基于 Skill 的流程编排与 RAG 缓存 (Skill-based Orchestration & RAG Caching)：** 当意图识别系统匹配到预置的特定能力或场景（即 Skill，例如“竞品分析”）时，Thinker 将加载该 Skill 中固化的 SOP（标准作业程序）。为了降低频繁调用带来的 Token 消耗，Thinker 借助 RAG（或向量数据库）将复杂 Skill 的“拆解逻辑与编排流程”持久化。后续遇到同类任务时，直接从 RAG 召回并复用已缓存的任务拆解结构与历史优化的 Prompt 模板，避免每次都需要让大模型从零开始消耗大量 Token 去理解和分解冗长的 Skill 指令。

### 2. 上下文与记忆管理 (Context & Memory Management)
在将问题提交给大模型之前，Thinker 负责组装最完美的上下文：

- **全局思考上下文 (Pipeline Context)：** 在整个 ReAct 引擎管线中维护并共享状态。详尽记录每个环节（Thinker 决策、Actor 执行、Observer 观察、Terminator 判定）的输入数据与输出结果，这不仅作为历史轨迹供 LLM 随时回溯，也为各组件间的状态传递提供“数据总线”。
- **提示词组装与清洗：** 利用 `PromptBuilder` 过滤无关字符，组装系统预设指令 (System Prompt)。
- **轨迹管理 (Agent Scratchpad)：** 在多步 ReAct 循环中，Thinker 负责收集并格式化之前的“思考记录(Thought)”、“动作(Action)”和“观察结果(Observation)”，作为当前轮次思考的历史依据。
- **滑动窗口与上下文压缩：** 当历史对话或观察结果（如超长网页源码）过长时，触发 Token 压缩策略（摘要、截断或向量化检索），严格控制会话窗口的 Token 消耗，防止爆显存或超出 LLM 限制。

### 3. 高级 RAG 增强 (Retrieval-Augmented Generation)
结合 GoRAG 的先进理念，提升大模型回答的准确率和时效性：
- **记忆检索 (Memory RAG)：** 从短期/长期记忆库中提取与当前意图相关的历史偏好和事实。
- **知识库增强 (Knowledge RAG)：** 挂载外部向量数据库或文档。
- **高级查询转换 (Query Transformations)：**
  - **HyDE (Hypothetical Document Embeddings)：** 让 LLM 先生成假设性答案，再以此去检索更相关的底层文档。
  - **Step-Back Prompting：** 自动将用户的具象问题抽象化，以检索到更丰富的背景知识。

> GoRAG提供了完整的RAG组件得处理任何形式的RAG功能

### 4. 动作决策与工具绑定 (Action Formulation & Tool Binding)
思考的最终目的是为了行动。Thinker 需要决定下一步“做什么”以及“用什么工具”：
- **动态工具路由 (Tool RAG)：** 当系统注册了海量工具时，全部注入 Prompt 会导致 Token 溢出。Thinker 会根据当前意图，动态检索并过滤出当前步骤最相关的 Top-K 个工具 Schema (如 JSON Schema)。
- **思维链引导 (Chain-of-Thought)：** 强制或引导大模型输出严密的推理步骤。
- **动作参数构造：** 引导 LLM 为选定的工具生成精准的输入参数。


### 5. 输出解析与反思纠错 (Output Parsing & Reflection)
将大模型的自然语言输出，转化为下游 `Actor` 能够直接执行的结构化指令：
- **结构化解析 (Output Parsers)：** 使用正则或 JSON/XML 解析器，强制提取出 `[Thought, Action, ActionInput]` 结构。
- **格式自动修复 (Auto-Fixing)：** 当 LLM 输出格式错乱时，Thinker 内部实施重试或提供纠错提示（"Your output was not valid JSON, please fix..."）。
- **反思与自我纠正 (Reflexion)：** 接收 `Observer` 传来的执行失败反馈（如工具调用超时、参数类型错误）或 `Terminator` 的不满意评估。在下一次 Prompt 中注入反思经验：“上次尝试失败了，原因是 XXX，我这次应该换一种工具或改变参数”。

---

## 架构集成位置 (Integration in ReAct Loop)

在一次典型的 GoReAct 执行循环中，Thinker 的位置如下：

1. **[Thinker]** 接收 User Input 或前一轮的 Observation。
2. **[Thinker]** 检索记忆、加载工具、进行思考，最终输出 `Action` 和 `ActionInput`。
3. **[Actor]** 接收 Thinker 的输出，实际执行工具调用（如发 HTTP 请求、执行代码）。
4. **[Observer]** 观察 Actor 的执行结果，清理并格式化反馈数据。
5. **[Terminator]** 检查是否达到最终目标。若未达到，将 Observation 再次送入 **[Thinker]** 开启下一轮。

## 设计指引 (Design Guidelines)

- **接口隔离：** `Thinker` 应该是一个顶层 Interface。内部的 Planner, OutputParser, QueryTransformer 应当拆分为独立的可插拔组件。
- **Pipeline模式：** Thinker 的预处理阶段（如 RAG 检索、Token 压缩）是一种独立的管线或者是处理步骤，gochat/pkg/pipeline 的管线模式可以将每个部分进行进理步骤的包装分片独立处理，以达到既能直接调用包内的接口又能通过步骤组合灵活组成不同的管线。
- **实时思考流同步 (Thought Streaming)：** 在进行“慢思考”时，大模型可能需要数秒甚至十几秒来生成 `Thought` 或 `Action` 的 JSON 文本。Thinker 必须支持流式（Streaming）回调通道，让外界（如前端 UI）能像打字机一样，实时看到 Agent “正在思考的过程” 和 “准备调用的工具名称”，极大优化用户体验。
- **精准的 Token 审计与归因 (Token Accounting)：** 为了控制 Agent 的高昂使用成本，每一次 ReAct 循环都必须将消耗的 `PromptTokens` 和 `CompletionTokens` 精准累加至 PipelineContext 的生命周期中，以便 Terminator 基于财务预算做出硬性熔断裁决。

- **可观测性 (Observability)：** 思考过程的每一个子阶段（检索了什么、压缩了多少 Token、生成的原始 Prompt 是什么）都必须对外暴露 Hook 或 Trace，以便于调试和性能监控。
- **依赖注入(Dependence Injection)**
Thinker 通过DI的方式，采用Go特有的Option Pattern注入不同接口实例，与不同的外部功能进行交互。
- LLMClient - gochat/pkg/core.Client
- Tools RAG - gorag/infra/searcher/native

