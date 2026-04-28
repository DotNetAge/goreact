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
  - 当 `ContextWindow` Tokens 的大小达到其最大容纳边界时，会触发【滚动】，有了`SessionStore`的加持后被就不会丢失对话内容，当内容被【滑出】上下文窗口后，可以触发`SessionStore`上的“滑动”方法，被滑出的一条或多条消息在掉落出上下文窗口时，客户端可以将这些被滑出的内容存入RAG或其它存储进行语义化成为“知识”，在用户当前上下文中按语义反向注入，这就避免了“上下文腐烂”的问题，同时也可以支持了无限的上下文，以及长期记忆与短期记忆的完美融合。
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
- [ ] 补充关于“Tools VS Skills”的文章，论证为何Skills会更优于Tools（由用户手动撰写）
  1. 对于绝大多数大模型，Tools并无法实现延时加载，而一次性加载更多的Tools会使模型的注意力下降，Token消耗增加。
  2. Skill 采用三级“渐进式加载”机制，当Skill被激活时才会从`AllowedTools`中加载工具，相比之下会降低思考链的的复杂度，同时也可以降低Token消耗；
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
- [ ] 功能设计问题：当Agent正在执行的时候，是否还能向Agent发送消息？
  - Agent空闲，而SubAgent正在执行，是否还能向Agent发送消息？
  - 当Agent在等待LLM完成执行时，是否还能向Agent发送消息？实现【紧急叫停】功能；
  - 如果SubAgent正在执行，不能够切换身份，只能等待执行完成；