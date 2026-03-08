package middlewares

import (
	"log"
	"time"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/types"
)

// Logger 日志接口
type Logger interface {
	Info(msg string, fields map[string]interface{})
	Error(msg string, err error, fields map[string]interface{})
}

// DefaultLogger 默认日志实现
type DefaultLogger struct{}

func (l *DefaultLogger) Info(msg string, fields map[string]interface{}) {
	log.Printf("[INFO] %s %v", msg, fields)
}

func (l *DefaultLogger) Error(msg string, err error, fields map[string]interface{}) {
	log.Printf("[ERROR] %s: %v %v", msg, err, fields)
}

// LoggingMiddleware 日志中间件
// 记录每次 Think 的输入、输出、耗时
func LoggingMiddleware(logger Logger) thinker.ThinkMiddleware {
	if logger == nil {
		logger = &DefaultLogger{}
	}

	return func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			startTime := time.Now()

			// 记录请求
			logger.Info("Think started", map[string]interface{}{
				"task":      task,
				"timestamp": startTime,
			})

			// 执行
			thought, err := next(task, ctx)

			// 记录响应
			duration := time.Since(startTime)
			fields := map[string]interface{}{
				"duration_ms": duration.Milliseconds(),
				"success":     err == nil,
			}

			if thought != nil {
				fields["should_finish"] = thought.ShouldFinish
				if thought.Action != nil {
					fields["action"] = thought.Action.ToolName
				}
			}

			if err != nil {
				logger.Error("Think failed", err, fields)
			} else {
				logger.Info("Think completed", fields)
			}

			return thought, err
		}
	}
}
