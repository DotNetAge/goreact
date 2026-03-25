# GoReAct 重构与功能补全计划 (Phase 4 落地指南)

本计划直接指导下一阶段（即现在的开发重点）的代码编写。基于“基座与应用分离”的架构哲学，Phase 4 的核心目标是构建高级任务编排模式体系。

## 🎯 核心里程碑：引入 `pkg/pattern` 编排模式包

我们将把智能体任务编排的策略（Policy）从底层基座中解耦出来，放入专用的 `pattern` 包中。这将是一个双轨并行、先基建后起飞的工程演进过程。

---

### 阶段一：筑基 (Master-Sub 模式落地)

**目标**：实现现阶段最主流、最稳健的复杂任务处理方案：主从式 ReAct 编排（`pkg/pattern/mastersub`）。

*   **Task 1.1 定义 Task 模型 (pkg/pattern/mastersub/types.go)**
    *   定义 `Task` 结构，包含描述、依赖、状态（Pending/Running/Success/Failed/Skipped）。
    *   定义 `TaskResult`，包含最终答案和执行 Trace。
*   **Task 1.2 增强 Thinker 的拆解能力 (pkg/pattern/mastersub/master.go)**
    *   利用 LLM，将 `SKILL.md` 的内容以及用户意图拆解为 `[]Task`。
    *   这是整个主循环的“发电机”。
*   **Task 1.3 实现 Sub-Reactor (pkg/pattern/mastersub/sub.go)**
    *   封装底层 `engine.Reactor`。
    *   提供“单次原子调用”或“开启完整子 ReAct 循环”两种执行模式。
    *   处理执行结果，收集完整的 `PipelineContext.Traces` 以备后续“编译”使用。
*   **Task 1.4 Master-Sub 的状态机闭环**
    *   Master 管理任务队列，串行（或并行）分发给 Sub，收集结果，并在必要时触发“重排 (Re-plan)”。

---

### 阶段二：巅峰进化 (Evo 模式与智能体编译)

**目标**：在 Master-Sub 的基础上，实现 GoReAct 的独家杀手锏：基于预期修正的“三态转化”和自适应快径（`pkg/pattern/evo`）。

*   **Task 2.1 重新定义“肌肉记忆”结构 (pkg/pattern/evo/graph.go)**
    *   定义 `CompiledAction`，这是真正的编译态数据结构。
    *   包含变量提取模板 (`InputSchema`)、原子步骤序列 (`ActionSteps`)。
    *   **关键定义**：为每个步骤定义 `ExpectedObservation`（预期结果指纹）。
*   **Task 2.2 实现编译器 Compiler (pkg/pattern/evo/crystallizer.go)**
    *   读取 Master-Sub 成功执行后的完整 Trace。
    *   通过 LLM 或启发式算法，提取出参数变量和确定的调用路径。
    *   **最难点**：从成功的 `Actual` 中生成高度泛化的 `Expectation`。
*   **Task 2.3 升级 MemoryBank (pkg/memory/memory.go)**
    *   将 `MuscleMemory` 接口重构，支持读写序列化后的 `CompiledAction`，而不是简单的 `string`。
*   **Task 2.4 实现法官与警察模式 (pkg/pattern/evo/runner.go)**
    *   **执行快径**：读取图形，按顺序调用 Actor。
    *   **法官 Observer 裁定**：对比实际结果与预期指纹。匹配则短路执行（零 Thinker 介入）。
    *   **警察 Thinker 唤醒 (Escalation)**：当法官驳回时，挂起线性执行，唤醒 Full-ReAct 循环分析误差原因。
    *   **自我修复闭环**：若是预期落后，通知 `Compiler` 更新指纹。

---

## 📅 执行优先级建议 (The Action Plan)

1.  **P0 (Blocker): `pkg/pattern/mastersub` 基建。** 我们必须先有一个能够成功跑通复杂任务、并产生结构化 Trace 的系统，才谈得上“编译”。
2.  **P1: 定义 `CompiledAction`。** 这是 Evo 模式的核心数据契约，尽早确定，能指引 `mastersub` 收集哪些必要信息。
3.  **P2: `pkg/pattern/evo` 自适应快径实现。** 这是最硬核的挑战，标志着 GoReAct 完成从“提示词工程”到“编译引擎”的跨越。
