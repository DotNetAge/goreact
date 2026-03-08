package middlewares

import (
	"fmt"
	"sync"
	"time"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/types"
)

// RateLimiter 速率限制器接口
type RateLimiter interface {
	// Allow 检查是否允许请求
	Allow() bool
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	capacity int           // 桶容量
	tokens   int           // 当前令牌数
	refillRate int         // 每秒补充的令牌数
	lastRefill time.Time   // 上次补充时间
	mu       sync.Mutex
}

// NewTokenBucketLimiter 创建令牌桶限流器
func NewTokenBucketLimiter(capacity, refillRate int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许请求
func (l *TokenBucketLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 补充令牌
	now := time.Now()
	elapsed := now.Sub(l.lastRefill)
	tokensToAdd := int(elapsed.Seconds()) * l.refillRate
	if tokensToAdd > 0 {
		l.tokens += tokensToAdd
		if l.tokens > l.capacity {
			l.tokens = l.capacity
		}
		l.lastRefill = now
	}

	// 检查是否有令牌
	if l.tokens > 0 {
		l.tokens--
		return true
	}

	return false
}

// RateLimitMiddleware 速率限制中间件
func RateLimitMiddleware(limiter RateLimiter) thinker.ThinkMiddleware {
	return func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			// 检查速率限制
			if !limiter.Allow() {
				return nil, fmt.Errorf("rate limit exceeded, please try again later")
			}

			return next(task, ctx)
		}
	}
}
