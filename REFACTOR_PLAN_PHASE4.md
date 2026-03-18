# GoReAct 重构与功能补全计划 (Phase 4 落地指南)

基于我们对架构设计的深度梳理，现将各个模块从“设计图纸”走向“代码落地”的具体重构任务清单总结如下。

本计划作为 `TESTING_PLAN.md` 的延续和深化，直接指导下一阶段的代码编写。

## 🎯 模块一：`@pkg/tools` 与 `@pkg/actor` (安全与执行防线)

### 1.1 Tool 安全定级机制 (已完成 ✅)
- **动作**：修改 `tools.Tool` 接口，增加 `SecurityLevel() SecurityLevel` 方法。
- **动作**：为 `builtin` 目录下的所有内置工具实现该方法，分类为 `LevelSafe`, `LevelSensitive`, `LevelHighRisk`。
- **状态**：代码已提交，编译通过。

### 1.2 基于 Pipeline Hook 的 HITL 阻断机制 (待开发 ⏳)
这是当前架构中最核心的安全特性落地。
- **任务目标**：利用 `gochat` 的 Pipeline Hook 机制，在 `Actor Step` 执行前拦截高危工具，等待人类（宿主应用）授权。
- **具体实现步骤**：
  1. 定义 `SecurityHook` 结构体，实现 `gochat.Hook` 接口的 `BeforeStep` 方法。
  2. 在 `BeforeStep` 中检查当前要执行的工具的 `SecurityLevel`。
  3. **挂起与交互设计**：
     - 如果检测到 `LevelHighRisk`，通过 Go 的 `channel` 或回调函数将当前 `Context` (包含即将执行的命令参数) 抛给外部宿主应用。
     - 宿主应用（如 CLI 终端或 Web UI）负责渲染提示并等待用户输入（Approve/Reject）。
     - Hook 内部使用 `<-chan` 阻塞等待宿主应用的回应。
  4. **白名单存储设计 (Whitelist Storage)**：
     白名单本质上是 Agent “记忆”的一部分，因此必须将其与 `pkg/memory` 融合：
     - **Session 级短效加白**：通过 `reactCtx.Set("whitelist:xxx", true)` 记录，仅在当前大循环内有效，引擎结束即销毁。
     - **跨会话持久加白 (Long-term)**：如果用户点击了“永久加白”，Hook 将调用 `memory.Manager.Store(ctx, sessionID, "whitelist:xxx", true)`。下次拦截时，Hook 优先调用 `memory.Retrieve` 检查该动作是否已经被人类永久许可。
  5. **如果被拒**：Hook 直接返回特定的 `ErrOperationRejectedByUser`，阻止 Actor 执行，并让 Observer 捕获此错误喂给 Thinker。

## 🎯 模块二：`@pkg/agent` 与 `@pkg/model` (应用层入口装配)

### 2.1 Agent Builder / 装配工厂 (待开发 ⏳)
目前的 `agent.Manager` 只返回了纯数据结构 `*agent.Agent`，它离真正的应用层入口还差一个“组装工厂”。
- **任务目标**：让开发者可以直接调用 `agent.Chat()`。
- **具体实现步骤**：
  1. 在 `pkg/agent` 中新增一个 `AgentRunner` 结构体或直接改造 `Agent` 接口，让其持有一个私有的 `*engine.Reactor`。
  2. 修改 `agent.Manager.SelectAgent()`，让其在返回前，自动完成依赖装配：
     - 调用 `model.Manager.CreateLLMClient(agent.ModelName)` 获取大脑。
     - 初始化 `Thinker`，将 Agent 的 `SystemPrompt` 和大模型客户端注入进去。
     - 初始化 `Actor` 和需要的 `Tools`。
     - 把它们全绑进一个新建的 `Reactor` 实例里。
  3. 暴露多模态入口方法：
     - `func (a *AgentRunner) Chat(ctx, task string) (string, error)`
     - `func (a *AgentRunner) ChatWithFiles(ctx, task string, files []string) (string, error)`

### 2.2 Agent-as-a-Tool (AAAT) 的终极闭环 (待开发 ⏳)
- **任务目标**：让一个装配好的 Agent 能够直接变成 Tool 塞给另一个 Agent。
- **具体实现步骤**：
  - 为上面实现的 `AgentRunner` 直接添加 `Name()`, `Description()`, `SecurityLevel()` 和 `Execute()` 方法，让其在接口签名上完全等价于一个 `tools.Tool`。

## 🎯 模块三：`@pkg/skill` (达尔文演化与生命周期)

### 3.1 执行数据的闭环上报 (待开发 ⏳)
虽然 `skill.Manager` 写好了 `RecordExecution` 等统计方法，但引擎从未调用过它们。
- **任务目标**：将每次 ReAct 循环的统计数据喂给 Skill 系统。
- **具体实现步骤**：
  - 在 `engine.Reactor.Run()` 的最后阶段（或者利用 Pipeline 的 `AfterRun` Hook），检查当前 PipelineContext 中是否被打上了“当前正在使用某个 Skill”的标记。
  - 如果有，提取 `reactCtx.TotalTokens`、`TotalTime` 和是否成功，调用 `skillManager.RecordExecution(...)`。

### 3.2 演化调度器 (Scheduler) (待开发 ⏳)
- **任务目标**：让 `EvolveSkills` 不再是死代码，而是能自动触发。
- **具体实现步骤**：
  - 在 `skill.Manager` 中提供一个 `StartEvolutionScheduler(ctx context.Context, interval time.Duration)` 方法。
  - 内部起一个 `goroutine`，按照给定的时间间隔（如每天凌晨）自动调用 `m.EvolveSkills()`，完成劣质技能的软淘汰。

---

## 🎯 模块四：`@pkg/memory` 与幻觉消除机制 (Agent 记忆体抽象)

### 4.1 废弃独立 RAG 接口 (已完成 ✅)
- **动作**：删除游离的 `pkg/rag` 目录。
- **原因**：RAG 只是一项技术（向量检索），而它在系统中的应用学名叫做“记忆 (Memory)”。

### 4.2 定义 `MemoryBank` 与三模态记忆体系 (待开发 ⏳)
- **任务目标**：将记忆抽象为 Agent 独有的属性（即 Agent = Prompt + Reactor + MemoryBank）。
- **具体实现步骤**：
  - 在 `pkg/memory` 下定义 `MemoryBank` 接口，它应组合三种记忆：
    1. **短期记忆 (Working Memory)**：接口提供 `RecallContext(intent)` 和带权重调整的 `Update(key, deltaWeight)`。用于记录会话经验与白名单授权。内部支持基于 $e^{-\lambda t}$ 的时间衰减洗牌机制。
    2. **长期知识库 (Semantic Memory / RAG)**：提供只读的 `RecallKnowledge(intent)`，由应用层挂载基于外部 GraphRAG 或 AdvancedRAG 的实现。
    3. **肌肉记忆 (Muscle Memory)**：提供 `RecallExperience(skillName)` 和 `DistillExperience(skillName, newSOP)`。用于存储并召回大模型在执行 Skill 过程中“拨乱反正”后蒸馏出的成功经验捷径。
  - 修改 `agent.Agent` 的装配工厂，允许给 Agent 挂载其专属的 `MemoryBank`。
  - 在 `engine.Reactor` 的初始思考流（Thinker Hook）中，依次调用这三种 Recall 接口，将结果以绝对事实和最佳实践的形式注入到 System Prompt 头部，从根源上锚定模型认知并加速任务执行。

  ### 4.3 终极进化：著书立说与开疆拓土 (Knowledge Crystallization) (待开发 ⏳)
  - **任务目标**：完成经验的反写与**全新技能**的自动涌现。
  - **具体实现步骤**：
    - **现有技能打磨 (Refinement)**：扫描积累的同一任务下的高权重“肌肉记忆”。当肌肉记忆的触发次数与信度达到设定阈值时，利用大模型对这些历史经验进行二次总结蒸馏，将提炼出的最优大纲与排坑指南直接覆写到本地的 `SKILL.md` 文件中。
    - **新技能涌现 (Discovery)**：在后台分析长期沉淀的“短期会话记忆”，寻找用户高频请求但缺乏专属 Skill 的长尾工作流。大模型自动总结这些散落的交互指令，无中生有地撰写出一份全新的 `NEW_SKILL.md`，注册进 Skill Manager。
    - 随后，清空相关的低级短期记忆与过渡态肌肉记忆，释放上下文空间，完成认知维度的全面跃迁。
  ---

  ## 📅 执行优先级建议
  1. **P0 (Blocker)**: 完成「模块二：Agent 装配工厂」。因为如果没有这个，外部应用根本无法正常使用这个框架，所有的 Example 都是手拼的脏代码。
  2. **P1 (Core Feature)**: 完成「模块一：基于 Hook 的 HITL 阻断机制」。这是构建安全可信 Agent 的护城河。
  3. **P2 (Enhancement)**: 完成「模块三：Skill 数据上报与调度器」。补全系统的“自我进化”能力。
  4. **P3 (Ultimate)**: 完成「模块四：Agent 记忆体抽象与知识固化」。实现从短期记忆到肌肉记忆，再反写 `SKILL.md` 的终极进化闭环。