# 📚 Claude Skill 与子代理机制深度解析

## 一、Skill 系统的整体架构

### 1. **Skill 的数据结构定义**

在 [bundledSkills.ts:15-41](file:///Users/ray/workspaces/ai-ecosystem/cludecode/skills/bundledSkills.ts#L15-L41) 中定义了 `BundledSkillDefinition` 类型：

```typescript
export type BundledSkillDefinition = {
  name: string                    // 技能名称
  description: string             // 技能描述
  aliases?: string[]              // 别名
  whenToUse?: string              // 🔑 关键字段：何时使用此技能的详细描述
  argumentHint?: string           // 参数提示
  allowedTools?: string[]         // 允许的工具列表
  model?: string                  // 模型覆盖
  disableModelInvocation?: boolean // 是否禁用模型调用
  userInvocable?: boolean         // 用户是否可调用
  isEnabled?: () => boolean       // 启用条件函数
  hooks?: HooksSettings           // 钩子设置
  context?: 'inline' | 'fork'     // 🔑 关键字段：执行上下文（内联或forked子代理）
  agent?: string                  // 🔑 关键字段：指定子代理类型
  files?: Record<string, string>  // 参考文件
  getPromptForCommand: (args, context) => Promise<ContentBlockParam[]>  // 获取提示词
}
```

### 2. **Command 类型的扩展**

在 [types/command.ts:25-57](file:///Users/ray/workspaces/ai-ecosystem/cludecode/types/command.ts#L25-L57) 中的 `PromptCommand` 类型进一步明确了这些字段的作用：

```typescript
export type PromptCommand = {
  type: 'prompt'
  // ... 其他字段
  context?: 'inline' | 'fork'   // 'inline' = 在当前对话中展开
                                 // 'fork' = 在子代理中独立运行
  agent?: string                // forked时使用的代理类型（如 'Explore', 'general-purpose'）
}
```

---

## 二、🎯 Skill 如何识别和匹配子代理类型

### **核心机制：两个字段决定一切**

Claude 通过 Skill 定义中的**两个关键字段**来识别和启用子代理：

| 字段      | 位置                                                                                                   | 作用                                                  |
| --------- | ------------------------------------------------------------------------------------------------------ | ----------------------------------------------------- |
| `context` | [bundledSkills.ts:28](file:///Users/ray/workspaces/ai-ecosystem/cludecode/skills/bundledSkills.ts#L28) | 决定执行模式：`'inline'`（默认）或 `'fork'`（子代理） |
| `agent`   | [bundledSkills.ts:29](file:///Users/ray/workspaces/ai-ecosystem/cludecode/skills/bundledSkills.ts#L29) | 指定 forked 时使用哪个具体的子代理类型                |

### **匹配流程详解**

**第一步：Skill 描述中的触发信号**

在 [SkillTool/prompt.ts:174-196](file:///Users/ray/workspaces/ai-ecosystem/cludecode/tools/SkillTool/prompt.ts#L174-L196) 中，系统生成如下提示给 Claude：

```
When users ask you to perform tasks, check if any of the available skills match. 
Skills provide specialized capabilities and domain knowledge.

When a skill matches the user's request, this is a BLOCKING REQUIREMENT: 
invoke the relevant Skill tool BEFORE generating any other response about the task
```

**第二步：whenToUse 字段的智能匹配**

每个 Skill 都有 `whenToUse` 字段（见 [batch.ts:105-106](file:///Users/ray/workspaces/ai-ecosystem/cludecode/skills/bundled/batch.ts#L105-L106)），例如：

```typescript
whenToUse: 'Use when the user wants to make a sweeping, mechanical change across 
many files (migrations, refactors, bulk renames) that can be decomposed into 
independent parallel units.'
```

Claude 模型通过语义匹配用户请求与 `whenToUse` 描述来决定调用哪个 Skill。

**第三步：上下文判断 - 是否需要子代理**

在 [SkillTool/SkillTool.ts:622](file:///Users/ray/workspaces/ai-ecosystem/cludecode/tools/SkillTool/SkillTool.ts#L622) 中的关键判断逻辑：

```typescript
// 检查 skill 应该以 forked 子代理方式运行
if (command?.type === 'prompt' && command.context === 'fork') {
  return executeForkedSkill(
    command,
    commandName,
    args,
    context,
    canUseTool,
    parentMessage,
    onProgress,
  )
}
```

---

## 三、🚀 子代理的启用和完整调用流程

### **完整的 Forked Skill 调用链路**

```
用户请求 → Skill 匹配 → context检查 → agent解析 → 子代理启动 → 结果返回
```

#### **阶段 1：Skill 调用入口**

[SkillTool/SkillTool.ts:580-632](file:///Users/ray/workspaces/ai-ecosystem/cludecode/tools/SkillTool/SkillTool.ts#L580-L632) 的 `call()` 方法：

1. **验证输入**（validateInput 已确认技能存在且有效）
2. **查找命令对象**：`const command = findCommand(commandName, commands)`
3. **关键判断**：
   ```typescript
   if (command?.type === 'prompt' && command.context === 'fork') {
     // 🔄 进入 Forked 执行路径
   }
   ```

#### **阶段 2：准备 Forked 上下文**

[utils/forkedAgent.ts:191-232](file:///Users/ray/workspaces/ai-ecosystem/cludecode/utils/forkedAgent.ts#L191-L232) 的 `prepareForkedCommandContext()` 函数：

```typescript
export async function prepareForkedCommandContext(
  command: PromptCommand,
  args: string,
  context: ToolUseContext,
): Promise<PreparedForkedContext> {
  // 1. 获取 Skill 内容（替换 $ARGUMENTS 占位符）
  const skillPrompt = await command.getPromptForCommand(args, context)
  
  // 2. 解析允许的工具列表
  const allowedTools = parseToolListFromCLI(command.allowedTools ?? [])
  
  // 3. 🔑 关键：解析 agent 类型
  const agentTypeName = command.agent ?? 'general-purpose'  // 默认使用 general-purpose
  
  // 4. 从已注册的代理列表中查找匹配的代理定义
  const agents = context.options.agentDefinitions.activeAgents
  const baseAgent = agents.find(a => a.agentType === agentTypeName)
    ?? agents.find(a => a.agentType === 'general-purpose')  // 回退到通用代理
    ?? agents[0]  // 最终回退
  
  return { skillContent, modifiedGetAppState, baseAgent, promptMessages }
}
```

#### **阶段 3：执行 Forked Skill**

[SkillTool/SkillTool.ts:122-289](file:///Users/ray/workspaces/ai-ecosystem/cludecode/tools/SkillTool/SkillTool.ts#L122-L289) 的 `executeForkedSkill()` 函数：

```typescript
async function executeForkedSkill(
  command: Command & { type: 'prompt' },
  commandName: string,
  args: string | undefined,
  context: ToolUseContext,
  canUseTool: CanUseToolFn,
  parentMessage: AssistantMessage,
  onProgress?: ToolCallProgress<Progress>,
): Promise<ToolResult<Output>> {
  const startTime = Date.now()
  const agentId = createAgentId()
  
  // 1. 准备 Forked 上下文（包含 agent 解析）
  const { modifiedGetAppState, baseAgent, promptMessages, skillContent } =
    await prepareForkedCommandContext(command, args || '', context)
  
  // 2. 合并 skill 的 effort 设置到 agent 定义
  const agentDefinition = command.effort !== undefined
    ? { ...baseAgent, effort: command.effort }
    : baseAgent
  
  // 3. 🚀 运行子代理
  for await (const message of runAgent({
    agentDefinition,
    promptMessages,
    toolUseContext: {
      ...context,
      getAppState: modifiedGetAppState,
    },
    canUseTool,
    isAsync: false,
    querySource: 'agent:custom',
    model: command.model as ModelAlias | undefined,
    availableTools: context.options.tools,
    override: { agentId },
  })) {
    agentMessages.push(message)
    // ... 处理进度报告
  }
  
  // 4. 提取结果文本
  const resultText = extractResultText(agentMessages, 'Skill execution completed')
  
  return {
    data: {
      success: true,
      commandName,
      status: 'forked',  // 标记为 forked 执行
      agentId,
      result: resultText,
    },
  }
}
```

#### **阶段 4：子代理实际执行**

[tools/AgentTool/runAgent.ts:248-860](file:///Users/ray/workspaces/ai-ecosystem/cludecode/tools/AgentTool/runAgent.ts#L248-L860) 的 `runAgent()` 函数负责实际的子代理运行：

- 创建独立的 `agentToolUseContext`（隔离状态）
- 初始化 MCP 服务器（如果 agent 定义了）
- 预加载 Skills（如果 agent 定义了 `skills` 字段）
- 调用 `query()` 函数启动 AI 对话循环
- 收集消息并记录 transcript

---

## 四、📖 内置子代理类型示例

从 [builtInAgents.ts:22-72](file:///Users/ray/workspaces/ai-ecosystem/cludecode/tools/AgentTool/builtInAgents.ts#L22-L72) 可以看到 Claude 内置的代理类型：

```typescript
export function getBuiltInAgents(): AgentDefinition[] {
  const agents: AgentDefinition[] = [
    GENERAL_PURPOSE_AGENT,      // 通用目的代理
    STATUSLINE_SETUP_AGENT,     // 状态栏设置代理
  ]
  
  if (areExplorePlanAgentsEnabled()) {
    agents.push(EXPLORE_AGENT, PLAN_AGENT)  // 探索和计划代理
  }
  
  agents.push(CLAUDE_CODE_GUIDE_AGENT)  // 代码指南代理
  
  return agents
}
```

**Explore Agent 示例**（[exploreAgent.ts:64-83](file:///Users/ray/workspaces/ai-ecosystem/cludecode/tools/AgentTool/built-in/exploreAgent.ts#L64-L83)）：

```typescript
export const EXPLORE_AGENT: BuiltInAgentDefinition = {
  agentType: 'Explore',
  whenToUse: 'Fast agent specialized for exploring codebases...',
  disallowedTools: [
    AGENT_TOOL_NAME,
    FILE_EDIT_TOOL_NAME,
    FILE_WRITE_TOOL_NAME,  // 只读代理，禁止写操作
  ],
  source: 'built-in',
  model: process.env.USER_TYPE === 'ant' ? 'inherit' : 'haiku',
  omitClaudeMd: true,  // 不加载 CLAUDE.md 以节省 token
  getSystemPrompt: () => getExploreSystemPrompt(),
}
```

---

## 五、🔗 完整的数据流向图

```
┌─────────────────────────────────────────────────────────────────────┐
│                        用户请求                                      │
│  "帮我重构这个模块"                                                   │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│  SkillTool.prompt() - 系统提示                                       │
│  "When users ask you to perform tasks, check if any of the            │
│   available skills match..."                                         │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Claude 语义匹配 whenToUse 字段                                      │
│                                                                     │
│  Skill Registry:                                                    │
│  ┌────────────┬─────────────┬──────────┬────────┬──────────┐        │
│  │ Name       │ whenToUse   │ context  │ agent  │ tools    │        │
│  ├────────────┼─────────────┼──────────┼────────┼──────────┤        │
│  │ batch      │ 大规模变更   │ inline   │ -      │ Agent... │        │
│  │ verify     │ 验证代码     │ inline   │ -      │ Bash...  │        │
│  │ custom-skill│ 特定任务   │ fork     │ Explore│ Read,... │        │
│  └────────────┴─────────────┴──────────┴────────┴──────────┘        │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│  SkillTool.call()                                                   │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ 1. findCommand(commandName)  // 查找技能定义                 │    │
│  │ 2. check command.context === 'fork'?                        │    │
│  │    ├─ Yes → executeForkedSkill()                            │    │
│  │    └─ No  → processPromptSlashCommand() [Inline execution]  │    │
│  └─────────────────────────────────────────────────────────────┘    │
└──────────────────────────┬──────────────────────────────────────────┘
                           │ (if fork)
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│  executeForkedSkill()                                               │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ prepareForkedCommandContext():                               │    │
│  │   1. getPromptForCommand()  // 获取技能内容                   │    │
│  │   2. agentTypeName = command.agent ?? 'general-purpose'      │    │
│  │   3. baseAgent = agents.find(a => a.agentType === agentType) │    │
│  │                                                              │    │
│  │ runAgent({                                                    │    │
│  │   agentDefinition: baseAgent,  // 使用解析出的代理             │    │
│  │   promptMessages: [skillContent],                             │    │
│  │   querySource: 'agent:custom',                                │    │
│  │ })                                                            │    │
│  └─────────────────────────────────────────────────────────────┘    │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│  runAgent() - 子代理执行引擎                                         │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ • createSubagentContext()  // 创建隔离的执行上下文            │    │
│  │ • initializeAgentMcpServers()  // 初始化 MCP 服务            │    │
│  │ • preload skills  // 预加载代理需要的技能                     │    │
│  │ • query()  // 启动 AI 对话循环                               │    │
│  │ • yield messages  // 返回结果消息流                          │    │
│  └─────────────────────────────────────────────────────────────┘    │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│  结果返回给父级代理                                                  │
│  { success: true, status: 'forked', agentId, result: text }         │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 六、💡 关键设计要点总结

### **1. 声明式配置驱动**

Skill 通过简单的声明式字段就能控制复杂的子代理行为：

- `context: 'fork'` → 自动启用子代理模式
- `agent: 'Explore'` → 指定使用探索型子代理
- 无需编写代码，只需配置

#### **2. 智能回退机制**

在 [forkedAgent.ts:212-217](file:///Users/ray/workspaces/ai-ecosystem/cludecode/utils/forkedAgent.ts#L212-L217) 中实现了三级回退：

```typescript
const baseAgent =
  agents.find(a => a.agentType === agentTypeName) ??  // 1. 精确匹配
  agents.find(a => a.agentType === 'general-purpose') ??  // 2. 通用代理
  agents[0]  // 3. 第一个可用代理
```

### **3. 完全隔离的执行环境**

Forked 子代理拥有：
- 独立的文件状态缓存
- 独立的权限控制上下文
- 可选的独立 MCP 服务器
- 独立的 abort controller
- 独立的 transcript 记录

### **4. 内联 vs Forked 的选择标准**

根据 [skillify.ts:144](file:///Users/ray/workspaces/ai-ecosystem/cludecode/skills/bundled/skillify.ts#L144) 的指导：

- **`context: inline`**（默认）：适用于需要用户中途交互的技能
- **`context: fork`**：适用于自包含、不需要中途用户输入的任务

          
## 七、🎯 核心答案总结

**Claude 如何在 Skill 描述中识别并启用子代理？**

**答案：通过两个声明式字段的组合：**

1. **`context: 'fork'`** - 告诉系统"这个技能需要以子代理方式运行"
2. **`agent: 'Explore'`** - 告诉系统"使用 Explore 类型的子代理来执行"

**工作原理：**
- Claude 模型在收到用户请求后，会语义匹配所有可用 Skill 的 `whenToUse` 描述
- 当匹配到某个 Skill 后，检查其 `context` 字段
- 如果是 `'fork'`，则读取 `agent` 字段（默认 `'general-purpose'`）
- 在已注册的代理列表中查找匹配的代理定义
- 调用 `runAgent()` 启动独立的子代理进程执行任务
- 子代理完成后返回结果给父级代理

这种设计的精妙之处在于：
- ✅ **声明式配置** - 无需编写代码，只需配置字段
- ✅ **智能回退** - 自动回退到通用代理保证健壮性
- ✅ **完全隔离** - 子代理拥有独立的执行环境
- ✅ **灵活扩展** - 可以轻松添加新的代理类型和技能

这就是 Claude Skill 系统的完整机制！🎉