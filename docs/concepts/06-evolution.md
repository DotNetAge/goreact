# 范式演进：从 ReAct 到 Reflexion 与 Self-Reflection

ReAct 作为一个开创性的框架，证明了交织推理与行动的巨大潜力。然而，基础版本的 ReAct 依然存在一些局限性，特别是在面对长期复杂任务时的试错成本和缺乏自我反思能力。这促使了学术界在 ReAct 基础上进行了进一步的演进。

## 基础 ReAct 的局限性

*   **易陷入死循环 (Repetitive Loops):** 如果模型在一个步骤中持续获得无用的 Observation（例如，用错误的关键字反复搜索），基础的 ReAct 可能无法意识到自己陷入了死胡同，从而耗尽所有步骤配额（Max Steps）。
*   **缺乏长期经验积累:** 在一个任务（Episode）中犯的错误，模型无法记忆下来并在未来的类似任务中避免。它每次都是“从头开始”。

为了解决这些问题，研究者将人类认知科学中的**元认知（Metacognition）**——即对自身思考过程的思考——引入到大模型中。

## Reflexion：基于语言的强化学习

Reflexion [1] 框架由 Shinn 等人提出，是对 ReAct 的重要升级。它的核心思想是：**让智能体在失败后进行自我反思，并将反思的结果转化为文本形式的“记忆”，用于指导下一次尝试。**

### Reflexion 的运作流程

1.  **Actor (执行者):** 通常就是一个 ReAct Agent，它执行任务并生成一条轨迹 (Trajectory)。
2.  **Evaluator (评估者):** 判断任务是否成功（例如，通过规则匹配或让另一个 LLM 充当裁判）。
3.  **Self-Reflection (自我反思模型):** 如果任务失败，该模型会审视刚才失败的轨迹，找出错误原因，并生成一条**改进建议（Heuristic）**。
4.  **Memory (记忆库):** 改进建议被存入短期或长期记忆。
5.  **重试 (Retry):** 在下一次尝试时，先前的反思建议会被加入到 Actor 的上下文 Prompt 中，指导其避免同样的错误。

*反思示例：* “在之前的尝试中，我试图一次性搜索两个实体导致没有结果。下次我应该分别搜索每一个实体。”

这种机制被称为“基于语言的强化学习”（Linguistic Reinforcement Learning），它用文本形式的梯度（梯度的文字描述）代替了传统强化学习中的标量奖励（Scalar Rewards），大大降低了模型优化的成本。

## Plan-and-Solve 架构

另一个演进方向是对 ReAct 中零散的 Thought 进行整合。基础 ReAct 倾向于“走一步看一步”。

Wang 等人提出的 Plan-and-Solve [2] 及其变体，主张在采取任何具体 Action 之前，先让模型进行宏观的全局规划（Global Planning）。

*   **Plan:** 模型首先输出一个完整的多步计划列表。
*   **Execute:** 随后按照计划列表，依次执行 ReAct 的 T-A-O 循环。
*   **Re-plan:** 当执行遇到意外时，触发重新规划。

这种架构分离了“规划器（Planner）”和“执行器（Executor）”，在复杂工程任务和代码编写代理（Coding Agents）中应用广泛。

## 演进的本质

从 ReAct 到 Reflexion，再到更复杂的 Agent 架构（如 AutoGPT, BabyAGI），其技术演进的本质可以总结为：

1.  **从短期上下文到长期记忆 (Long-term Memory) 的引入。**
2.  **从单一模型循环到多代理/多角色协同 (Multi-Agent Collaboration)。**
3.  **从单纯的外部行动 (Acting on Environment) 扩展到对自身内部状态的管理 (Acting on Memory/Plans)。**

这些演进使得 Agent 的行为越来越接近一个能够进行试错学习的高级系统。

## 参考文献
[1] Shinn, N., Cassano, F., Gopinath, A., Narasimhan, K., & Yao, S. (2023). Reflexion: Language Agents with Verbal Reinforcement Learning. *NeurIPS*.
[2] Wang, L., Xu, W., Lan, Y., Hu, Z., Lan, Y., Lee, R. K.-W., & Lim, E.-P. (2023). Plan-and-Solve Prompting: Improving Zero-Shot Chain-of-Thought Reasoning by Large Language Models. *ACL 2023*.