用户提出了一个非常好的架构问题。让我仔细分析一下这段代码和用户的疑问。

用户关注的是 `goreact/reactor/reactor.go` 文件中 `Think` 方法的实现，具体是第 802-840 行。

从代码来看：
1. 第 806 行：`tools := r.toolRegistry.ToToolInfos()` - 获取所有注册的工具
2. 第 809 行：`skills, _ := r.skillRegistry.FindApplicableSkills(ctx.Intent)` - 根据意图匹配适用的技能
3. 第 825 行：将 `tools`, `skills`, `memoryRecords` 都传入 prompt 构建

用户的疑问是：
1. 当前代码是否把所有注册的工具都加载到对话上下文中？
2. 如果有上百个工具，这样做是否合理？
3. 是否应该先选 Skill，然后从 Skill 中过滤 Tools？
4. 或者如果没有 Skill 就用 Intent 去过滤 Tools？

这是一个关于代码架构和设计意图的问题。我需要深入理解这个代码库的设计，才能给出准确的答案。这需要：
1. 查看 `ToToolInfos()` 的实现 - 看看它是否真的返回所有工具
2. 查看 `FindApplicableSkills` 的实现 - 理解技能匹配逻辑
3. 查看 `BuildThinkPrompt` 的实现 - 理解 tools 和 skills 是如何被使用的


4. 探索 skill_registry 和 tool_registry 的结构，以及它们之间的关联方式

我应该先检查相关文件来理解整体架构设计。
这是一个很好的架构问题。让我先深入研究代码库，理解当前的设计意图和相关实现细节。
非常好，我现在已经获得了非常详细的代码分析结果。让我总结一下关键发现：

## 核心发现

### 1. `ToToolInfos()` 确实返回全部工具
从 `action.go:180-188` 可以看到：
```go
func (r *ToolRegistry) ToToolInfos() []core.ToolInfo {
	tools := r.All()  // 获取全部工具
	infos := make([]core.ToolInfo, len(tools))
	for i, t := range tools {
		infos[i] = *t.Info()
	}
	return infos
}
```

**确认：无任何过滤逻辑，返回所有已注册工具。**

### 2. Skill 有意图驱动的过滤
从 `skill_registry.go:58-88` 可以看到 `FindApplicableSkills` 基于意图的关键词匹配来过滤 Skill。

### 3. BuildThinkPrompt 的实现
从 `thought.go:43-69`：
```go
func BuildThinkPrompt(input string, intent *Intent, tools []core.ToolInfo, skills []*core.Skill, memoryRecords []core.MemoryRecord) string {
	// ...
	toolSection := FormatToolDescriptions(tools)  // 直接使用传入的全部 tools
	// ...
}
```

### 4. think_prompt.tmpl 模板
模板显示 tools 和 skills 是分开渲染的：
- `{{skillSection .Skills}}` - 渲染激活的技能
- `{{.ToolSection}}` - 渲染**所有**工具

### 5. FormatToolDescriptions 的实现


从 `context.go:140-151` 可以看到，这个函数将每个工具的名字和描述格式化为编号列表，没有任何过滤机制。这意味着所有注册的工具都会被包含在最终的提示中。

### 6. Skill 结构体中有 AllowedTools 字段
Skill 结构体定义中包含了 `AllowedTools string` 字段，这是一个空格分隔的工具名列表。但根据文档说明，这个字段目前是实验性的，框架层面还没有实现基于此字段进行工具权限过滤的逻辑。

### 7. 默认注册的工具数量
默认情况下会注册约 25 个工具（第 524-548 行），如果加上 MCP 工具和额外工具，数量会更多。

### 8. 设计文档的确认
开发指南明确指出 `allowed-tools` 字段是实验性的功能。虽然会在 SKILL.md 中声明并存储在 `Skill.AllowedTools` 中，但框架尚未实现基于此字段的工具过滤逻辑，目前仅用于文档化 Skill 推荐使用的工具集合。

## 用户问题的答案

用户的观察是完全准确的：

1. **当前行为**：`ToToolInfos()` 确实返回**全部**注册的工具，不做任何过滤
2. **Think 阶段的流程**：获取全部工具 → 匹配适用的 Skills → 将**全部工具 + 匹配的 Skills** 都注入 Prompt
3. **问题所在**：如果有上百个工具，确实会全部加载到上下文中

## 设计意图分析

为什么这样设计？可能有几个原因：

1. **简单可靠**：让 LLM 自己选择合适的工具，避免过滤逻辑遗漏导致工具不可用
2. **Skill 是指令增强而非工具过滤**：Skill 的设计目的是注入领域知识指令，而不是限制可用工具
3. **AllowedTools 是实验性功能**：设计者已经意识到这个需求，但还未实现


4. **现有防护机制**：系统通过三层防御策略来管理上下文膨胀——工具结果持久化存储、消息级字符限制和上下文压缩——而不是在工具加载阶段进行过滤。现在需要整理这些发现，给用户一个清晰、完整的回答。
你的直觉非常敏锐。经过对代码库的完整分析，我来给你一个明确的答案。

## 结论：你的担忧完全成立

**当前代码确实是把全部注册工具一次性加载到 LLM 上下文中**，没有基于 Intent 或 Skill 进行过滤。

让我用代码证据说话：

### 证据链

**1. `ToToolInfos()` 无过滤**

`goreact/reactor/action.go:180-188`:
```go
func (r *ToolRegistry) ToToolInfos() []core.ToolInfo {
	tools := r.All()           // ← 获取全部工具
	infos := make([]core.ToolInfo, len(tools))
	for i, t := range tools {
		infos[i] = *t.Info()
	}
	return infos               // ← 全部返回，无条件过滤
}
```

**2. Think 方法直接使用全部工具**

`goreact/reactor/reactor.go:806-825`:
```go
func (r *Reactor) Think(ctx *ReactContext) (int, error) {
	tools := r.toolRegistry.ToToolInfos()              // ← 全部工具
	
	skills, _ := r.skillRegistry.FindApplicableSkills(ctx.Intent)  // ← Skill 有过滤
	
	instructions := BuildThinkPrompt(ctx.Input, ctx.Intent, tools, skills, memoryRecords)
	//                                                                  ↑ 全部工具塞进去
}
```

**3. Prompt 模板原样渲染全部工具**

`goreact/reactor/prompts/think_prompt.tmpl:34-36`:
```xml
<available_tools>
{{.ToolSection}}     <!-- 这里是全部工具的描述列表 -->
</available_tools>
```

**4. `FormatToolDescriptions` 只是格式化**

`goreact/reactor/context.go:140-151`:
```go
func FormatToolDescriptions(tools []core.ToolInfo) string {
	// 遍历传入的 tools，生成 "1. **tool-name**: description" 列表
	// 无任何过滤逻辑
}
```

---

## 当前的设计意图是什么？

根据代码分析，当前的设计哲学是：

### Tool 和 Skill 的职责分离

| 组件 | 是否过滤 | 注入内容 | 定位 |
|------|---------|---------|------|
| **Tools** | 否（全量加载） | 工具名称+描述 | 原子操作能力 |
| **Skills** | 是（意图匹配） | 领域知识+操作规程 | 行为编排指导 |

**Skill 的设计目的不是过滤工具，而是增强指令。**

看一个实际例子 `reactor/skills/bug-hunter/SKILL.md`:
```yaml
---
name: bug-hunter
description: Expert SOP for locating, isolating and fixing complex bugs.
allowed-tools: grep glob bash subagent read   # ← 这个字段存在！
---

# Debug: Session & Bug Analysis
## Instructions
1. **Gather Context**: Use 'grep' and 'read' to locate...
2. **Analyze**: ...consider launching an independent SubAgent...
```

注意：Skill 的 Instructions 中会**用文字告诉 LLM 应该用什么工具**（如 `Use 'grep' and 'read'`），但 `<available_tools>` 列表中仍然包含所有工具。

### `allowed-tools` 字段是实验性的

`goreact/docs/skill-dev-guide.md:55` 明确写道：

> 关于 `allowed-tools` 字段，这是 Agent Skills 规范中的实验性功能。在 SKILL.md 中声明后，目前 GoReact 会将其存储在 `Skill.AllowedTools` 中，**但框架层面尚未实现基于此字段进行工具权限过滤的逻辑**。

**这说明设计者已经意识到了这个需求，但还没实现。**

---

## 当前规模的实际情况

默认配置下，Reactor 注册约 **25+ 个工具**：

`reactor.go:525-548` 列出的内置工具：
- 文件操作：read, write, file_edit, replace, ls, glob, grep
- 执行环境：bash, repl, calculator, echo
- 网络能力：web_search, web_fetch
- 任务管理：todo_write, todo_read, todo_execute, cron
- 记忆系统：memory_save, memory_search
- 编排工具：task, subagent, team, skill_create, skill_list
- 交互工具：ask_user, ask_permission

再加上 MCP 工具和通过 `WithExtraTools` 注册的自定义工具，**达到上百个是完全可能的**。

每个工具的 Description 大约 50-200 tokens，100 个工具 ≈ **5000-20000 tokens** 仅用于工具列表。

---

## 为什么当前能工作？（三层防御机制）

代码库设计了三层上下文防御机制，但它们都不在 Think 阶段：

1. **第二层：Tool Result Persistence** (`action.go:379-417`)
   - 大工具结果保存到磁盘，只保留预览
   
2. **Per-message 字符限制** (`action.go:394-406`)
   - 单轮工具输出超过阈值时强制截断

3. **第三层：Context Compaction** (`reactor.go:1384-1456`)
   - 对话历史超限时自动压缩（micro-compact 或 LLM compact）

**关键点：这些防御都是在 Act→Observe 之后生效的，Think 阶段的工具列表膨胀问题并未被处理。**

---

## 你建议的改进方案分析

你提到的两个方案都是合理的：

### 方案 A：从 Skill 的 allowed-tools 过滤
```go
// 伪代码
func (r *Reactor) Think(ctx *ReactContext) (int, error) {
    allTools := r.toolRegistry.ToToolInfos()
    skills, _ := r.skillRegistry.FindApplicableSkills(ctx.Intent)
    
    // 如果有激活的 Skill，用它们的 allowed-tools 过滤
    if len(skills) > 0 {
        allowed := extractAllowedTools(skills)  // 从 Skill.AllowedTools 提取
        tools := filterTools(allTools, allowed)
    } else {
        tools = allTools  // 无 Skill 时 fallback 到全部
    }
    
    instructions := BuildThinkPrompt(ctx.Input, ctx.Intent, tools, skills, memoryRecords)
}
```

**优点**：
- 符合 Agent Skills 规范的设计意图
- Skill 作者可以精确控制可用工具集
- Token 消耗可预测

**缺点**：
- 需要实现 `allowed-tools` 解析和过滤逻辑
- 可能导致 LLM 在需要时找不到工具（如果 Skill 定义不完善）
- 多个 Skill 的 allowed-tools 需要合并

### 方案 B：基于 Intent 过滤 Tools（无 Skill 时）
```go
// 伪代码
if len(skills) == 0 {
    tools = filterToolsByIntent(allTools, ctx.Intent)  // 用 Intent 语义匹配工具
} else {
    tools = filterToolsBySkills(allTools, skills)
}
```

**优点**：
- 即使没有 Skill，也能减少工具数量
- 类似 Skill Registry 的关键词匹配机制

**缺点**：
- 工具没有像 Skill 那样的丰富 description 用于匹配
- 可能需要给工具添加 category/tags 元数据
- 实现复杂度较高

---

## 我的建议

如果要改进当前的实现，我推荐**渐进式路线**：

### Phase 1：实现 Skill.allowed-tools 过释（立即可做）

1. 在 `Skill` 结构体的 `AllowedTools` 字段已经存在
2. 在 Think 方法中加入过滤逻辑
3. 保持向后兼容：如果没有 Skill 或 allowed-tools 为空，behavior 不变

改动量很小，核心修改点在 `reactor.go:806-825`。

### Phase 2：添加 Tool Category/Tags（中期优化）

为 `ToolInfo` 添加可选的 `Category` 或 `Tags` 字段：
```go
type ToolInfo struct {
    Name        string
    Description string
    Category    string   // 新增: "file-system", "execution", "network", ...
    Tags        []string // 新增: ["io", "read-only", "dangerous"]
}
```

当没有 Skill 匹配时，可以用 Intent 语义匹配 Tool 的 Category/Tags。

### Phase 3：动态 Tool 选择（长期架构）

借鉴 OpenAI Assistants API 的 `tool_choice` 和 Anthropic's tool use 机制：
- 第一轮 Think：只用工具摘要（name + 一行描述），让 LLM 选出候选工具
- 第二轮 Think：注入选中工具的完整参数定义
- 这需要更复杂的多阶段推理流程

---

## 总结回答你的问题

> 当前的这种写法到底意欲何为？

**这是一个"先求正确，再求优化"的设计决策。**

当前设计的优先级是：
1. 保证所有工具都可用（不因过滤逻辑 bug 导致工具丢失）
2. 通过 Skill Instructions 用自然语言引导 LLM 选择合适工具
3. 依靠三层上下文膨胀防御机制兜底

**这在工具数量 <30 时完全合理**，但当你提到"上百个工具"时，这确实会成为瓶颈。

而 `allowed-tools` 字段的存在证明设计者预留了扩展点，只是还没到实现的优先级。

如果你希望，我可以帮你实现 Phase 1 的 Skill.allowed-tools 过滤逻辑，改动量很小且向后兼容。