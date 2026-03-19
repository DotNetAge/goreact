# GoReAct 重构与功能补全计划 (Phase 4 落地指南)

基于我们对架构设计的深度梳理，现将各个模块从“设计图纸”走向“代码落地”的具体重构任务清单总结如下。

本计划作为 `TESTING_PLAN.md` 的延续和深化，直接指导下一阶段的代码编写。

## 🎯 模块一：`@pkg/tools` 与 `@pkg/actor` (安全与执行防线)

### 1.1 Tool 安全定级机制 (已完成 ✅)
- **动作**：修改 `tools.Tool` 接口，增加 `SecurityLevel() SecurityLevel` 方法。
- **动作**：为 `builtin` 目录下的所有内置工具实现该方法，分类为 `LevelSafe`, `LevelSensitive`, `LevelHighRisk`。
- **状态**：代码已提交，编译通过。

### 1.2 基于 Pipeline Hook 的 HITL 阻断机制 (已完成 ✅)
这是当前架构中最核心的安全特性落地。
- **任务目标**：利用 `gochat` 的 Pipeline Hook 机制，在 `Actor Step` 执行前拦截高危工具，等待人类（宿主应用）授权。
- **状态**：通过 `actor.SecurityHook` 实现 `gochat.pipeline.Hook` 接口，结合 `MemoryBank` 的 `Whitelist` 管理和回调等待。

## 🎯 模块二：`@pkg/agent` 与 `@pkg/model` (已完成 ✅)

### 2.1 Agent Builder / 装配工厂 (已完成 ✅)
- **任务目标**：让开发者可以直接调用 `agent.Chat()`。
- **状态**：通过 `agent.Builder` 实现了从 `Agent` 配置到 `engine.Reactor` 的全自动化装配。`agent.Manager` 现在可以根据 `Name` 或语义描述召回并自动激活 Agent。
- **功能**：
  - 自动创建 LLM Client 并配置给 `Thinker`。
  - 注入 `SystemPrompt` 和可用工具集。
  - 暴露 `Chat(ctx, task)` 入口。

### 2.2 Agent-as-a-Tool (AAAT) 的终极闭环 (已完成 ✅)
- **任务目标**：让一个装配好的 Agent 能够直接变成 Tool 塞给另一个 Agent。
- **状态**：`agent.Agent` 结构体已通过实现 `tools.Tool` 接口的所有方法（`Name`, `Description`, `Execute` 等）完成了闭环。支持 Agent 间的递归嵌套调用。

## 🎯 模块三：`@pkg/skill` (已完成 ✅)

### 3.1 执行数据的闭环上报 (已完成 ✅)
- **任务目标**：将每次 ReAct 循环的统计数据喂给 Skill 系统。
- **状态**：在 `agent.Agent.Chat()` 结束时，提取 `PipelineContext` 中的 `TotalTokens`、`StartTime` 以及状态，调用 `a.skillManager.RecordExecution(...)` 上报指标。打通了数据的自反馈闭环。

### 3.2 演化调度器 (Scheduler) (已完成 ✅)
- **任务目标**：让 `EvolveSkills` 能自动触发完成优胜劣汰。
- **状态**：在 `skill.Manager` 接口与实现中新增了 `StartEvolutionScheduler(ctx, interval)` 方法。基于定时器定期扫描内存中的技能库，执行淘汰逻辑。

---

## 🎯 模块四：`@pkg/memory` 与幻觉消除机制 (Agent 记忆体抽象)

### 4.1 废弃独立 RAG 接口 (已完成 ✅)
- **动作**：删除游离的 `pkg/rag` 目录。
- **原因**：RAG 只是一项技术（向量检索），而它在系统中的应用学名叫做“记忆 (Memory)”。

### 4.2 定义 `MemoryBank` 与三模态记忆体系 (已完成 ✅)
- **任务目标**：将记忆抽象为 Agent 独有的属性（即 Agent = Prompt + Reactor + MemoryBank）。
- **状态**：
  - 在 `pkg/memory` 重新定义了 `MemoryBank` 接口，包含 `WorkingMemory`, `SemanticMemory`, `MuscleMemory`。
  - 在 `agent.Agent` 与 `agent.Builder` 中成功挂载了 `MemoryBank` 实例。
  - 在 `thinker.defaultThinker` 的 `buildMessages` 中，拦截 `MemoryBank`，聚合召回三种记忆（包括短期上下文、知识库与技能执行经验），并封装入 System Prompt 前端。

  ### 4.3 终极进化：著书立说与开疆拓土 (Knowledge Crystallization) (已完成 ✅)
  - **任务目标**：完成经验的反写与**全新技能**的自动涌现。
  - **状态**：
    - 在 `pkg/skill/crystallizer.go` 中定义了 `Crystallizer` 接口。
    - 提供 `RefineSkill` 用于基于 `MuscleMemory` 重新打磨和覆写现有技能。
    - 提供 `DiscoverNewSkill` 用于基于 `WorkingMemory` 发现长尾会话需求，并无中生有地生成和注册新技能。
  ---

  ## 📅 执行优先级建议
  1. **P0 (Blocker)**: 完成「模块二：Agent 装配工厂」。因为如果没有这个，外部应用根本无法正常使用这个框架，所有的 Example 都是手拼的脏代码。
  2. **P1 (Core Feature)**: 完成「模块一：基于 Hook 的 HITL 阻断机制」。这是构建安全可信 Agent 的护城河。
  3. **P2 (Enhancement)**: 完成「模块三：Skill 数据上报与调度器」。补全系统的“自我进化”能力。
  4. **P3 (Ultimate)**: 完成「模块四：Agent 记忆体抽象与知识固化」。实现从短期记忆到肌肉记忆，再反写 `SKILL.md` 的终极进化闭环。