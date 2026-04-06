# 正向与反向提示词

正向提示词用于激发模型能力，反向提示词用于抑制模型的不期望行为。两者结合形成双向引导机制，确保模型输出符合预期。

## 1. 双向引导机制

```mermaid
graph LR
    subgraph Positive[正向提示词 - 激发能力]
        Role[角色定义]
        Tools[工具列表]
        Format[输出格式]
        Examples[示例引导]
    end
    
    subgraph Negative[反向提示词 - 抑制能力]
        Forbidden[禁止行为]
        Constraints[约束条件]
        Boundaries[能力边界]
    end
    
    subgraph RAG[RAG 注入 - 知识增强]
        Documents[文档检索]
        GraphContext[图谱上下文]
    end
    
    Positive --> Model[LLM]
    Negative --> Model
    RAG --> Model
    Model --> Output[受控输出]
```

**双向引导的作用**：

| 方向 | 目的         | 手段               |
| ---- | ------------ | ------------------ |
| 正向 | 激发模型能力 | 角色定义、工具列表 |
| 反向 | 抑制不当行为 | 禁止规则、边界约束 |
| RAG  | 增强知识能力 | 文档检索、图谱注入 |

## 2. 反向提示词设计

反向提示词用于抑制模型的不期望行为：

```mermaid
classDiagram
    class NegativePromptGroup {
        +ID string
        +Name string
        +Description string
        +Prompts []NegativePrompt
        +Enabled bool
    }
    
    class NegativePrompt {
        +Pattern string
        +Reason string
        +Alternative string
    }
    
    NegativePromptGroup "1" *-- "*" NegativePrompt
```

### 2.1 反向提示词类型

| 类型     | 说明         | 示例                             |
| -------- | ------------ | -------------------------------- |
| 行为禁止 | 禁止特定行为 | "不要编造不存在的工具"           |
| 格式约束 | 约束输出格式 | "不要在 Action 之外输出额外解释" |
| 能力边界 | 限制能力范围 | "不要尝试执行需要权限的操作"     |
| 安全约束 | 安全相关限制 | "不要泄露敏感信息"               |

### 2.2 反向提示词组

```mermaid
graph TB
    subgraph NegativePromptGroups[反向提示词组]
        SafetyGroup[安全约束组]
        FormatGroup[格式约束组]
        BehaviorGroup[行为约束组]
        PermissionGroup[权限约束组]
    end
    
    subgraph SafetyGroup[安全约束组]
        S1[禁止泄露敏感信息]
        S2[禁止执行危险操作]
    end
    
    subgraph FormatGroup[格式约束组]
        F1[禁止格式错误]
        F2[禁止额外解释]
    end
    
    subgraph BehaviorGroup[行为约束组]
        B1[禁止编造工具]
        B2[禁止跳过推理]
    end
    
    subgraph PermissionGroup[权限约束组]
        P1[禁止越权操作]
        P2[禁止访问未授权资源]
    end
```

### 2.3 反向提示词配置示例

```go
type NegativePromptGroup struct {
    ID          string
    Name        string
    Description string
    Prompts     []NegativePrompt
    Enabled     bool
}

type NegativePrompt struct {
    Pattern    string
    Reason     string
    Alternative string
}

var DefaultNegativePromptGroups = []NegativePromptGroup{
    {
        ID:          "safety",
        Name:        "安全约束组",
        Description: "保障系统安全的基础约束",
        Enabled:     true,
        Prompts: []NegativePrompt{
            {Pattern: "不要泄露敏感信息", Reason: "防止数据泄露", Alternative: "使用脱敏数据"},
            {Pattern: "不要执行危险操作", Reason: "防止系统损坏", Alternative: "请求人工确认"},
        },
    },
    {
        ID:          "format",
        Name:        "格式约束组",
        Description: "确保输出格式正确",
        Enabled:     true,
        Prompts: []NegativePrompt{
            {Pattern: "不要在 Action 之外输出额外解释", Reason: "保证解析正确", Alternative: "将解释放入 Thought"},
        },
    },
}
```

## 3. 提示词冲突消解机制

在动态组装的过程中，由于 Agent 配置（System Prompt）、全局规则（Negative Prompts）和技能指南（Skill Prompt）来源不同，极其容易发生冲突（例如：Skill 允许读取所有文件，而 Negative Prompt 禁止访问配置目录）。

PromptBuilder 内置了一套**优先级权重与冲突发现机制**：

```mermaid
flowchart TB
    SysRole[1. 系统角色 System Prompt] --> Assembly[提示词组装管道]
    NegRules[2. 全局抑制 Negative Prompts] --> Assembly
    SkillInstructions[3. 技能指令 Skill Prompt] --> Assembly
    UserReq[4. 临时请求 User Request] --> Assembly
    
    Assembly --> ConflictDetect{语义冲突检测}
    
    ConflictDetect --> |发现冲突| Resolution[应用优先级策略]
    ConflictDetect --> |无冲突| Compile[编译最终 Prompt]
    
    Resolution --> Log[记录可观测性告警 Observability Warning]
    Resolution --> Override[按优先级覆写]
    Override --> Compile
```

### 3.1 优先级策略

**绝对边界**：`Negative Prompts` （定义安全红线） > `System Role` > `Skill Prompt` > `User Request`。

当系统检测到 Skill 请求打破了 Negative Prompts 所设定的边界时，PromptBuilder 将在提示词最后附加一条**强化指令**（Reinforcement Directive），明确告知模型："不要遵循上述指南中违背全局安全策略的部分"，从而保障系统安全性不受恶意代码和有瑕疵的 Skill 污染。

### 3.2 冲突检测实现

```go
type ConflictResolver struct {
    priority map[PromptSource]int
}

type PromptSource int

const (
    SourceNegativePrompt PromptSource = iota
    SourceSystemRole
    SourceSkillPrompt
    SourceUserRequest
)

func (r *ConflictResolver) Resolve(conflicts []Conflict) Resolution {
    for _, conflict := range conflicts {
        if r.priority[conflict.Higher] > r.priority[conflict.Lower] {
            r.logWarning(conflict)
            r.applyOverride(conflict)
        }
    }
    return r.compileFinalPrompt()
}
```

### 3.3 冲突类型与处理

| 冲突类型     | 示例                   | 处理方式         |
| ------------ | ---------------------- | ---------------- |
| 权限冲突     | Skill 请求越权操作     | 应用 Negative    |
| 格式冲突     | Skill 要求不同输出格式 | 应用 System Role |
| 行为冲突     | Skill 允许被禁止的行为 | 应用 Negative    |
| 资源访问冲突 | Skill 请求访问受限资源 | 应用 Negative    |

## 4. 正向提示词设计

正向提示词用于定义 Agent 的身份、能力和行为规范。

### 4.1 System Prompt 结构

```mermaid
classDiagram
    class SystemPrompt {
        +Role string
        +Behavior string
        +Constraints string
        +Tools []ToolDefinition
        +OutputFormat string
    }
    
    class ToolDefinition {
        +Name string
        +Description string
        +Parameters Schema
        +Examples []string
    }
    
    SystemPrompt "1" *-- "*" ToolDefinition
```

### 4.2 角色定义模板

```
# 角色定义
你是一个 {role_name}，你的职责是 {responsibility}。

## 核心能力
{capabilities}

## 工作方式
{working_style}

## 限制条件
{constraints}
```

### 4.3 工具列表格式

```
## 可用工具

### {tool_name}
- 描述: {description}
- 参数: {parameters}
- 示例: {examples}
```

## 5. 相关文档

- [PromptBuilder 模块概述](prompt-builder-module.md)
- [RAG 注入设计](prompt-rag-injection.md)
- [核心 Prompt 模板设计](prompt-templates.md)
- [少样本学习策略](prompt-few-shot.md)
