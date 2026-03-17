package middlewares

import (
	"fmt"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/types"
)

// UserProfile 用户画像
type UserProfile struct {
	UserID      string                 // 用户 ID
	Name        string                 // 用户名
	Preferences map[string]interface{} // 用户偏好
	History     []string               // 历史交互
	Metadata    map[string]interface{} // 其他元数据
}

// Summary 生成用户画像摘要
func (p *UserProfile) Summary() string {
	summary := fmt.Sprintf("User: %s", p.Name)
	if len(p.Preferences) > 0 {
		summary += fmt.Sprintf(", Preferences: %v", p.Preferences)
	}
	return summary
}

// UserProfileStore 用户画像存储接口
type UserProfileStore interface {
	// Get 获取用户画像
	Get(userID string) (*UserProfile, error)
}

// UserProfileMiddleware 用户画像中间件
// 加载用户画像并增强任务上下文
func UserProfileMiddleware(store UserProfileStore) thinker.ThinkMiddleware {
	return func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			// 获取用户 ID
			userID, ok := ctx.Get("user_id")
			if !ok {
				// 没有用户 ID，跳过
				return next(task, ctx)
			}

			// 加载用户画像
			profile, err := store.Get(userID.(string))
			if err != nil {
				// 加载失败不影响主流程
				ctx.Set("user_profile_error", err.Error())
				return next(task, ctx)
			}

			// 将用户画像注入 context
			ctx.Set("user_profile", profile)
			ctx.Set("user_preferences", profile.Preferences)

			// 增强任务描述
			enhancedTask := fmt.Sprintf("%s\n\nUser context: %s", task, profile.Summary())

			return next(enhancedTask, ctx)
		}
	}
}
