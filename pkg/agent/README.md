# Agent & Multi-Agent 架构设计

本目录 (`pkg/agent`) 负责管理智能体（Agent）的定义、协调与多智能体协作网络。

在 GoReAct 的设计哲学中，我们摒弃了复杂、难以调试的分布式 P2P 消息总线模型，转而拥抱第一性原理：**Agent-as-a-Tool (AAAT)**。

## 1. 核心理念：Skill 才是一切的抓手

在处理复杂系统时，大模型（LLM）的 Zero-Shot 推理能力往往不足以支撑长链路的精确执行。解决复杂问题的唯一抓手是 **Skill（技能）**。

- **Skill = SOP（标准作业程序） + Tools（特定工具集） + Sub-Agent（子智能体）**。
- 当高层 Agent 面临一个巨大任务时（如“帮我 Review 这段代码并提交 PR”），它不需要自己去一步步执行 git clone、查代码、写评论。
- 它只需要调用一个名为 `CodeReviewerSkill` 的工具。

## 2. Agent-as-a-Tool (AAAT) 多智能体协作

在 GoReAct 中，**一个 Agent 本身也可以被包装成一个实现 `tools.Tool` 接口的特殊对象**。

### 协作流程图：
1. **[Supervisor Agent (Manager)]** 收到用户指令：“帮我分析这个数据，并写一份视觉报告”。
2. **[Supervisor Thinker]** 查阅 ToolManager，发现有两个牛逼的工具：`DataAnalysisAgent` 和 `ReportWritingAgent`。
3. **[Supervisor Actor]** 执行 `DataAnalysisAgent.Execute(input)`。
4. **[Child Reactor]** 此时，底层的 `DataAnalysisAgent` 启动了属于自己的 `Thinker -> Actor -> Observer` 闭环，进行 SQL 查询、计算。
5. **[Supervisor Observer]** 拿到子 Agent 的最终结果（作为 Observation）。
6. **[Supervisor Thinker]** 再将数据传给 `ReportWritingAgent`。

### 优势
- **极度优雅的良性闭环**：顶层框架的 `Reactor` 接口与底层的 `Tool` 接口完美嵌套（类似分形结构）。
- **职责隔离**：父 Agent 只负责规划调度，子 Agent 拥有自己独立的 System Prompt、小模型（更便宜）和独立的专属小工具集。
- **降级与熔断安全**：子 Agent 的死循环会被自身的 `Terminator` 掐断，只返回一个 Error 给父 Agent，父 Agent 可以选择重试或更换方案，不会导致整个系统崩溃。

## 3. Coordinator (协调器)
对于预定义好的线性或 DAG 任务，`Coordinator` 可以通过 `TaskDecomposer` 预先将大任务拆解为子任务（SubTask），并硬性指派给对应的子 Agent 执行，从而节省让大模型实时规划的 Token 成本。
