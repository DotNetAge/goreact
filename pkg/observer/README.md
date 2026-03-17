# Observer (观察者 / 感官)

`Observer` 位于 GoReAct 架构（Thinker - Actor - Observer - Terminator）的第三环。在人类的认知循环中，当我们“思考(Think)”并“行动(Act)”后，需要通过眼睛和耳朵来“观察(Observe)”世界发生了什么改变。

在 Agent 引擎中，Observer 就是连接物理世界反馈与大模型逻辑推理的“感知接口”。它负责接收 Actor 的执行结果（无论是成功的数据，还是抛出的异常/Panic），并将其清洗、提炼、格式化，最终转化为大模型能够轻易理解的上下文（Context）。

---

## 核心职责 (Core Responsibilities)

Observer 不能仅仅是做 `fmt.Sprintf("%v", result)`，因为外部环境的反馈往往是嘈杂、冗长且不可预测的（例如：一个数十兆的 HTML 源码，或者几万行的日志错误栈）。因此，一个健壮的 Observer 需具备以下四大能力：

### 1. 结果解析与降噪 (Result Parsing & Denoising)
大模型对 Token 是极度敏感的。Observer 需要将冗杂的原始数据提纯：
- **数据提取：** 将 Actor 返回的庞大 JSON/XML 或网页源码进行解析，仅提取 Thinker 所需的核心字段（如：仅保留网页的文本正文，去除 `<script>` 和 `<style>` 标签）。
- **截断与摘要：** 当 API 返回的数据量超过了预设的 Token 窗口上限时，Observer 需要自动执行截断（Truncation）或者调用一个轻量级的小模型对结果进行“摘要聚合（Summarization）”，再交还给主干网络。

### 2. 异常翻译与语义化 (Error Translation & Semanticization)
当执行发生错误时，原始的系统报错（如 `connection refused`、`index out of bounds` 或 `schema validation failed`）对大模型来说可能是冰冷或难以从中学习的：
- **语义化错误反馈：** Observer 需要捕捉这些底层报错，并将其包装为自然语言。例如，将 `401 Unauthorized` 包装为：“Observer 反馈：调用天气 API 失败，原因是 API Key 无效或未授权，请尝试检查你的凭证配置或使用其他工具。”，以激发 Thinker 更好的反思（Reflexion）能力。

### 3. 多模态感官 (Multimodal Perception)
随着大模型能力的进化，Observation 不再局限于纯文本：
- **视觉/听觉接入：** 如果 Actor 截取了一张网页的截图，或录制了一段音频，Observer 负责将这些多模态数据（图片 Base64、音频流）按特定的协议（如 OpenAI 的 Vision 格式）嵌入到 Context 中，供 Thinker 下一步观察。

### 4. 状态校验与审计记录 (State Validation & Audit Logging)
- **断言与契约校验：** 验证 Actor 的输出是否符合预期的 Schema，如果返回的数据结构残缺，Observer 可以直接标记此次 Action 为 `Failed`，避免脏数据污染 Thinker 的后续判断。
- **可观测性探针：** 详细记录该步骤中 Actor 花费的时间、实际产生的 Token 消耗，并将这段“动作 -> 观察”的快照写入全局的 Pipeline Context 和 Tracing 系统中。

---

## 架构集成位置 (Integration in ReAct Loop)

在一次典型的 GoReAct 执行循环中，Observer 的位置如下：

1. **[Actor]** 执行工具逻辑（例如发起一次对 Google 的搜索请求）。
2. **[Actor]** 将执行完毕的原始 HttpResponse 字节流传递给 Observer。
3. **[Observer]** 解析该字节流。提取搜索结果的 Title 和 Snippet，丢弃无用的 HTML 结构。若结果集多达 100 条，则截断保留前 10 条以节省 Token。
4. **[Observer]** 将处理好的、高度浓缩的纯文本（或结构化 JSON）附加到全局 Pipeline Context 中。
5. **[Terminator]** 根据当前的 Context（包括最新的 Observation）来判定任务是否已完成。若未完成，Context 将流转回 **[Thinker]** 供下一轮思考使用。

## 设计指引 (Design Guidelines)

- **管道模式 (Pipeline)：** 建议将 Observer 设计为一条处理流（Chain）。原始数据进来后，依次经过 `Sanitizer (清洗)` -> `Truncator (截断)` -> `Semanticizer (语义化)`，最终输出给大模型。
- **动态阈值配置：** 截断与摘要的阈值（如 Max Tokens）应当可以通过配置动态调整，以适配不同上下文窗口大小的 LLM（例如给 Claude 3 分配较大的保留窗口，给 Llama 3 8B 分配较小的窗口）。