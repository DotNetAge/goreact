# GoReAct Architecture (GoReAct 架构深度指南)

## 1. 核心理念：管线即驱动 (Pipeline as Driving Force)

GoReAct 的核心不再是一个简单的 LLM 封装，而是一个**动态生成的逻辑执行引擎**。

### 1.1 编程语言化管线 (Universal Pipeline)
我们将 `Pipeline` 视为一种具备编程逻辑的执行序列。通过引入核心控制流原语，管线具备了万能逻辑：
- **Sequence (顺序)**: 默认的任务流。
- **Branch (分支 - `IfStep`)**: 基于上下文观察结果的实时决策。
- **Iteration (循环 - `LoopStep`)**: 带有物理熔断（MaxLoops）的重复执行能力。

## 2. Thinker 的进化：从推理到架构 (Thinker as Architect)

`Thinker` 不仅是决策中心，更是 **任务编译器 (Task Compiler)**。

### 2.1 任务编译流程
1. **意图获取**：用户输入或暗语（如 `/plan`）。
2. **逻辑生成**：LLM 生成包含 "If", "Step", "Repeat" 关键词的结构化计划。
3. **语义解析**：`parser.ParsePlan` 将文本编译为逻辑任务流。
4. **驱动注入**：将任务流注入 `PipelineContext` 的计划队列中。
5. **任务切换**：自动将 `ctx.Input` 路由到当前计划步骤，开启后续的 ReAct 执行。

## 3. Skill 的终极形态：Pipeline Blueprint

一个 `Skill`（能力）本质上是一个 **预置的管线蓝图**。
- 当 Agent 激活一个 Skill 时，它实际上是加载了一套包含 `If/Loop` 的 `Nested Pipeline`。
- 这确保了 Agent 在执行复杂 SOP 时具备极高的稳定性和自纠错能力。

## 4. 提示词工程工具箱 (Prompt Toolkit)

作为语言中枢，它通过以下方式支持上述架构：
- **Fluent API**: 精准构建指令。
- **Context Compression**: 通过滑动窗口管理长程管线的 Token 溢出。
- **Few-Shot Injector**: 确保 LLM 生成的逻辑计划符合 `Parser` 的编译器规范。
