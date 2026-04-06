# 技能 (Skills)：Agent 的领域专家能力封装

在早期的 AI Agent 研究（如基础 ReAct 框架）中，模型主要通过“工具调用 (Tool Calling)”来与外部世界交互。然而，随着 Agent 在复杂工程任务中的深入应用，单纯的原子工具（如 `read_file`, `grep_search`）已不足以支撑复杂的业务逻辑。

**技能 (Skills)** 规范的引入，定义了一种标准化的方式来打包和共享 AI Agent 的专业能力。

## 技能 (Skill) vs 工具 (Tool)

虽然两者都赋予了 Agent 外部能力，但其设计哲学和执行粒度有本质区别：

| 特性         | 工具 (Tool / Atomic Action)       | 技能 (Skill / Packaged Expertise)        |
| :----------- | :-------------------------------- | :--------------------------------------- |
| **定义**     | 单一的、确定性的 API 或函数调用。 | 模块化的、包含指令、代码和数据的能力包。 |
| **结构**     | 通常是代码中的一个函数定义。      | 具有标准目录结构的便携式文件夹。         |
| **逻辑**     | 输入 A，执行 B，返回 C。          | 包含推理逻辑、反思、重试和多步 SOP。     |
| **发现机制** | 始终存在于工具列表中。            | 仅暴露元数据，按需“激活”完整指令。       |

##  标准化目录结构

根据 [Agent Skills Specification](https://agentskills.io/specification)，一个标准的技能是一个独立的目录，通常包含以下组件：

*   **`SKILL.md` (核心):** 必须位于根目录。包含 YAML 元数据（Frontmatter）和 Markdown 指令。
*   **`scripts/` (可选):** 包含 Python、Bash 或 JS 脚本，供 Agent 执行以完成具体任务。
*   **`references/` (可选):** 存放补充文档、模板或领域特定的数据集。
*   **`assets/` (可选):** 静态资源，如图片、Schema 定义或配置文件。

## 3. 元数据与声明 (YAML Frontmatter)

`SKILL.md` 的顶部必须包含 YAML 块，用于定义技能的身份和触发条件：

```yaml
name: code-review-expert
description: 用于审查 Python 代码的安全性、性能和风格指南。
license: MIT
compatibility:
  os: [darwin, linux]
allowed-tools: [read_file, grep_search, run_shell_command]
```

*   **`name`:** 唯一标识符，必须与文件夹名称一致。
*   **`description`:** 至关重要。Agent 通过阅读描述来决定是否需要“激活”该技能。

## 核心执行原理：渐进式披露 (Progressive Disclosure)

为了保护有限的上下文窗口（Context Window），技能遵循三级加载策略：

1.  **发现阶段 (Discovery):** 系统仅加载所有技能的 `name` 和 `description`。此时 Token 消耗极低。
2.  **激活阶段 (Activation):** 当 Agent 判断当前任务需要某项专业能力时，调用 `activate_skill(name)`。系统此时将 `SKILL.md` 的**完整正文（指令部分）**读入上下文。
3.  **按需执行 (On-Demand Execution):** 只有当 Agent 在执行过程中明确需要时，才会读取 `references/` 中的文档或执行 `scripts/` 中的代码。

## Hook 系统与事件驱动

高级技能规范支持 **Hooks (钩子)** 机制，允许技能拦截和修改 Agent 的行为：

*   **Pre-Tool / Post-Tool Hooks:** 在原子工具运行前后执行。例如，一个 `security-guard` 技能可以在 `run_shell_command` 执行前检查是否有危险指令。
*   **事件驱动:** 技能可以监听 `SessionStart`、`UserPromptSubmit` 或 `Stop` 等事件，从而在特定时机自动介入，无需 LLM 显式调用。

## 为什么这种范式至关重要？

1.  **便携性 (Portability):** 技能是独立于运行时的文件夹，可以轻松地在不同项目或不同 Agent 框架之间迁移。
2.  **上下文效率:** 避免了“指令过载”，确保 Agent 始终专注于当前任务最相关的专业知识。
3.  **低代码扩展:** 领域专家只需编写 `SKILL.md` 中的自然语言指南，即可赋予 Agent 复杂的专业能力，而无需深厚的编程背景。

## 参考资料
*   [Agent Skills Specification (Official)](https://agentskills.io/specification)
*   [Claude Code Skills Documentation](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code) (注：此为相关生态参考)
