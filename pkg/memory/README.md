# 记忆系统 (Memory System) 架构设计

本目录 (`pkg/memory`) 负责定义 GoReAct 架构中极其核心的“仿真记忆系统”。

## 1. 核心洞察：仿真记忆系统 = 短期记忆 + 永久记忆 + 经验记忆

在重新审视了 Agent 消除“幻觉 (Hallucination)”的需求与技能演化机制后，我们得到了一个终极的仿生学洞察：**Memory 与 RAG 并非孤立的系统，它们是“记忆”这一宏大概念的不同维度。**

一个真正成熟的 Agent 记忆系统，应该完美映射人类的认知模型：
- **短期记忆 (Working Memory)**：源自当前或近期会话的交互上下文、人类白名单授权、临时状态。轻量级 KV Store 即可满足。主要用于**维持对话连贯性和短期约束**。
- **永久/知识记忆 (Semantic Memory / RAG)**：Agent 赖以生存的基础世界观与外部事实（如企业内部代码库、财务报表）。由底层的 AdvancedRAG 或 GraphRAG 提供支撑。主要用于**消除事实类主幻觉**。
- **经验/肌肉记忆 (Procedural/Muscle Memory)**：Agent 在过往执行 `Skill` 时蒸馏沉淀下来的“最佳实践”与“避坑指南”。主要用于**消除操作类幻觉，加速任务执行**。

---

## 2. 谁拥有记忆？Agent！

在面向对象的逻辑中，既然记忆是为了消除大模型的幻觉和提升效率，那么**记忆的宿主必须是 Agent**。

因此，在我们的架构里：
**Agent 实例 = 角色设定 (Prompt) + 大脑 (Model/Reactor) + 记忆体 (MemoryBank)**

### 2.1 记忆体 (MemoryBank) 的终极抽象

`MemoryBank` 是挂载在 Agent 上的顶层聚合体，它内部协调三种截然不同的记忆引擎：

```go
type MemoryBank interface {
    // 1. 获取近期会话相关的强相关经验与约束 (Short-Term)
    RecallContext(intent string) (string, error)
    
    // 2. 获取背景知识层面的重磅专业资料 (Long-Term / RAG)
    RecallKnowledge(intent string) (string, error)

    // 3. 获取肌肉记忆：提取该任务之前被蒸馏/修正过的最佳操作流 (Muscle Memory)
    RecallExperience(skillName string) (string, error)
    
    // Agent 自动写入近期状态或更新经验权重
    Update(key string, deltaWeight float64) error
    // 蒸馏并固化一条操作经验
    DistillExperience(skillName string, newSOP string) error
    }
    ```

    ---

    ## 3. 架构大闭环：从“开卷说明书”到“老司机”的蜕变

    通过将 Skill 演化与 MemoryBank 深度绑定，GoReAct 实现了 Agent 认知的完美进化链路：

    1.  **初见 (Level 1: Standard SOP)**：Agent 接收到一个从未运行过的 `Skill`，它像新手一样完整阅读 `SKILL.md`（开卷考试），依赖 **Long-Term Memory** 提供的背景知识，在 ReAct 循环中小心翼翼地探索。
    2.  **纠偏 (Level 2: Error Reflection)**：如果 Skill 定义有瑕疵（说明书有坑），Agent 会在执行中碰壁。此时通过 `Observer` 的反馈，Agent 在 `Thinker` 阶段实现“拨乱反正”，寻找到了真正的成功路径。
    3.  **升华 (Level 3: Distillation)**：任务成功后，由 `Terminator` 触发经验蒸馏。Agent 将刚才“碰壁 -> 思考 -> 修正”的过程浓缩为一条极简的捷径指令（Success Shortcut），并存入 **Muscle Memory (经验记忆)**。
    4.  **老练 (Level 4: Mastery)**：当再次遇到相同任务，Agent 优先唤醒 **Muscle Memory**。它不再需要阅读冗长的原始说明书，而是像“老司机”拥有肌肉记忆一样，绕过陷阱，直达目标。
    5.  **著书立说 (Level 5: Skill Refinement)**：这是对**现有技能**的打磨。Agent 周期性地“温故而知新”，将积累的“肌肉记忆”二次蒸馏，反写回原始的 `SKILL.md`，完成对既有 SOP 的优化与固化。
    6.  **开疆拓土 (Level 6: Skill Discovery)**：这是认知的最高维度——**从日常对话中涌现新技能**。当 Agent 发现自己在处理某些意图（Intent）时频繁地手动组合了一系列基础工具，且这种组合模式在短期记忆中反复出现并被验证有效时，Agent 会主动提议：“这似乎是一个新的模式”。它会自动撰写一份全新的 `NEW_SKILL.md`，从而拓展其能力的哲学边界，从“执行者”进化为“创造者”。

    **这套三模态记忆体系（短期+永久+经验）不仅从根源上消除了大模型的“事实幻觉”与“操作幻觉”，更赋予了 GoReAct 框架真正意义上的“自进化”生命力。**

    ---

    ## 4. 短期记忆的演化与洗牌 (Weight Decay)

    短期记忆接口向 Agent 提供了一个 `Update` 暴露点：
    ...
- **动态权重**：当 Agent 在执行任务时，如果在 `Observer` 发现某种路径屡试不爽，或人类赋予了某个工具权限（加白），Agent 应调用 `Update` 增加这条记忆的权重。
- **自然衰减 (Time Decay)**：短期记忆不应该被直接硬删除。通过引入时间衰减函数（如 $e^{-\lambda t}$），长期未被激活的记忆权重会无限接近于 0。当发生 Memory Compress 或 RAG 检索时，这些死去的记忆自然就会被淘汰出局，从而保证 Agent 上下文的极度纯净且不会被塞爆。

---

## 4. 架构闭环：消除幻觉的最终管线

当这个“仿真记忆体”挂载到 Agent 后，整个系统的工作流形成了一个完美的极简闭环：

1. **[接收任务]**：用户发来意图 `Intent`。
2. **[唤醒记忆]**：Agent 在启动 `Thinker` 之前，向自己的 `MemoryBank` 发起询问。
   - `ShortTermRAG` 召回了：“昨天用户刚骂过我不要用 JSON 格式输出。”
   - `LongTermRAG` 召回了：“该公司的最新退货政策是 7 天无理由。”
3. **[思维注入]**：这些被检索出的高权重记忆被作为不可挑战的事实（System Note）死死地锚定在 Prompt 的最前方。
4. **[开始推演]**：此时大模型（Thinker）再开始输出 Thought，彻底告别了“一本正经胡说八道”的主幻觉。