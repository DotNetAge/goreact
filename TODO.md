- [x] Clude的任务编排机制是什么？是ReAct吗？
- [x] SubAgent 的分派与执行机制的运行原理是什么？
- [x] Clude查找文件作为参考的机制是怎么运作的？
- [x] Clude执行工具的流程是怎么样的？ 
- [x] 当工具执行的返回结果是一个非常长的字符串时，这个字符串很可能会塞爆上下文窗口应该怎么处理？
- [x] Skill 是怎么被执行的? 工作原理是什么？
- [x] CludeCode 的提示词工程做得相当好，而且非常出色，我想知道它们到底有些最具特色的地方。

---

- [x] 是否将内建立Prompt全部采用 Go Template 技术可以增加执行效率？
- [x] 我设计的ContextWindow是想将其注入到Agent中，使其持有其实例，以隔离不同会话间的上下文内容，也可以将上下文内容进行外部保存或加载处理，也可以对有限长度的上下文窗口进行压缩或剪枝以避免上下文爆炸。
- [x] 我如何能为实现TODO与按照TODO执行的能力？
- [x] 当一个任务需要查看大量的文件时应该如何处理呢，如果将全部文件一次过作为附件加载到上下文之中，是否会塞爆上下文？
- [x] 当主任务与子任务同时在思考时如何能让客户端能同时显示它们的思维链与流式输出？
- [x] 当Think阶段(意图识别)发现问题需要澄清时，应该如何与客户端交互？流程是如何，因为澄清时当前执行的流程就会被打断，当客户端回复问题后又要将流程恢复至原来的状态，并将用户的问题接上上次的聊天的上下文；
- [x] 与上一问题相关的是当Action在执行工具时一定会遇到"高风险"的工具需要用户授权，其过程也就是"打断"->"唤醒"->"接受新答案"->"继续执行"，那这个过程应该如何设计？
- [x] 我想在T-A-O中加RAG以抑制大模型的幻觉，因此才提供了一个简单的Memory接口，由外部调用者来实现这个Memory，既能用简单的数据库方式也可以采用RAG的方式；
- [x] 为解决上下文爆炸或上下文过长的问题的根因，我是想通过Memory(RAG)实现动态上下文模式，每次只从记忆中获取与当前对话主题相关的有限上下文，而不是单纯按时间顺序获取全部上下文，而ContextWindows 与 Memory之间关系与交互处理就需要进行更深入的处理。

--- 

- [x] 如何向客户端输出 Execution Summary, 如：工具数的使用，耗时，总Tokens消耗。
- [x] 是否有必要需要增加文字汇总工具, summarize ？无必要LLM本身就可以完成这个功能。

---

- [x] 重构Skill系统，实现两种Skill的加载方式，两种方式都是基于文件系统方式加载Skill.md文件，可支持指定Skill目录的加载，第二种则内置Skill加载，内置Skill可以用go的embed包实现。
- [X] 团队工具：SendMessage, TeamDelete ? 这些工具是可以辅助SubAgent的协作吗？

---

- [x] 整理所有内置工具，全部工具都应该存放于 tools 目录
- [x] 增加一种可以自动编写Skill的Skill或者工具；
- [x] 完全仿照Clude的WebSearch的实现方式重构 WebSearch与WebFetch工具；

---

- [x] 提供一份完整的GoRact功能描述，纯文字讲述 GoReact提供了什么功能，可以实现什么，最大的特色是什么；
  - GoReact 的设计目标是希望能帮助开发人员专注于Tools与Skills的开发与运用，内核机制与性能由GoReact负责，保证以最少量的Token收获最大的价值。开发人员则在垂直领域中提供各种各样不同的Tools和Skills才能让AIAgent能在垂直领域中创造最大的价值。
- [x] 补充如何使用 Agent 快速实现智能助手系统（无Memory版,只配置API_KEY就马上可以运行）
  - 我们可以提供一套 `DefaultAgent()` 生成一个基于 qwen3.5-flash 的AgentModel
  - 提供一个 `DeaultModel()` 生成一个基于 qwen3.5-flash 的默认 Model 配置
- [x] 补充GoRact的Tools的开发指南
- [x] 补充GoRact的Skills的开发指南
- [x] 补充GoRact的MCP的开发指南
- [x] 补充一份GoRact通信指南，包括但不限于以下的内容：
  - 如何获取思想流
  - 如何获取结果注
  - 如何获取Token消耗情况
  - 如何获取上下文窗口的状态
- [x] 当启用团队模式研究一个问题时，主Agent会干什么？空置？
---

- [x] 从入口开始Agent就附带了SystemPrompt，而这个SystemPrompt应该是每轮对话中都必须写LLM的，我们当前的代码是这样的吗？
- [x] SubAgent 是通过 "subagent" 工具唤醒的吗？我们是否要提供一份内置工具清单？
- [x] 对于subagent 的命名可以使用 `@{name}` 的形式，这样有利于LLM识别出subagent的名称，以便于在Skill中进行引用。
- [x] Skill 的描述是否采用 XML 格式效果会更加好？
- [ ] 如果在Skill中启用了Cron工具，那么是否就可以具有计划任务能力？如果是这个计划任务被唤醒时是怎么调起Agent呢？
- [x] Skill.md 内的变量：`{base_dir}` 这是Skill目录地址读入Reactor在执行之前要将这个变量格式化掉
- [x] 增加新的开关，ReActor 要设置一个 IsLocal 的属性，这个属性是由Agent的Model中带入的，如果这个属性为真，所有的LLM调用都必须采用串方式调用，而不能采取当前并发模式。但如果SubAgent的ReActor的`IsLocal`为假时则可以继续采用并发模式，这样做的目的是可以支持"本地+云端"的混合调用模式。如果全部不采用云端则整个Agent项目都会以LLM的单线程模式运行。
- [x] 用户如何打断正在执行T-A-O流程，在打断后如何让其重新执行？
- [x] 重新设计ReActor的主测试（端到端），要完全覆盖功能描述中的全部内容与场景；
- [x] 当Agent完成一个完整的任务后应该输出一段任务的总结，对于团队协作则要收集全部Agent的处理总结汇总后写总结；

---

- [x] 增加"滑动窗口"的机制, `ContextWindow`定义了"短期记忆"，是大模型完成一次完整对话的上下文窗口；当前的上下文窗口只是一个临时变量，在对话完成后会被清空，下一次对话又会从头开始。
  - 增加SessionStore接口，用于存储与恢复对话上下文窗口，可以采用文件存储或数据库存储，也可以采用RAG存储。
  - 框架内默认提供 `MemorySessionStore` 与 `BoltSessionStore` 两种实现，`Agent` 在没有设置 `SessionStore`实现时 Fallback 到 `MemorySessionStore` 为默认会话实现；
  - 每次向LLM发送请求时，都要将当前的`Message`存入至`SessionStore`
  - 当 `ContextWindow` Tokens 的大小达到其最大容纳边界时，会触发【滚动】，有了`SessionStore`的加持后被就不会丢失对话内容，当内容被【滑出】上下文窗口后，可以触发`SessionStore`上的"滑动"方法，被滑出的一条或多条消息在掉落出上下文窗口时，客户端可以将这些被滑出的内容存入RAG或其它存储进行语义化成为"知识"，在用户当前上下文中按语义反向注入，这就避免了"上下文腐烂"的问题，同时也可以支持了无限的上下文，以及长期记忆与短期记忆的完美融合。
- [x] ReActor 在Think阶段没有对工具进行过滤，而是全量加载工具这样可能会导致Token消耗瓶颈，需要考虑按意图过滤工具的能力，以及使工具与技能的过滤都可以支持语意化方式筛选过滤；
- [x] 不应该在TAO中直接保存经验，而应该由客户端去处理，客户端是通过评估机制来处理这个问题；
- [x] 初始化callLLM时要对齐全新的ModelConfig的定义
- [x] 在ReActor反向引用了Tool中的内置方法，这种做法有点本末倒置，感觉上是限制了工具机制的通用性与灵活性，就不能仍然沿用 Skill 与 Tool 的机制实现类似  AskUser 和 AskPermission 这类的操作吗？
- [x] 为`Agent` 增加一个`Switch`方法，用于切换成其它的Agent，切换的内容仅限于 AgentConfig 与 Model,其它的初始化参数不变；
- [x] Think阶段中当选中某个技能时，如果这个技能中的Allows_Tools属性长度>0时，那当前上下文加载的Tools就只能是AllowTools中限定的工具，因为第一次是全量加载工具，而这一次则是过滤应该加载相关适用的工具。
- [x] 要核查当技能被激活后，如果技能中明确说明需要采用多Agent协作，T-A-O是否能正确调起多Agent进行协作？
- [x] **重点**: 当SubAgent被创建时，SubAgent的全部参数除了 AgentConfig 与 Model 之外，其它的一切参数都应从父Agent中继承，或者说`Agent`类要提供一个`Clone`方法，用`Clone`的方式来创建子Agent，同时`Switch`方法让Agent切换身份更换Model.
- [x] **重点**:当Skill完全被激活时，如果Skill中有明确说明调用Skill目录下的 `scripts`的脚本时，如 `python m.scripts/<function_name>`, 那么这就需要解决python环境的问题，以及如何让python环境能找到脚本文件。
  - [x] 采用全局环境还是采用虚拟环境？→ 每个Skill目录独立虚拟环境(.venv)
  - [x] 是在Skills的根目录下 `.../skills` 还是在Skill自身目录下建立虚环境？→ Skill自身目录
  - [x] scripts中引用到的依赖应该如何安装？→ 自动检测requirements.txt变化后pip install
  - [x] 遇到python脚本当前项目是否会默认采用 Bash 运行脚本？→ ScriptExecutor通过.venv/bin/python执行
  - [x] 想办法建立一个集成测试证明python脚本可以正确执行？→ core/script_executor_test.go 3个测试PASS
- [x] 增加 `Rule`,`RuleRegistry` 用于定义Agent的行为规约，Rule的注册可以有内部的，也可以在客户端暴露界面给用户去设置自定义的Rule，所有的Rule条目都会被应用Agent当中。Rule 应该有一个比较重要的属性，就是"Scope"(适用范围)，可以有 Global(全部Agent) , Local(当前Agent), Conversation(当前会话)三种选择。
  - [x] 向LLM发起会话时就要检查是否应向SystemPrompt插入Rules；

---
- [ ] 补充关于"Tools VS Skills"的文章，论证为何Skills会更优于Tools（由用户手动撰写）
  1. 对于绝大多数大模型，Tools并无法实现延时加载，而一次性加载更多的Tools会使模型的注意力下降，Token消耗增加。
  2. Skill 采用三级"渐进式加载"机制，当Skill被激活时才会从`AllowedTools`中加载工具，相比之下会降低思考链的的复杂度，同时也可以降低Token消耗；
  3. 对于实用场景如文件内容查找，网络搜索往往不是只靠一个工具完成，而是依赖于多个工具同时配合协同完成，因而基础Skill会显得更加具有灵活性与可扩展性。
  4. 增加Tools的代价比增加Skill的代价要高，每个Tools的加入就意味着要重新编译，而Skill仅是复制文件；
---

- [x] 到底是将ScriptExecutor放在core还是应该实现成为一个Tool?进入core就意味着外部第三方客户端可以进行开发与自定义，而且耦合度是最高的。是否这个Executore被Reactor在Skill的第三阶段被调用了呢? 如果是的话是一个比较大的问题，我更加倾向于将其制作成为 Tools，这会更加轻量而且必须要能支持 LLM 解释 Skill 中说明的 `python ...` 执行指令。
- [x] 会话应该对应角色，会话不应该在不同的Agent之间共享，这很容易产生混乱，因此会话本身就需要有角色的标识，用于区分不同的会话。这就要对会话机制进行重构。
  - ContextWindow 增加 Role 字段标识 Agent 身份
  - SessionStore 接口增加 SessionInfo、GetByRole、ListSessions 方法
  - MemorySessionStore 实现完整的 Role 隔离机制
  - NewSession 自动绑定 Role 到会话
- [x] 会话管理应该增加获取当前会话的方法，`GetSessionByRole(role string)` 该方法会获取时间最新会话，由其是在使用 Agent.Switch 方法切换身份时不是直接创建会话而是切换至最新的会话，会话管理内部会检查是否有最新会话，没有的时候才会创建。
  - Switch 方法重构：优先通过 GetByRole 获取目标角色的最新会话并恢复上下文
  - 仅在无历史会话时才创建新会话
  - 新增 Agent.GetSessionByRole()、Agent.Role()、Agent.ListSessions() 公开方法
- [x] 将T-A-O的最大对话轮次调整到30轮；（已确认 DefaultMaxSteps = 30）
- [x] 补充实用型的基础Skill
  - [x] 技能：文件内容查找 AllowedTools: [Glob, FileRead, Grep] → reactor/skills/file-search/SKILL.md
  - [x] 技能：网络搜索 AllowedTools: [WebSearch, WebFetch] → reactor/skills/web-search/SKILL.md
  - [x] 重新规划最常用的Skill，排除不常用的Tools，删除作用不大的Tools
    - **移除 5 个冗余 Tool**: echo, calculator, ls, repl, replace（LLM 可直接输出/计算/glob 覆盖/bash 替代）
    - **bundledTools 从 21 个精简为 16 个**，按功能分组注释
    - **新增 code-edit Skill**: AllowedTools [read, glob, grep, file_edit, write]
    - **新增 task-manager Skill**: AllowedTools [todo_write, todo_read, todo_execute]
    - **当前内置 Skill 总数**: 4个 (file-search, web-search, code-edit, task-manager)
    - **保留原则**: Tools 仅在无法用 Skill 实现时才开发（如基础设施级操作）

---

- [x] Tokens 计算集中化处理
    - **问题**: `Reactor.tokenEstimator` 默认使用 `DefaultTokenEstimator(3.0)` 粗略启发式 (`len/3.0`)，而 `core.EstimateTokens()` 使用精确 tiktoken BPE 分词，两条路径并存导致预算追踪不准
    - **修复**: 将 `defaultTokenEstimator` 重构为委托到 `EstimateTokens()` (tiktoken BPE + fallback)，统一所有 Token 计算入口
    - **影响范围**: `core/compact.go` (TokenEstimator 实现 + 测试)、Reactor 全链路 (Think/Session/LLM builder)
    - **向后兼容**: `NewDefaultTokenEstimator(charsPerToken)` 签名保留但参数不再使用

---

- [ ] reactor.go L21 	MaxHistoryTurns = 30 根据Model的MaxTokens动态计算发向LLM的最大轮次.
- [ ] reactor 没有向外部提供Logger接口，是如何可以实现更换Logger的？
- [x] 功能设计问题：当Agent正在执行的时候，是否还能向Agent发送消息？
  - Agent空闲，而SubAgent正在执行，是否还能向Agent发送消息？
  - 当Agent在等待LLM完成执行时，是否还能向Agent发送消息？实现【紧急叫停】功能；
  - 如果SubAgent正在执行，不能够切换身份，只能等待执行完成；
- [x] 当切换Agent的时候Skill是不共享的，因为每个Agent有自己领域的Skill，因此不能共享。只能通过输入该Agent定义的Skills目录来加载Skill。
- [x] 在渐进式披露过程中，当进入Skill激活阶段时，
  - 如果在 Skill 中需要引用其它的Skill的时候，应该如何处理？这是涉及跨Skill的引用，所以在SystemPrompt的skill的元数据列表是不可以不加载的；同样地，全量的工具元数据也不能不加载（但这不就破坏了原有定下的原则）？
    - 解决办法：当遇到子Skill时，可以利用当前的子Agent编排能力，克隆一个当前的Agent, 然后构造一个任务上下文给该子Agent，主Agent则处于等待状态，直到子Agent完成后，主Agent获取其结果后再继续执行。此方法需要关注以下的问题：
      - 子Skill是怎么从主Skill的正文中被识别出来的，至少是正确的名字；
      - 顺带就要检查一项当前Think是怎么样从Skill的正文中识别子需要调用某个指定名子的子Agent的；
      - 值得研究：由于Skill的嵌套调用可能会导致产生一条很长的"调用链"，Gemini与LangChan据说是采用Graph的方式先构建起一个完整"图"，然后再原子化调用，表达出来的貌似可行但过于抽象，不知道是否可以实现，又应如何实现？
  - 如果在 Skill的描述中需要进一步让LLM阅读 Skill中的 references 目录中的各类指引或参考文件时应该如何进入第3阶段更深层次的内容披露？
    - [x] 这里需要增加一特殊的识别，可以借助LLM的推理能力让LLM识别出当前Skill是要加载 references 内的参考文件如："请参考 references/guide.md" 文件，此时就会激会第三阶段的Skill内容披露, 我认为可以分成两类处理：
      - [x] 如果是文本文件就直接读取文件内容插入到当前会话上下文内"<references>[文件内容1] ...</references>"
      - [x] 对于文件过长暂时没有想到有什么办法，因为可能会塞爆当前上下文。先做[TODO]标记待有方案再议；
      - [x] 如果文件是二进制文件则直接将文件名，大小 作为引用列表插入至当前上下文"<referece-links>文件全路径 ，大小 </reference-links>"
      - [x] 如果是非markdown类型的文件则直接放弃读取，因为这并不符合Skill的定义规范；或者将来可以考虑采用文件分析器来扩展对不同文件格式的文本类文件转换成统一的Markdown文件；
    
---

## 编排器智能化改造 (Design §6 / §8 / §12) — 2026-04-30

- [x] **LLM Router 实现** (`orchestration/router.go`)
  - LLM 驱动的智能路由引擎，语义匹配任务→Agent (§6.3)
  - 轻量级 prompt 模板（仅加载 Name+Description，不加载 Body）
  - JSON 输出解析：`RoutingDecision{SelectedAgent, Reasoning, Confidence}`
  - 带缓存层（TTL=10min，避免重复 LLM 调用）
  - 置信度阈值机制：<0.4 自动降级为 CREATE_NEW
  - **关键词 fallback**：LLM 不可用时基于规则的三级降级策略
    1. Description 关键词命中 + 绩效加权
    2. 最高分空闲 Agent 选择
    3. 无可用 Agent → 建议动态创建
  - Markdown 包装 JSON 自动剥离

- [x] **Agent Factory 实现** (`orchestration/factory.go`)
  - 动态 Agent 创建工厂，Router 返回 `__CREATE_NEW__` 时触发 (§12)
  - 两阶段 LLM 生成：Description（≤1024字，面向路由）+ Body（完整指令，面向执行）
  - 能力需求提取 → Skill 匹配 → 配置构造全流程
  - 重叠检测：复用已覆盖能力的现有 Agent（避免重复创建）
  - 数量上限保护（默认 20 个）
  - 规则降级生成：LLM 不可用时自动生成基础配置

- [x] **ScoreTracker 接入** (`orchestration/score.go` 已有，现接入使用)
  - `ChannelOrchestrator` 结构体新增 `router`, `factory`, `scoreTracker` 字段
  - `handleResult()` 新增 `recordScore()` 调用：任务完成时自动记录 0-3 分制绩效
  - 分数同步更新到 RuntimeDirectory 的 Score 字段 + TaskCount 计数
  - 冷启动信任分：动态创建的 Agent 初始 Score = 2.0

- [x] **RouteTask() 智能路由公开方法**
  - 新增 `RouteTask(ctx, taskDescription, desiredCapability, parentID, metadata)` 方法
  - 与现有 `DelegateTo(agentName, ...)` 形成互补：
    - `DelegateTo`: 显式指定 Agent 名称（向后兼容，原有路径不变）
    - `RouteTask`: 不指定名称，由 Router 自动选择/创建最佳 Agent（新增智能路径）
  - 完整流程：收集候选 → LLM/规则路由决策 → 选择/创建 → DelegateTo 委托执行

- [x] **新增构造选项**
  - `WithLLMRouter(router)` — 注入自定义 LLM Router
  - `WithAgentFactory(factory)` — 注入自定义 Agent 工厂
  - `WithScoreTracker(tracker)` — 注入自定义绩效追踪器
  - 自动化：提供 WithDefaultModel 时自动创建 Router + Factory + ScoreTracker

- [x] **新增消息类型**
  - `RouteTaskRequest` — 智能路由请求结构体
  - `DelegateRequest.AgentName` 为空时的语义变更为"触发智能路由"

- [x] **测试覆盖** (`orchestration/router_test.go`)
  - Router: 11 个测试（创建、空候选、fallback 关键词/busy 跳过、缓存、JSON 解析、低置信度、Markdown 包装、分词）
  - ScoreTracker: 6 个测试（记录/查询、epsilon-greedy 优选、单候选/空/全部）
  - 集成: 2 个测试（组件自动创建、RouteTask fallback 路由端到端）

### 当前编排器 vs 设计文档对照表

| 设计组件             | 设计文档                            | 当前状态                                      |
| -------------------- | ----------------------------------- | --------------------------------------------- |
| **LLM Router**       | §6.3: 完整的 LLM 语义匹配引擎       | ✅ 已实现（含缓存+fallback）                   |
| **Dispatcher**       | §6.4: Router→选择→状态更新→转发     | ✅ 已实现（handleDelegate + RouteTask）        |
| **AgentFactory**     | §12: 动态创建 + 两阶段生成          | ✅ 已实现（含重叠检测+降级）                   |
| **ScoreTracker**     | §8: 0-3分 + EMA衰减 + 多因子排序    | ⚠️ 基础版接入（多因子排序待集成到 Dispatcher） |
| **Coordinator 模态** | §4.3/§10: Executor↔Coordinator 转换 | ❌ 未实现（下一阶段）                          |
| **事件类型全集**     | §7.1: TaskDispatchEvent 等          | ⚠️ 部分已有（core 层已定义）                   |

### 待实现（Phase 4 范围）

1. **ScoreTracker 多因子排序**: 将 rankAgents()（§8.5）集成到 Dispatcher 的 Agent 选择逻辑中
2. **Coordinator 模态**: 在 Reactor Think 中插入四步判定门控（§5），支持 WBS 分解和 Observe-Wait 循环
3. **中断/继续/取消**: 基于 Context 级联传播的生命周期控制（§10.5）
4. **超时分级**: 三级超时策略（单任务/总体软/总体硬）（§10.3）
5. **AgentRegistry 接口抽象**: 将 goreact.AgentRegistry 替换为接口引用（降低耦合）


---

## Prompt 模板化改造 + 设计文档覆盖率提升 — 2026-04-30 (Phase 3.5)

### Prompt 模板化（遵循 reactor/prompts.go 模式）

- [x] **创建 `orchestration/prompts/` 目录**，4 个 `.tmpl` 文件全部英文
  - `routing_prompt.tmpl` — LLM Router 系统提示词 (Design §6.3)
  - `capability_extraction_prompt.tmpl` — Agent Factory 能力提取提示词 (Design §12.2.1)
  - `body_generation_prompt.tmpl` — Agent Factory System Prompt 生成提示词 (Design §12.2.1)
  - `wbs_decomposition_prompt.tmpl` — WBS 分解判定提示词 (Design §11.2)
- [x] **创建 `orchestration/prompts.go`** — 完全遵循 `reactor/prompts.go` 模式
  - `embed.FS` 嵌入模板文件系统
  - `template.Must(template.New(...).ParseFS(...))` init 时解析
  - 定义 Data 结构体: `routingPromptData`, `capabilityExtractionPromptData`, `bodyGenerationPromptData`, `wbsDecompositionPromptData`
  - 渲染函数: `renderRoutingPrompt()`, `renderCapabilityExtractionPrompt()`, `renderBodyGenerationPrompt()`, `renderWBSDecompositionPrompt()`
  - 辅助函数: `toAgentViews()`, `formatScore()`
- [x] **重写 `router.go`** — 移除所有硬编码中文 prompt
  - `buildRoutingPrompt()` 改为调用 `renderRoutingPrompt(data)` 
  - 新增 `rankAgents()` 多因子排序方法 (Design §8.5)
  - 新增 `SelectBest()` epsilon-greedy 优选方法
  - 新增 `min()` / `minF64()` 辅助函数
- [x] **重写 `factory.go`** — 移除所有硬编码中文 prompt
  - `generateWithLLM()` 改为调用模板渲染函数
  - `generateRuleBased()` 英文化默认 Introduction（含 behavioral principles）
  - 新增 `truncateStr()` 通用截断工具函数

### WithDefaultModel 自动创建 LLM Router

- [x] **修复构造函数**: 当 `WithDefaultModel(cfg)` 提供了有效 APIKey 时，自动创建 LLMRouter 并注入
  - 自动创建 AgentFactory（以 Registry 为后端）
  - 始终创建 ScoreTracker
  - 日志记录 "LLM Router auto-created from default model"

### 设计文档覆盖率提升

- [x] **§7.1 完整事件类型** (`orchestration/events.go`)
  - 上行事件: `TaskDispatchEvent`, `QueryStatusEvent`, `AgentScoreEvent`
  - 下行事件: `TaskAssignedEvent`, `TaskResultEvent`, `TimeoutWarningEvent`
  - 生命周期控制: `CoordControlCommand`, `CoordLifecycleEvent`, `ResumeTaskEvent`, `TaskPausedEvent`
  - WBS 类型: `ResponsibilityCheckResult`, `AtomicityCheckResult`, `TaskDecomposition`
  - Coordinator 类型: `TaskProgressTable`, `TaskEntry`, `TaskState`(8状态), `LifecycleState`(4状态)
  - 工具方法: `IsFinalState()`, `PendingTaskIDs()`, `CompletedCount()`, `FailedCount()`
- [x] **§8.5 多因子排序** 集成到 router.go
  - Factor 1: 绩效分 (40%)
  - Factor 2: 关键词语义匹配 (30%)
  - Factor 3: 可用性/空闲度 (20%)
  - Factor 4: 近期活跃度 10%/30天衰减 (10%)
- [x] **§11 WBS 分解** 模板已创建（待集成到 Reactor Think 流程）
- [x] **§12.2.1 Description/Body 分离生成** 通过双模板实现

### 当前设计文档覆盖率

| 设计组件             | 设计文档                                | 当前状态                           |
| -------------------- | --------------------------------------- | ---------------------------------- |
| **LLM Router**       | §6.3: LLM 语义匹配 + 缓存 + fallback    | ✅ 完整（含模板+多因子排序）        |
| **Dispatcher**       | §6.4: Router→选择→状态更新→转发         | ✅ RouteTask + DelegateTo 双路径    |
| **AgentFactory**     | §12: 动态创建 + 双阶段 LLM 生成         | ✅ 完整（含重叠检测+降级+英文模板） |
| **ScoreTracker**     | §8: 0-3分 + EMA衰减 + epsilon-greedy    | ✅ 已接入 handleResult + SelectBest |
| **事件类型全集**     | §7.1: 8种上行 + 4种下行 + 控制事件      | ✅ 完整实现                         |
| **多因子排序**       | §8.5: 四因子加权排序                    | ✅ rankAgents() + SelectBest()      |
| **WBS 分解**         | §11: 模板就绪，待接入 Think             | ⚠️ 模板完成，待 Reactor 集成        |
| **Coordinator 模态** | §4.3/§10: TaskProgressTable + Wait 循环 | ⚠️ 数据结构就绪，待行为实现         |
| **超时分级**         | §10.3: 三级超时策略                     | ❌ 待实现                           |
| **生命周期控制**     | §10.5: Interrupt/Resume/Cancel          | ⚠️ 类型定义完成，待行为实现         |
| **Prompt 模板化**    | 遵循项目 embed.FS 模式                  | ✅ 全部英文 .tmpl 文件              |

### 测试结果: 34/34 PASS (0.947s)

---

- [ ] buildLLMBuilder 和 estimateInputTokens 包含了 完全相同的历史消息裁剪逻辑 （从末尾向前遍历、按 token 预算截断）
  - 位置 : llm.go:L115-L139 和 llm.go:L188-L201
  - 影响 : 修改裁剪逻辑时需要两处同步更新
  - 建议 : 抽取公共的 trimHistoryByTokenBudget() 函数
- [ ] List() 和 Models() 方法 功能完全重复 ，都是遍历 m.models 返回列表
  - 位置 : model_registry.go:L50-L56 和 model_registry.go:L91-L103
  - 建议 : 删除其中一个，或明确两个方法的不同语义
- [ ] Ask() 和 AskStream() 使用 context.TODO() 而非 context.Background() 或带超时的 context
  - 影响 : context.TODO() 是一个占位符，表明开发者不确定应该用什么 context。 在生产环境中，这会导致请求没有超时机制，可能无限期阻塞
  - 建议 : 使用 context.WithTimeout 设置合理的超时（如 5 分钟），或将 context 作为参数传入
  - 位置 : agent.go:L520 , agent.go:L591
- [ ] 问题 : 直接对 a.sessionStore 进行 *core.MemorySessionStore 类型断言——这违反了接口设计原则，且假定 sessionStore 一定是内存实现
  - 建议 : 在 SessionStore 接口中添加 RegisterRole 方法，避免类型断言
  - 位置 : agent.go:L468 , agent.go:L996
- [ ] 问题 : historyTokenBudgetRatio = 0.7 在两个地方独立定义
  - 建议 : 统一定义在 core/constants.go ，只在 reactor 中定义一次
  - 位置 : reactor/reactor.go:L40-L41 和 agent.go:L681
- [ ] 131072 硬编码作为会话 token 预算，注释说 "如果 MaxTokens 小于 40K 对于一般任务都难以处理"
  - 建议 : 定义为常量 DefaultSessionTokens
  - 位置 : agent.go:L281 , agent.go:L399
- [ ] 问题 : Coordinator 轮询间隔 500ms 、最大 5s 、超时 10min 全部硬编码
  - 建议 : 定义为常量或配置项
  - 位置 : reactor/reactor.go:L752-L753
- [ ] 问题 : NewReactor() 函数 超过 180 行 ，承担了 Reactor 的全部初始化职责：配置校验、注册表创建、技能加载、工具注册、事件绑定等
  - 建议 : 拆分为 applyDefaults() , initRegistries() , loadSkills() , registerTools() , setupToolExecutor() 等独立方法
  - 位置 : reactor.go:L225-L407
- [] 问题 : runTAOLoop() 函数 约 160 行 ，混合了循环控制、T-A-O 调度、消息持久化、步骤记录、结果构建等多种职责
  - 建议 : 将步骤记录和结果构建拆分为独立方法
  - 位置 : reactor.go:L573-L731
- [ ] 问题 : Reactor 结构体是一个 "上帝对象" ，持有 15+ 个字段，囊括了:
  - 配置管理 ( config )
  - 意图注册表 ( intentRegistry )
  - 工具注册表 + 执行器 ( toolRegistry , toolExecutor )
  - 技能注册表 ( skillRegistry )
  - 规则注册表 ( ruleRegistry )
  - LLM 调用 ( llmClient )
  - 内存检索 ( memory )
  - 事件总线 ( eventBus )
  - 会话管理 ( sessionStore , contextWindow )
  - 编排器 ( orchestrator )
  - 快照/暂停状态管理 ( snapshotHolder , pauseRequested )
  - 影响 : 单个类的变更原因过多，测试困难，难以复用
  - 建议 : 将 Reactor 拆分为：
  - Executor — 负责 T-A-O 循环执行
  - RegistryManager — 管理 tool/skill/rule 注册
  - SessionManager — 管理上下文窗口和会话状态
  - 位置 : reactor/reactor.go:L120-L152
  - 影响 : 单个类的变更原因过多，测试困难，难以复用
  - 建议 : 将 Reactor 拆分为：
  - Executor — 负责 T-A-O 循环执行
  - RegistryManager — 管理 tool/skill/rule 注册
  - SessionManager — 管理上下文窗口和会话状态