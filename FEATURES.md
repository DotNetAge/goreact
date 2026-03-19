# GoReAct 核心特性 (Core Features) - Phase 4 进化版

GoReAct 是一个具备**逻辑推理、自主规划与自进化能力**的 ReAct (Reasoning + Acting) 智能体框架。基于深度架构重构，它实现了从“简单的 ReAct 循环”到“可编程认知引擎”的跨越。

以下是贯穿整个框架体系的核心特性集：

## 1. 万能逻辑管线 (Universal Pipeline & Logic Primitives)
* **图灵完备的执行流**：GoReAct 不再局限于线性的任务执行。通过深度集成 `gochat` 原生的控制流原语，管线原生支持：
    -   **Sequence (顺序)**: 默认的任务流。
    -   **Branch (分支 - `IfStep`)**: 基于上下文观察结果的实时决策。
    -   **Iteration (循环 - `LoopStep`)**: 带有物理熔断（MaxLoops）与哨兵错误（`Break`/`Return`）支持的重复执行。
* **逻辑驱动力**：这意味着 Agent 可以处理“如果失败则重试”、“循环直到找到结果”等复杂逻辑，而无需在 Thinker 内部编写混乱的 Prompt。

## 2. 认知流拆解与模式驱动 (Cognitive Decomposition)
* **暗语驱动模式 (Codeword-Driven)**：通过前缀暗语（如 `/plan`, `/specs`）精准控制 Thinker 的思考深度。
    -   **`/plan` (Planning Mode)**：专注于任务拆解，生成可被解析为 Pipeline Steps 的结构化路线图。
    -   **`/specs` (Specification Mode)**：强制召回记忆中的所有技术约束，生成详尽的需求说明。
* **自维护指令**：支持 `/clear`（清空轨迹）和 `/compress`（强制压缩），赋予 Agent 自动管理上下文生命周期的能力。

## 3. 三模态仿真记忆系统 (Tri-Modal MemoryBank)
*记忆的本质是为了消除幻觉。Agent 独占的 MemoryBank 完美映射了人类认知：*
* **短期/工作记忆 (Working Memory)**：记录当前会话的状态与临时授权，内置时间衰减机制，保证上下文不过载。
* **永久/知识记忆 (Semantic Memory/RAG)**：通过分布式 RAG 锚定外部事实，**消除事实类主幻觉**。
* **经验/肌肉记忆 (Muscle Memory)**：Agent 将历史任务中“碰壁-反思-成功”的过程蒸馏为 SOP 捷径。再次执行同类任务时，直接唤醒经验，**消除操作类幻觉，实现极速响应**。

## 4. 技能进化与达尔文演化 (Skill Evolution)
* **Markdown SOP 驱动**：技能（Skill）是一份人类可读的 `SKILL.md` 指南。框架将其动态注入子智能体的 System Prompt 中，实现柔性执行。
* **著书立说 (Skill Refinement)**：Agent 定期对高权重的“肌肉记忆”进行二次蒸馏，反向覆写回原始的 `SKILL.md`，实现对既有经验的升华。
* **开疆拓土 (Skill Discovery)**：Agent 通过分析长期沉淀的对话轨迹，自主挖掘高频需求，**自动撰写出全新的 `NEW_SKILL.md`**，实现从“执行者”向“创造者”的跃迁。

## 5. 提示词工程工具箱 (Prompt Toolkit)
* **协议化提示词**：Prompt 被视为 Agent 的运行时配置协议。
* **Fluent API Builder**：流式组装 System 指令、Tools 定义、Few-Shot 示例与记忆上下文。
* **智能窗口压缩**：内置 `SlidingWindow` 策略与多模态 Token 计数器（精确支持中英混合），确保在长程管线执行中的 Token 安全与成本最优。

## 6. 万物皆工具 (Agent-As-A-Tool, AAAT)
* **分形嵌套架构**：任何一个 Agent 实例都可以被包装为 `tools.Tool`。Supervisor Agent 可以像调用计算器一样调用 Sub-Agent。
* **Sudo HITL 安全防线**：所有工具强制声明 `SecurityLevel`。遇到敏感操作（如删除文件、发送邮件），系统自动触发安全钩子（SecurityHook），向人类请求授权，支持“单次允许”与“永久加白”。

## 7. 极致解耦的 Reactor 引擎
* **Reactor 状态机**：极致精简的四步循环（Think -> Act -> Observe -> Terminate）。
* **应用与引擎分离**：底层极客可以定制微观状态转换，而应用层开发者只需通过 `AgentBuilder` 配置角色与模型即可获得完整的战斗力。
