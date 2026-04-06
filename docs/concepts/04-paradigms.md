# 设计范式：ReAct 的提示工程与架构

要让一个通用的大语言模型（如 GPT-4, Claude 等）表现出 ReAct 的行为，通常不需要重新预训练或进行复杂的微调（Fine-tuning）。在大多数情况下，通过精心设计的**提示工程（Prompt Engineering）**即可实现。

## 核心 Prompt 模板设计

一个标准的 ReAct Agent 架构包含一个系统提示（System Prompt），它定义了智能体的角色、可用的工具列表，以及严格的输出格式。

### 基础模板结构

```text
你是一个可以解决复杂问题的智能助手。
你可以使用以下工具：

- Search[query]: 在互联网上搜索信息。
- Calculator[expression]: 计算数学表达式。
- Finish[answer]: 当你得出最终答案时调用。

为了解决用户的问题，你必须严格遵循以下格式进行思考和行动交替的循环：

Question: 用户提出的问题
Thought: 思考你需要做什么，以及如何利用手头的工具。
Action: 必须是上述可用工具之一，格式为 ToolName[Argument]。
Observation: 工具执行的结果。（注意：这不是由你生成的，系统会自动返回）
... (Thought/Action/Observation 循环可以重复多次)
Thought: 我现在知道了最终答案。
Action: Finish[最终答案]
```

### 解析与执行器 (Parser and Executor)

在工程实现层面（如在 LangChain 或 LlamaIndex 框架中，或者在您正在开发的 goRAG 中），系统需要一个“控制器”（Controller/Executor）来处理 LLM 的输出。

1.  LLM 生成到 `Action: ...` 停止。
2.  **Parser** 解析文本，提取出工具名称（如 `Search`）和参数（如 `Tim Cook`）。
3.  **Executor** 调用实际的 Python/Go 代码或 API 执行搜索。
4.  将结果格式化为 `Observation: ...\n`，拼接到原来的 prompt 后面，再次发送给 LLM。

## 少样本学习策略 (Few-shot Prompting)

在 ReAct 的原始论文中，作者发现，对于具有挑战性的任务（如 ALFWorld 文本游戏或 HotpotQA 多跳问答），仅仅提供系统提示（Zero-shot）往往不够。模型可能无法领会如何进行有效的分解，或者容易产生格式错误。

引入高质量的**少样本示例 (Few-shot Demonstrations)** 是激发 ReAct 潜能的关键。

### 示例设计原则

1.  **覆盖多样性:** 示例应涵盖成功路径、错误恢复路径（如搜索不到结果时换词重试）以及提前终止的路径。
2.  **展示复杂逻辑:** 在 Thought 中明确展示人类是如何推导和规划的。
3.  **遵循严格格式:** 示例的格式必须与系统提示中定义的格式绝对一致。

> **Few-shot 示例片段：**
>
> Question: Which documentary is about Finnish rock groups, Paradise Lost or The Saimaa Gesture?
> Thought: I need to search Paradise Lost and The Saimaa Gesture, and find which documentary is about Finnish rock groups.
> Action: Search[Paradise Lost]
> Observation: Paradise Lost is a documentary film about the West Memphis Three...
> Thought: Paradise Lost is not about Finnish rock groups. I should search The Saimaa Gesture.
> Action: Search[The Saimaa Gesture]
> Observation: The Saimaa Gesture is a 1981 Finnish documentary film... about three Finnish rock groups.
> Thought: The Saimaa Gesture is about Finnish rock groups. I have the answer.
> Action: Finish[The Saimaa Gesture]

## 动态 Prompt 构建

在实际的 Agent 开发中，Prompt 往往是动态生成的。

*   **动态工具列表:** 根据用户的权限或当前的任务阶段，向 Prompt 中注入不同的工具集。
*   **截断历史 (Context Window Management):** 随着 T-A-O 循环的增加，Prompt 长度可能超过模型的上下文窗口。系统需要策略性地压缩或丢弃早期的 Observation，或者通过汇总（Summarization）来浓缩历史。

通过这种“模板约束+实例引导”的范式，ReAct 成功地将非结构化的自然语言模型转化为了一个能够执行确定性工作流的推理机。