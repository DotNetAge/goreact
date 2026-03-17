# Terminator (判定器 / 终结者)

`Terminator` 位于 GoReAct 架构（Thinker - Actor - Observer - Terminator）的最后一环。在无限的 ReAct 循环（Thought -> Action -> Observation -> ...）中，必须有一个组件来决定“何时该停止”。

它负责审查整个执行管线的状态（Pipeline Context），基于预设的边界条件、安全约束、成本消耗或任务完成度，来判定 Agent 是否已经达成了用户最初的意图（或由于资源耗尽而被迫中止）。

---

## 核心职责 (Core Responsibilities)

一个成熟的 Agent 系统极易陷入死循环（如大模型反复调用同一个无效工具），因此 Terminator 不仅仅是一个简单的 `if isFinished == true` 开关，它承担着系统最后一道“防线”的职责：

### 1. 成功态判定 (Success Resolution)
最理想的情况，大模型自我感知到了任务完成：
- **终结指令识别 (Finish Signal)：** 解析 Thinker/Actor 输出的特殊信令（如 OpenAI 的 `finish_reason: stop` 或 LangChain 的 `Final Answer:` 动作）。
- **目标一致性校验 (Goal Validation)：** 基于用户的原始意图，校验最终的 Observation 是否真正回答了用户的问题。有时候 LLM 会“虚假陈述”任务已完成，Terminator 可以调用另一个轻量级裁判模型（Critic Model）来验证答案的可靠性。

### 2. 停滞与死循环检测 (Stagnation & Loop Detection)
Agent 的通病是“原地打转”，Terminator 必须能及时打断这种愚蠢行为：
- **重复动作检测 (Repetitive Action Tracing)：** 分析历史轨迹（Scratchpad），如果发现 Agent 连续 N 次输出了完全相同的 `Action` 和 `ActionInput`（且得到了相同的 `Observation`），立刻判定为死循环并强制终止。
- **无意义探索阻断 (Hallucination Cutoff)：** 如果 LLM 开始胡编乱造不存在的工具，或者连续 M 次调用工具失败且没有实质进展，强行介入并向上层抛出 `StagnationError`。

### 3. 边界与硬性约束 (Boundaries & Hard Constraints)
即使任务进展顺利，也可能因为外部物理限制而必须叫停：
- **最大迭代步数 (Max Steps / Max Iterations)：** 设定硬性上限（如最多循环 15 次），超过则视为执行超时并退出。
- **全局 Token 与成本熔断 (Token / Cost Budgeting)：** 根据 Pipeline Context 中累计的 Token 消耗量或估算 API 费用。一旦当前请求的预算（如：这笔查询最多允许消耗 0.5 美元）即将超支，立刻切断循环并返回阶段性成果。
- **时间窗限制 (Time-to-Live, TTL)：** 基于 `context.Context` 设置总耗时（如 30 秒），超时无条件中断 Actor 的长连接并返回失败。

### 4. 阶段性产出与降级 (Partial Output & Graceful Degradation)
当任务被强制终止（超时或预算耗尽）时，Terminator 不应直接抛出冰冷的 Panic：
- **优雅降级 (Graceful Degradation)：** 尝试从已有的 `Observation` 中提取阶段性成果，拼接一段类似于“*很抱歉，由于分析过程过于漫长，我暂时为您总结了前几个步骤的结论：XXX...*” 的回复。
- **转交人工 (Escalation)：** 当判定该任务超出 Agent 的处理边界时，自动触发转交人类客服（Human Fallback）的流程。

---

## 架构集成位置 (Integration in ReAct Loop)

在一次典型的 GoReAct 执行循环中，Terminator 的位置如下：

1. 引擎管线完成了一次完整的 `Thinker -> Actor -> Observer` 轮次。
2. 全局的 Pipeline Context 数据被更新（步数+1，累积 Token，最新的 Observation 等）。
3. **[Terminator]** 介入，执行一系列断言与拦截规则：
   - 检查 `MaxSteps`：当前是第 10 步，未超限，继续。
   - 检查 `CostBudget`：当前消耗 0.05 美元，未超限，继续。
   - 检查 `Observation` 是否包含 `FinalAnswer` 标记。
4. **[Terminator]** 做出裁决：
   - 若返回 `true`（停止）：引擎退出循环，将最终结果通过 Thinker 的格式化后返还给用户。
   - 若返回 `false`（继续）：引擎将最新的 Context 流转回第一步，由 **[Thinker]** 发起新一轮推理。

## 设计指引 (Design Guidelines)

- **策略链模式 (Chain of Responsibility / Strategies)：** `Terminator` 的内部判断逻辑应设计为一个个独立的策略组件（如 `MaxStepsCondition`, `CostBudgetCondition`, `StagnationDetector`），并在初始化时注入为一个链表/数组。任何一个策略返回 `Stop`，整个管线即告终止。
- **状态透明化：** Terminator 在决定终止时，必须输出明确的终止原因（Reason），比如 `StoppedByBudget` 或 `FinishedWithSuccess`，这对于调试和日志审计至关重要。