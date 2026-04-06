# 什么是 ReAct？

大语言模型（LLM）的快速发展，展现了其在自然语言理解和生成方面的惊人能力。然而，仅依靠模型内部参数存储的静态知识，在处理复杂任务时往往显得捉襟见肘，尤其是在需要与外部环境交互、获取实时信息或执行长期规划时。为了解决这一问题，**ReAct（Reasoning and Acting）** 应运而生。

## 定义

ReAct 是由普林斯顿大学和 Google Research 的研究人员在 2022 年底提出的一个通用范式。它通过将大语言模型的**推理能力（Reasoning）**与**行动能力（Acting）**交织在一起，使其能够像人类一样，在解决复杂问题时，不仅能够思考，还能主动寻找外部信息或执行相关操作。

- **Reasoning (推理):** 指模型生成思考过程（Thoughts），用于规划、跟踪当前状态、分解任务或处理异常。
- **Acting (行动):** 指模型生成具体的动作指令（Actions），通过工具或 API 与外部环境（如维基百科、数据库、网络搜索）交互，并获取观察结果（Observations）。

## 核心直觉

人类的智能行为通常不仅包含对抽象概念的思考，还包含与物理或数字环境的互动。例如，当你在厨房做一道从未做过的菜时：

1.  **推理:** “我需要查一下食谱。”
2.  **行动:** 打开手机，搜索“红烧肉的做法”。
3.  **观察:** 看到食谱说需要“生抽和老抽”。
4.  **推理:** “我家好像没有老抽了，我得去超市买。”
5.  **行动:** 出门去超市。

ReAct 正是受到这种认知过程的启发。在 ReAct 提出之前，大语言模型的应用通常将这两者孤立开来：

*   **仅推理（Reasoning-Only）:** 如思维链（Chain-of-Thought, CoT）[1]，模型通过逐步推理来得出结论，但其思考过程是一个封闭系统，无法利用外部知识，容易产生“幻觉（Hallucination）”。
*   **仅行动（Acting-Only）:** 模型被训练来预测下一步的动作，但缺乏高层规划和状态跟踪机制，难以处理长序列任务。

ReAct 的核心创新在于：**通过文本空间（Text Space）实现推理和行动的协同。** 推理可以帮助模型决定下一步采取什么行动（如判断是否需要搜索新信息），而行动获取的观察结果又可以作为新的上下文，更新模型的推理状态。

## 范式的意义

引入 ReAct 范式，为构建智能体（Agents）带来了以下关键优势：

*   **直观且易于设计:** 人类专家可以很容易地通过构建包含 Thought、Action 和 Observation 的少样本提示（Few-shot prompts）来指导模型。
*   **强大的泛化能力:** 无需大量的微调，仅依靠上下文学习（In-context Learning），模型就能在跨领域任务中表现出色。
*   **高可解释性:** 模型的每一步思考和行动都以文本形式透明地展现，开发者可以清晰地追踪错误来源并进行干预。
*   **减少幻觉:** 通过强制模型基于外部观察（Observation）进行推理，有效降低了捏造事实的概率。

## 参考文献

[1] Wei, J., Wang, X., Schuurmans, D., Bosma, M., Xia, F., Chi, E., Le, Q. V., & Zhou, D. (2022). Chain-of-Thought Prompting Elicits Reasoning in Large Language Models. *Advances in Neural Information Processing Systems (NeurIPS)*.
[2] Yao, S., Zhao, J., Yu, D., Du, N., Narasimhan, I., Karthik, V., & Narasimhan, K. (2022). ReAct: Synergizing Reasoning and Acting in Language Models. *International Conference on Learning Representations (ICLR 2023)*.
