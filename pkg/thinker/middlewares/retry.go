package middlewares

import (
	"fmt"
	"time"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/types"
)

// RetryMiddleware 重试中间件
// 在 LLM 调用失败时自动重试
func RetryMiddleware(maxRetries int, retryInterval time.Duration) thinker.ThinkMiddleware {
	return func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			var lastErr error

			for attempt := 0; attempt <= maxRetries; attempt++ {
				thought, err := next(task, ctx)

				// 成功则返回
				if err == nil {
					// 记录重试次数
					if attempt > 0 {
						if thought.Metadata == nil {
							thought.Metadata = make(map[string]interface{})
						}
						thought.Metadata["retry_attempts"] = attempt
					}
					return thought, nil
				}

				lastErr = err

				// 最后一次尝试失败，不再重试
				if attempt == maxRetries {
					break
				}

				// 等待后重试
				time.Sleep(retryInterval)
			}

			return nil, fmt.Errorf("think failed after %d retries: %w", maxRetries, lastErr)
		}
	}
}
