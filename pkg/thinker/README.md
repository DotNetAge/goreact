# Thinker (思考者)

`Thinker` 位于 GoReAct 架构的最前端，是整个 ReAct 引擎的“大脑”。它负责推理、规划与动作决策。

---

## 1. 核心架构：内部子管线 (Internal Sub-pipeline)

重构后的 `Thinker` 不再是一个臃肿的单一函数，而是一个高度模块化的**内部子管线**结构：

1.  **Intent & Mode Resolution (意图与模式解析)**：识别输入中的“暗语（Codewords）”，决定当前思维模式。
2.  **Tool Discovery (工具发现)**：根据当前任务动态筛选最相关的工具集。
3.  **Prompt Synthesis (提示词合成)**：利用 `pkg/prompt/builder` 流式组装系统指令、历史记录、Few-Shot 示例及记忆体。
4.  **Context Compression (上下文压缩)**：自动应用 Token 窗口管理策略（如 SlidingWindow），防止上下文溢出。
5.  **LLM Execution (执行调用)**：通过 `gochat` 客户端执行流式推理。
6.  **Output Reflection (输出反思与解析)**：解析 `Thought/Action/ActionInput` 结构，并对格式错误进行自动纠错反思。

---

## 2. 思维暗语驱动 (Codeword Driven Modes)

通过在输入中加入前缀暗语，可以强制切换 `Thinker` 的思考深度与管线行为：

-   **`/plan` (Planning Mode)**：
    -   **职能**：专注于长程任务拆解。
    -   **行为**：不执行具体工具，而是输出一份结构化的阶段性目标路线图。
-   **`/specs` (Specification Mode)**：
    -   **职能**：需求与约束分析。
    -   **行为**：强制召回 `MemoryBank` 中的历史约束与技术细节，生成详尽的规格说明书。
-   **Default (ReAct Mode)**：
    -   **职能**：标准的推理-行动循环。
    -   **行为**：输出 `Thought -> Action -> ActionInput`，直接驱动 `Actor` 执行任务。

---

## 3. 自维护指令 (Self-Maintenance Commands)

除了思维模式切换，Thinker 还支持对上下文生命周期的直接干预：

-   **`/clear`**：
    -   **作用**：物理清空当前会话的所有历史轨迹（Traces）。
    -   **场景**：当上下文过长、出现逻辑死循环或需要切换完全无关的话题时。
-   **`/compress`**：
    -   **作用**：强制触发极度激进的上下文压缩（仅保留最近 1 轮）。
    -   **场景**：手动 Token 瘦身，保留当前意图但丢弃冗长过程。

---

## 4. 有机集成：Prompt & Memory

`Thinker` 与其他核心包实现了深度有机整合：

-   **Prompt Toolkit**：完全接管了提示词的构建与格式化，通过 `Fluent API` 确保了指令的精准度。
-   **Memory Bank**：三模态记忆（工作记忆、语义知识、肌肉经验）被作为动态变量无缝注入 Prompt 模板，消除了“事实幻觉”与“操作幻觉”。
-   **Auto-Reflection**：当 LLM 输出无法被 `Parser` 解析时，Thinker 会自动生成一条 `Format Error Reflection` 轨迹，告知模型错误原因并引导其纠正。

---

## 4. 接口指引

```go
type Thinker interface {
    // Think 执行一轮思考循环，将决策结果更新至 PipelineContext
    Think(ctx *PipelineContext) error
}
```
通过 Option 模式支持自定义模型、工具管理器、记忆体及系统模板。
