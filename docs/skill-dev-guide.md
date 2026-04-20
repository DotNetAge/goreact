# GoReAct Skill 开发指南

> Skill 是 GoReAct 中扩展 Agent 行为的专门能力模块。通过将领域知识和操作规程封装为 Skill，开发者可以让 Agent 在特定场景下展现出专业级的行为模式。本文档面向希望在 GoReAct 中开发和使用 Skill 的开发者。

## 一、Skill 在 GoReAct 中的角色

GoReAct 的设计哲学是：开发者专注于 Tools 与 Skills 的开发与运用，内核机制与性能由框架负责。在这个体系中，Tool 提供原子化的操作能力（如读取文件、执行命令），而 Skill 则提供更高层的行为编排和领域知识。

两者的关系可以理解为：Skill 定义"做什么"和"怎么做"的策略，Tool 提供"执行"的手段。在 T-A-O 循环的 Think 阶段，Skill 的指令会被注入到 Prompt 中，指导 LLM 做出更精准的推理决策。

GoReAct 的 Skill 遵循 [Agent Skills 规范](https://agentskills.io)，具备以下特性：

- 渐进式加载：三级渐进披露机制，最小化 Token 消耗
- 双来源加载：支持文件系统目录和 Go embed 内置两种方式
- 意图驱动匹配：基于关键词自动匹配用户意图，无需手动激活
- 运行时动态创建：LLM 可通过 `skill_create` 工具在运行时创建新 Skill

## 二、SKILL.md 文件格式详解

### 2.1 整体结构

SKILL.md 采用 YAML Frontmatter + Markdown Body 的格式，这是 Agent Skills 规范定义的标准格式：

```
---
[Frontmatter: YAML 格式的元数据]
---

[Body: Markdown 格式的指令内容]
```

文件以 `---` 开头，Frontmatter 部分位于两行 `---` 之间，Body 部分紧跟在第二个 `---` 之后。

### 2.2 Frontmatter 字段说明

| 字段 | 必填 | 约束 | 说明 |
|---|---|---|---|
| `name` | 是 | 1-64 字符，仅允许小写字母、数字和连字符，不能以连字符开头或结尾，不能包含连续连字符 | Skill 的唯一标识符，必须与所在目录名一致 |
| `description` | 是 | 1-1024 字符 | 描述 Skill 的功能和使用时机，直接影响匹配准确性 |
| `license` | 否 | 自由格式 | 许可证名称或引用文件路径 |
| `compatibility` | 否 | 最多 500 字符 | 环境依赖要求（运行环境、系统包、网络访问等） |
| `metadata` | 否 | key-value 映射 | 任意附加元数据 |
| `allowed-tools` | 否 | 空格分隔的工具名列表 | 预批准的工具白名单（实验性） |

关于 `name` 字段，GoReAct 的验证规则（`core.ValidateSkillName`）要求：

- 长度 1-64 字符
- 仅允许 `a-z`、`0-9`、`-`
- 不能以 `-` 开头或结尾
- 不能包含 `--`（连续连字符）

有效示例：`code-review`、`data-analysis`、`git-workflow`、`pdf-processing`
无效示例：`PDF-Processing`（含大写）、`-pdf`（连字符开头）、`pdf--processing`（连续连字符）

关于 `allowed-tools` 字段，这是 Agent Skills 规范中的实验性功能。在 SKILL.md 中声明后，目前 GoReAct 会将其存储在 `Skill.AllowedTools` 中，但框架层面尚未实现基于此字段进行工具权限过滤的逻辑。此字段可用于文档化 Skill 推荐使用的工具集合。

### 2.3 Markdown Body 编写指南

Markdown Body 是 Skill 的核心内容，当 Skill 被激活时，这段内容会完整注入到 Think Prompt 中。编写时需注意以下几点：

结构化组织。使用清晰的标题层级（H2/H3）划分不同阶段或步骤，帮助 LLM 快速定位信息。参考内置 Skill 的惯例，可以按 Phase 组织内容：

```markdown
## Phase 1: Research & Plan
1. **Analyze**: ...
2. **Plan**: ...

## Phase 2: Execute
1. **Step**: ...
2. **Step**: ...
```

具体可操作。指令要明确告诉 LLM 应该调用哪些工具、传入什么参数、期望什么结果。避免模糊的描述，用具体的命令和步骤替代：

```markdown
# 好的写法
1. 使用 'grep' 搜索项目中的错误日志
2. 使用 'read' 读取相关文件

# 不好的写法
1. 查找项目中的问题
2. 阅读相关代码
```

使用引号标注工具名称。在指令中用单引号括起工具名（如 `'grep'`、`'bash'`），使 LLM 能明确识别工具调用意图。

控制在 5000 Token 以内。Body 会被完整注入 Prompt，过长会消耗上下文空间并可能影响 LLM 的注意力。建议将详细参考资料放到 `references/` 目录下，在 Body 中引用即可。

## 三、三级渐进披露机制

GoReAct 的 Skill 采用三级渐进披露（Progressive Disclosure），以最小化 Token 消耗：

### 第一级：元数据（约 100 Token）

Reactor 启动时，所有 Skill 的 `name` 和 `description` 被加载到 `SkillRegistry` 中。这是 Skill 发现阶段，LLM 通过这两个字段判断哪些 Skill 可能与当前请求相关。此阶段所有已注册 Skill 的元数据常驻内存，不注入 Prompt。

### 第二级：指令（建议 < 5000 Token）

当用户请求进入 Think 阶段时，GoReAct 调用 `FindApplicableSkills` 进行意图匹配。匹配成功的 Skill 的 `Instructions`（即 SKILL.md 的 Markdown Body）会被注入到 Think Prompt 的 `<activated_skills>` 区域。这是按需加载——只有被激活的 Skill 才会消耗 Prompt 空间。

### 第三级：资源（按需加载）

Skill 目录下的 `scripts/`、`references/`、`assets/` 等文件不会被自动加载。LLM 在执行 Skill 指令时，可以通过 `read`、`bash` 等工具按需读取这些资源文件。这种设计允许 Skill 携带大量参考资料而不影响常规上下文消耗。

Token 预算建议：
- 元数据：每个 Skill 约 100 Token，10 个 Skill 共约 1000 Token
- 单个 Skill 指令：建议控制在 5000 Token 以内（约 3000-4000 个中文字符）
- 资源文件：不限制，但建议单个文件不超过 10000 Token

## 四、两种加载方式详解

### 4.1 文件系统加载

文件系统加载通过 `WithSkillDir` 选项配置。GoReAct 会在启动时扫描指定目录，将每个包含 SKILL.md 的子目录注册为一个 Skill。

配置方式：

```go
r := reactor.NewReactor(config,
    reactor.WithSkillDir("/path/to/skills"),
    // 可以多次调用以指定多个目录
    reactor.WithSkillDir("/another/path/to/skills"),
)
```

目录结构要求：

```
/path/to/skills/           # 由 WithSkillDir 指定的根目录
├── code-review/           # 每个子目录对应一个 Skill，目录名应与 name 字段一致
│   ├── SKILL.md           # 必须包含此文件
│   ├── scripts/           # 可选：可执行脚本
│   │   └── lint.sh
│   ├── references/        # 可选：参考文档
│   │   └── style-guide.md
│   └── assets/            # 可选：静态资源
└── deployment/
    └── SKILL.md
```

加载流程由 `core.FileSystemSkillLoader` 实现：
1. 读取根目录下的所有条目
2. 跳过非目录条目
3. 对每个子目录，尝试读取 `SKILL.md`
4. 如果不存在 SKILL.md，跳过该目录（不报错）
5. 解析 SKILL.md 的 YAML Frontmatter 和 Markdown Body
6. 验证 `name` 和 `description` 字段
7. 创建 `Skill` 结构体，`Source` 设为 `"filesystem"`，`RootDir` 设为目录的绝对路径

注意：目录不存在不会报错，返回空的 Skill 列表。单个 Skill 加载失败会中断整个加载过程并返回错误。

### 4.2 Go embed 内置加载

内置加载通过 Go 的 `embed.FS` 将 Skill 嵌入到二进制文件中，无需外部文件即可运行。GoReAct 的内置 Skill 就是通过这种方式加载的。

实现步骤：

首先，准备 Skill 文件。在项目中创建目录结构：

```
reactor/skills/
├── my-skill/
│   └── SKILL.md
└── another-skill/
    └── SKILL.md
```

然后，使用 `embed` 指令嵌入文件：

```go
//go:embed skills/*/SKILL.md
var bundledSkills embed.FS
```

最后，注册嵌入的 Skill：

```go
func RegisterBundledSkills(registry core.SkillRegistry) error {
    subFS, err := fs.Sub(bundledSkills, "skills")
    if err != nil {
        return err
    }

    loader := core.NewBundledSkillLoader(subFS, ".")
    skills, err := loader.Load()
    if err != nil {
        return err
    }

    for _, skill := range skills {
        if err := registry.RegisterSkill(skill); err != nil {
            return err
        }
    }
    return nil
}
```

内置 Skill 的 `Source` 为 `"bundled"`，`RootDir` 为空字符串（因为文件不在文件系统上）。

GoReAct 默认会加载内置 Skill。如需禁用，使用 `WithoutBundledSkills` 选项：

```go
r := reactor.NewReactor(config,
    reactor.WithoutBundledSkills(),
)
```

两种加载方式可以同时使用。`WithSkillDir` 加载的外部 Skill 会与内置 Skill 共存于同一个 `SkillRegistry` 中。如果两者有同名 Skill，后注册的会覆盖先注册的（外部 Skill 在内置 Skill 之后加载）。

## 五、Skill 匹配与激活机制

### 5.1 匹配流程

Skill 的匹配发生在 Think 阶段。当 Reactor 收到用户输入并完成意图分类后，会调用 `FindApplicableSkills` 方法查找匹配的 Skill。具体流程如下：

第一步，构建匹配文本。将当前 `Intent` 的各字段拼接为一段文本：

```go
intentText := intentType + " " + topic + " " + summary + " " + entityBlob
```

其中 `entityBlob` 是所有 Entities 的 key 和 value 拼接而成。所有文本均转为小写。

第二步，提取关键词。对拼接文本进行分词，过滤掉停用词（英文停用词表包含约 100 个常见词汇，如 "the"、"is"、"can"、"user" 等），得到关键词列表。

第三步，逐一匹配。对 `SkillRegistry` 中的每个 Skill，检查其 `name + description` 中是否包含至少一个长度 >= 3 字符的 Intent 关键词。如果命中，则该 Skill 被视为匹配。

第四步，注入 Prompt。所有匹配的 Skill 的 `Instructions` 通过 Go Template 的 `skillSection` 函数注入到 Think Prompt 中。

### 5.2 注入格式

匹配到的 Skill 在 Think Prompt 中以如下格式呈现（位于 `<rules>` 标签之后、`<intent>` 标签之前）：

```xml
<activated_skills>
## Skill: bug-hunter
[Skill Instructions 的完整 Markdown 内容]

## Skill: architect
[Skill Instructions 的完整 Markdown 内容]
</activated_skills>
```

如果没有任何 Skill 被激活，则整个 `<activated_skills>` 块不会出现在 Prompt 中，不消耗任何 Token。

### 5.3 匹配机制的设计考量

当前的匹配算法基于关键词包含检测，这是一个轻量级但有效的方案。理解其特点有助于编写更准确的 Skill：

关键词必须 >= 3 字符才会参与匹配。这意味着 "git"、"bug"、"fix" 这样的短词（<3 字符）会被忽略。在 Description 中应包含足够多的、长度 >= 3 的关键术语。

匹配是基于包含而非精确匹配。Intent 中的 "debug" 会匹配 Description 中包含 "debugging" 的 Skill（因为 "debug" 是 "debugging" 的子串）。

多个 Skill 可能同时被激活。Think Prompt 中可以包含多个 Skill 的指令，LLM 会综合所有激活的 Skill 进行推理。因此 Skill 之间不应存在矛盾的指令。

停用词会被过滤。诸如 "use"、"user"、"your" 等词在匹配时会被跳过，不要依赖这些词来触发 Skill。

### 5.4 示例

假设有如下 Skill：

```yaml
name: bug-hunter
description: >
  Expert SOP for locating, isolating and fixing complex bugs.
  Use when the user mentions bugs, errors, crashes, debugging, or fixing issues.
```

当用户输入"我的代码出了个 bug，帮我调试一下"时：
- Intent 的 type/topic/summary/entities 中可能包含"bug"、"调试"等词
- 关键词提取后，"bug" 长度 >= 3，会被保留
- "bug" 在 Skill 的 "name + description" 中存在包含关系
- Skill 被激活，Instructions 注入 Prompt

## 六、内置 Skill 开发流程

### 6.1 前置准备

内置 Skill 位于 `reactor/skills/` 目录下，通过 `embed.FS` 嵌入。开发内置 Skill 需要遵循以下流程。

### 6.2 创建 Skill 目录

在 `reactor/skills/` 下创建以 Skill 名称命名的子目录：

```bash
mkdir reactor/skills/my-skill
```

### 6.3 编写 SKILL.md

在目录中创建 SKILL.md 文件。以下是一个完整的示例：

```markdown
---
name: my-skill
description: >
  A brief description explaining what this skill does and when
  the agent should activate it. Include relevant keywords (>= 3 chars).
allowed-tools: read grep bash
---

# My Skill: Purpose and Scope

One-line summary of what this skill enables.

## Phase 1: Preparation
1. **Gather Context**: Use 'grep' to search for relevant patterns in the codebase.
2. **Read Files**: Use 'read' to examine the identified files.

## Phase 2: Analysis
1. Analyze the gathered information.
2. Identify the key patterns or issues.

## Phase 3: Execution
1. Perform the required actions.
2. Verify the results.

## Constraints
- Always use 'bash' for shell commands.
- Prefer 'read' over 'grep' when the file path is known.
```

### 6.4 更新 embed 指令

确认 `reactor/skills_bundled.go` 中的 embed 指令能够匹配新文件：

```go
//go:embed skills/*/SKILL.md
var bundledSkills embed.FS
```

由于使用的是通配符 `skills/*/SKILL.md`，新创建的 Skill 目录只要遵循 `skills/<name>/SKILL.md` 的结构，就会自动被包含，无需手动修改 embed 指令。

### 6.5 验证

编写单元测试验证 Skill 能正确加载和匹配。参考 `reactor/skill_registry_test.go` 中的测试用例：

```go
func TestDefaultSkillRegistry_FindApplicableSkills(t *testing.T) {
    r := NewSkillRegistry()

    // 注册测试 Skill
    r.RegisterSkill(&core.Skill{
        Name:        "test-skill",
        Description: "handles debugging and testing scenarios",
    })

    // 构造匹配 Intent
    intent := &Intent{
        Type:    "task",
        Topic:   "debugging",
        Summary: "fix the test failure",
    }

    skills, err := r.FindApplicableSkills(intent)
    // 验证匹配结果...
}
```

### 6.6 现有内置 Skill 参考

GoReAct 内置了以下 7 个 Skill，可供参考：

| Skill 名称 | 功能 | 推荐工具 |
|---|---|---|
| `architect` | 高层系统设计与重构编排 | glob, grep, subagent, todo-write |
| `batch` | 大规模并行变更的编排 | grep, subagent, bash |
| `bug-hunter` | 定位、隔离和修复复杂 Bug | grep, glob, bash, subagent, read |
| `simplify` | 代码质量审查和清理 | file-edit, bash, read, replace |
| `stuck` | Agent 卡住/循环时的诊断和突围 | grep, glob, bash |
| `verify` | 通过测试和执行验证变更 | bash, todo-write |
| `remember` | 项目约定和共享记忆管理 | grep, bash, read |

这些 Skill 的 SKILL.md 文件位于 `reactor/skills/` 目录下，可以直接阅读参考其编写风格和结构。

## 七、动态 Skill 创建

GoReAct 提供了 `skill_create` 工具，允许 LLM 在运行时动态创建 Skill。这是"经验自成长"特性的基础——Agent 可以将成功执行的经验保存为可复用的 Skill。

### 7.1 skill_create 工具参数

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `name` | string | 是 | Skill 唯一标识符（小写字母、数字、连字符） |
| `description` | string | 是 | Skill 功能描述和使用时机 |
| `instructions` | string | 是 | Skill 的核心指令内容（Markdown 格式） |
| `trigger` | string | 否 | 触发关键词（逗号分隔） |
| `category` | string | 否 | Skill 分类标签 |
| `save_to` | string | 否 | 保存目录路径（默认自动选择） |

### 7.2 创建流程

当 LLM 调用 `skill_create` 时，执行流程如下：

1. 验证 `name` 格式（必须以小写字母开头，仅含小写字母、数字和连字符，1-64 字符）
2. 确定保存目录（优先使用配置的 SkillDirs，其次使用 `./skills`）
3. 在目录下创建 `<name>/` 子目录
4. 如果 SKILL.md 已存在，返回错误（不允许覆盖）
5. 生成 SKILL.md 内容（包含 name、description，以及可选的 trigger、category）
6. 写入文件并返回成功信息

### 7.3 生成的 SKILL.md 格式

`skill_create` 生成的 SKILL.md 包含 `name`、`description` 两个规范字段，以及 `trigger` 和 `category` 两个扩展字段：

```markdown
---
name: my-dynamic-skill
description: Description of the skill
trigger: keyword1,keyword2,keyword3
category: development
---

[instructions content]
```

注意：`trigger` 和 `category` 是 `skill_create` 工具生成的扩展字段，不属于 Agent Skills 规范定义的标准字段。GoReAct 的标准 Frontmatter 解析器（`core.parseYamlFrontmatter`）不会解析这两个字段，但它们会保留在 SKILL.md 中，可用于文档化目的。

### 7.4 动态 Skill 的生命周期

动态创建的 Skill 会被保存到文件系统上的 Skill 目录中。但这些 Skill 在当前 Reactor 实例中不会立即生效——需要重启或重新加载 Skill 目录才能注册到 `SkillRegistry` 中。`skill_create` 的返回信息会明确提示这一点。

LLM 可以使用 `skill_list` 工具查看文件系统上的所有 Skill（包括动态创建的），以确认创建是否成功。

### 7.5 skill_list 工具

`skill_list` 工具列出 Skill 目录中的所有 Skill：

```go
// 参数
category: string // 可选，按分类过滤
```

输出格式：

```
Found 3 skill(s):

## code-review
  Description: Review code quality and suggest improvements
  Trigger: review,quality,refactor
  Category: development
  Path: /path/to/skills/code-review/SKILL.md

## ...
```

## 八、最佳实践

### 8.1 Description 编写

Description 是 Skill 匹配的唯一依据（与 Name 一起），其质量直接决定 Skill 能否被正确激活。

原则一：包含丰富的关键词。关键词长度必须 >= 3 字符才能参与匹配。在 Description 中应涵盖用户可能使用的各种表述方式：

```yaml
# 好的写法
description: >
  Expert SOP for locating, isolating and fixing complex bugs.
  Use when the user mentions bugs, errors, crashes, debugging, or fixing issues.

# 不好的写法
description: Help fix problems.
```

原则二：同时说明"做什么"和"什么时候用"。这有助于 LLM 判断是否应该激活 Skill：

```yaml
description: >
  Parallel orchestration of large-scale mechanical changes.
  Use when the user mentions batch, bulk, replace all, or migrate all.
```

原则三：控制长度在合理范围内。Description 最长 1024 字符，建议 100-300 字符即可。过长的 Description 不会显著提升匹配精度，但会增加元数据的 Token 消耗。

### 8.2 Token 预算控制

单个 Skill 的 Instructions 建议控制在 5000 Token 以内（约 3000-4000 个中文字符或 6000-8000 个英文单词）。如果内容超出这个范围，应考虑以下策略：

将详细参考资料移到 `references/` 目录，在 Instructions 中通过引用引导 LLM 按需读取：

```markdown
## Step 1: Check conventions
Read references/conventions.md for project coding standards.
```

将通用流程和关键约束保留在 Instructions 中，将示例和边界情况放到 `references/` 目录：

```markdown
## Core Steps
1. Analyze the input
2. Apply transformation
3. Validate output

See references/examples.md for detailed examples and edge cases.
```

多个 Skill 同时激活时，所有 Skill 的 Instructions 会累积在 Prompt 中。如果一次请求可能激活 3-4 个 Skill，每个 Skill 的 Instructions 应更短（建议 < 2000 Token）。

### 8.3 Skill 命名规范

命名应简洁、语义化、唯一：

- 使用小写字母、数字和连字符
- 以小写字母开头
- 使用连字符分隔单词（`code-review`，而非 `code_review` 或 `codeReview`）
- 名称应反映 Skill 的核心功能
- 避免过于宽泛的名称（如 `helper`、`util`）
- 避免与内置 Skill 重名

推荐命名示例：`code-review`、`git-workflow`、`pdf-processing`、`api-testing`、`db-migration`

### 8.4 Skill 目录结构

对于简单的 Skill，一个 SKILL.md 文件即可。当 Skill 需要携带额外资源时，推荐以下目录结构：

```
my-skill/
├── SKILL.md              # 必须有：元数据 + 核心指令
├── scripts/              # 可选：可执行脚本
│   ├── setup.sh
│   └── validate.py
├── references/           # 可选：参考文档
│   ├── guide.md          # 详细操作指南
│   └── examples.md       # 使用示例
└── assets/               # 可选：模板、数据文件
    └── template.yaml
```

资源文件的引用使用相对于 Skill 根目录的路径。LLM 通过 `read` 或 `bash` 工具访问这些文件时，需要使用完整的文件系统路径（`Skill.RootDir` + 相对路径）。

### 8.5 Skill 之间避免冲突

由于多个 Skill 可能同时激活，编写时需注意：

不要在 Instructions 中给出相互矛盾的指令。例如一个 Skill 要求"始终使用 `bash` 执行命令"，另一个要求"避免使用 `bash`"。

如果 Skill 之间有依赖关系，在 Instructions 中明确说明。例如 `verify` Skill 的指令可以引用 `batch` Skill 的工作成果。

保持每个 Skill 的职责单一。如果一个 Skill 的 Instructions 过于宽泛（既做代码审查又做部署又做测试），考虑拆分为多个独立的 Skill。

### 8.6 调试 Skill 匹配问题

如果 Skill 没有被正确激活，检查以下几点：

1. Description 中是否包含与用户输入相关的关键词（>= 3 字符）。
2. 关键词是否被停用词表过滤（如 "use"、"can"、"make" 等词会被过滤）。
3. Intent 的 type、topic、summary 中是否包含匹配的关键词。可以通过监听 EventBus 的事件来查看实际的 Intent 内容。
4. Skill 是否成功注册到 `SkillRegistry`。可以使用 `r.SkillRegistry().ListSkills()` 在启动后验证。

如果 Skill 被错误激活（误匹配），检查 Description 是否包含过于通用的词汇。考虑收窄 Description 的措辞，使其更具指向性。

### 8.7 编程式注册 Skill

除了通过目录加载和 `skill_create`，还可以在代码中直接注册 Skill：

```go
r := reactor.NewReactor(config)

// 直接注册自定义 Skill
skill := &core.Skill{
    Name:        "custom-skill",
    Description: "Custom skill for specific domain",
    Instructions: "# Custom Skill\n\nStep-by-step instructions...",
    Source:      "programmatic",
}
r.SkillRegistry().RegisterSkill(skill)
```

注意：必须在 `NewReactor` 返回之后、第一次 `Run` 之前注册。如果使用 `WithoutBundledSkills()` 禁用了内置 Skill 后又想添加部分，可以用这种方式精确控制。
