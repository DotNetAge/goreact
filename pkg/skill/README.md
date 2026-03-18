# Skill 系统设计与执行原理

本目录 (`pkg/skill`) 实现了基于 [Agent Skills 规范](https://agentskills.io) 的技能管理与演化系统。

在回答“Skill 是如何被分解成多个处理环节并执行”的这个问题之前，我们需要先纠正一个传统软件工程的思维定势：**在 GoReAct 中，Skill 的任务拆解不是由 Go 代码（如硬编码的 DAG 图或状态机）完成的，而是由大模型（LLM）的认知能力通过 Prompt 驱动完成的。**

## 1. 什么是 Skill？（SOP 数据包）

在 GoReAct 架构中，`Skill` 并不是一段可编译的二进制可执行代码，而是一个高度结构化的 **SOP（标准作业程序）数据包**。

一个标准的 Skill 实体包含：
- **元数据 (Frontmatter)**：名称、描述、兼容性限制、允许调用的基础 Tool 列表。
- **核心指令 (Instructions)**：这是核心！它是用 Markdown 编写的、写给大模型看的**分步操作指南**。
- **上下文外挂**：
  - `scripts/`：Skill 专属的辅助脚本代码。
  - `references/`：Skill 专属的参考文档（供 RAG 使用）。
  - `assets/`：静态资源。

## 2. 运行原理：认知流拆解 (Cognitive Decomposition)

当系统需要执行一个复杂 Skill 时，具体的运行原理和步骤如下：

### Step 1: 动态检索与组装 (RAG Agent Manager)
当应用层下发一个模糊任务（例如：“帮我清理并分析这份销售数据”）。
`skill.Manager` 利用 LLM 语义匹配，从技能库中“检索”出最匹配的技能：`DataCleaningSkill`。

### Step 2: 上下文注入 (Context Injection)
引擎并不会把 `DataCleaningSkill` 丢给某个代码执行器，而是**动态构建一个 Sub-Agent（子智能体）**，并将该 Skill 的 `Instructions`（Markdown 步骤指南）和 `References` 完整地无缝注入到这个子智能体的 **System Prompt** 中。

*提示词示例片段：*
> You are equipped with the DataCleaningSkill. 
> To execute this skill, you MUST strictly follow these steps:
> 1. Use the 'read' tool to inspect the CSV header.
> 2. Use the 'bash' tool to run the python script provided in your scripts directory to drop NaN values.
> 3. Use the 'calculator' tool to compute the average of the "Sales" column.

### Step 3: ReAct 循环执行 (The ReAct Loop)
这是真正的拆解发生的地方。大模型（Thinker）读取了被注入的 System Prompt。
由于大模型具备强大的上下文理解与遵循能力，它会在 ReAct 的多次迭代中，**自觉地、一步一步地**按照 Skill 定义的环节生成 `Thought` 和 `Action`。
- **Loop 1**: Thinker 意识到需要执行第 1 步，下发 `Action: read`。Actor 执行后 Observer 返回数据。
- **Loop 2**: Thinker 根据观察结果，进行下一步推理，发现必须执行第 2 步，下发 `Action: bash`。
- **Loop N**: Thinker 发现指南中的所有步骤均已完成，下发 `Final Answer`。

## 3. 为什么采用这种设计？

这种设计的优势是极其巨大的：
1. **降维打击的极度解耦**：如果使用代码去定义 DAG（先执行 A 再执行 B，如果失败执行 C），那开发者必须为每一个 Skill 编写复杂的 Go 代码逻辑（参考已被删除的 `Coordinator`）。而现在，**任何人只需要写一篇 Markdown 就能教会 Agent 一个新技能**。
2. **柔性容错**：在严格的 DAG 中，如果步骤 2 失败（比如文件格式稍有不同），程序直接崩溃。而在 Prompt 驱动的 ReAct 循环中，如果步骤 2 报错，Observer 会把错误信息告诉 Thinker，Thinker 会根据报错自行思考（“哦，文件编码是 GBK，我换个参数重试”），体现出了真正的“智能体”韧性。
3. **分形结构闭环 (Agent-as-a-Tool)**：一个装载了特定 Skill 的 Sub-Agent，对外可以被直接包装成一个简单的 `tools.Tool` 暴露给 Supervisor Agent。形成无限嵌套的超级大脑。

## 4. 技能蒸馏与经验固化 (Skill Distillation & Muscle Memory)

这是一个极为前沿的设计洞察：**`SKILL.md` 只是新手的“工具说明书”，而高级 Agent 应该拥有“肌肉记忆”。**

如果每次执行任务，Agent 都要把长达几千 Token 的 `SKILL.md` 全文读一遍，再去磕磕绊绊地尝试，不仅耗费大量算力，而且极易因为大模型的注意力偏移而犯错。

在 GoReAct 的进阶设计中，Skill 具备**“经验固化（蒸馏）”**的能力：
1. **初见 (Reading the Manual)**：Agent 第一次接触某个 Skill，完整阅读 `SKILL.md` 并艰难但成功地完成了任务。
2. **反思与蒸馏 (Distillation)**：在任务成功后（结合 `Observer` 和 `Terminator` 的确认），Agent 会对刚才的成功路径进行“反思”。它提取出最关键的步骤、参数捷径和避坑指南，生成一段极其精炼的**“成功经验 (Distilled SOP)”**。
3. **肌肉记忆 (Muscle Memory)**：这段被高度压缩的经验会被写入 Agent 的 `MemoryBank`（作为与该 Skill 绑定的永久记忆，并随着后续调用不断 `Update` 其权重）。
4. **再次调用 (Mastery)**：当 Agent 下次再被要求使用该 Skill 时，系统优先从 Memory 中召回这段“肌肉记忆”而非原始的说明书。Agent 凭借成功经验，可以极速、精确地完成操作。这极大地降低了 Token 消耗，并使 Agent 表现得越来越“老练”。

## 5. 扩展性设计：Manager 接口

系统为了实现最大的定制灵活性，在设计 `skill.Manager` 时采用了经典的“面向接口编程”策略。

### 5.1 基础接口与默认实现
`pkg/skill/manager.go` 中定义了一个强大的 `Manager` interface，同时提供了一个基于内存的 `defaultMgr` 作为“部分实现”（即开箱即用的基础实现）。
这个 `defaultMgr` 实现了技能加载、内存缓存、关键词/语义混合匹配（Hybrid Selection）以及基础的打分演化逻辑。

### 5.2 为什么需要应用层继承与扩展？
真实的生产级 Agent 平台，技能往往不是存在本地硬盘或内存里的。
如果你希望将 GoReAct 集成到更庞大的企业系统中，你（应用开发者）应当实现自己的 `Manager`（或采用组合模式包装 `defaultMgr`），以实现以下高级特性：
1. **分布式存储**：重写 `LoadSkill` 和 `GetSkill`，从 Redis、PostgreSQL 甚至是 AWS S3 桶中动态拉取技能。
2. **多租户隔离**：重写管理逻辑，根据不同的用户或租户 Token 隔离可见的技能池。
3. **企业级向量检索 (RAG)**：重写 `SelectSkill`，不再使用简单的本地匹配或单次 LLM 问询，而是对接专业的向量数据库（如 Milvus / Qdrant）进行海量技能的精准 Embedding 匹配。

因此，当前的 `defaultMgr` 只是一个极其坚实的骨架和 MVP，真正的生产级扩展完全交由应用层自主定义！

---

## 6. 当前代码库落实状态预警（审计发现）

**注意：当前框架中“优胜劣汰”的机制定义已完成，但尚未与核心执行引擎（Reactor）闭环。**

通过代码审计发现，`skill.Manager` 内部提供了精妙的统计结构和 `RecordExecution`、`EvolveSkills` 方法。但目前的 `engine.Reactor` 的 `Run()` 方法执行结束后，**尚未自动回调** `RecordExecution` 去登记执行数据，也没有后台协程（Scheduler）去定期触发 `EvolveSkills()`。

**Phase 4 待办补全项**：
这需要我们在后续重构中：
1. 在 `Reactor.Run()` 的终止阶段，如果上下文中挂载了 Skill，自动将 `reactCtx.TotalTokens` 和耗时反馈给 SkillManager。
2. 实现一个背景轮询任务或触发器，定期执行技能池的“大清洗”。
