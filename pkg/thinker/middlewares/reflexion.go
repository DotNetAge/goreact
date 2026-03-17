package middlewares

import (
	"fmt"

	gochatcore "github.com/DotNetAge/gochat/pkg/core"
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/types"
)

// ReflexionMiddleware 反思中间件
// 让AI在执行任务后进行自我反思，从失败中学习
func ReflexionMiddleware(llmClient gochatcore.Client, maxReflections int) thinker.ThinkMiddleware {
	return func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			var lastThought *types.Thought
			var lastErr error

			for reflectionCount := 0; reflectionCount <= maxReflections; reflectionCount++ {
				// 执行原始思考
				thought, err := next(task, ctx)
				lastThought = thought
				lastErr = err

				// 成功则返回
				if err == nil && isTaskCompleted(thought) {
					if reflectionCount > 0 {
						if thought.Metadata == nil {
							thought.Metadata = make(map[string]interface{})
						}
						thought.Metadata["reflection_count"] = reflectionCount
					}
					return thought, nil
				}

				// 最后一次反思，不再尝试
				if reflectionCount == maxReflections {
					break
				}

				// 生成反思提示
				reflectionPrompt := generateReflectionPrompt(task, thought, err)

				// 调用 LLM 进行反思
				messages := []gochatcore.Message{
					gochatcore.NewUserMessage(reflectionPrompt),
				}
				reflectionResponse, err := llmClient.Chat(ctx.Context(), messages)
				if err != nil {
					// 反思失败，继续使用原始思考
					continue
				}

				// 更新任务为带有反思的新任务
				task = fmt.Sprintf("%s\n\n反思: %s", task, reflectionResponse.Content)
			}

			return lastThought, lastErr
		}
	}
}

// isTaskCompleted 判断任务是否完成
func isTaskCompleted(thought *types.Thought) bool {
	// 简单实现：如果思考结果表示应该结束且有最终答案，则认为任务完成
	return thought != nil && thought.ShouldFinish && thought.FinalAnswer != ""
}

// generateReflectionPrompt 生成反思提示
func generateReflectionPrompt(task string, thought *types.Thought, err error) string {
	var errorInfo string
	if err != nil {
		errorInfo = fmt.Sprintf("执行错误: %v", err)
	} else if thought != nil {
		errorInfo = fmt.Sprintf("执行结果: %v", thought)
	} else {
		errorInfo = "无执行结果"
	}

	return fmt.Sprintf(`你刚刚尝试执行任务："%s"，但遇到了问题：%s。

请反思以下问题：
1. 任务失败的根本原因是什么？
2. 你本可以采取什么不同的方法？
3. 下次遇到类似任务时，你会如何改进？

请提供具体的反思和改进建议，以便重新尝试完成任务。`, task, errorInfo)
}
